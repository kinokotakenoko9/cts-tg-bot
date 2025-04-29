package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/datepicker"
	"github.com/go-telegram/ui/keyboard/inline"
	"golang.org/x/net/html"
)

// helpers

// msgs

func sendMessage(ctx context.Context, b *bot.Bot, update *models.Update, msg string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   msg,
	})
}

func sendResposeIsInvalid(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendMessage(ctx, b, update, "Invalid message")
}

func sendInfo(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendMessage(ctx, b, update, "Type /start to start") // TODO: better msg
}

func sendFormSaved(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendMessage(ctx, b, update, "Ваш запрос сохранён! 🚆Я уведомлю вас, как только появятся билеты, соответствующие вашим параметрам.\n\nДля просмотра списка отслеживаемых билетов используйте /list.")
}

func sendNoForms(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendMessage(ctx, b, update, "Нет форм. зарегистрируйте форму через /start.") // TODO: better msg
}

func sendButtonList(ctx context.Context, b *bot.Bot, update *models.Update, names []string, text string, onSelect inline.OnSelect) {
	citiesInlineKeyboard := inline.New(b, inline.NoDeleteAfterClick()) // TODO: bug here, remove inactive keyboards

	for _, name := range names {
		citiesInlineKeyboard.Row().Button(name, []byte(name), onSelect)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ReplyMarkup: citiesInlineKeyboard,
	})
}

func sendDatePicker(ctx context.Context, b *bot.Bot, update *models.Update, text string, onSelect datepicker.OnSelectHandler) {
	kb := datepicker.New(b, onSelect)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        text,
		ReplyMarkup: kb,
	})
}

// misc
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func stringToCompartmentNumber(s string) ([]int, bool) {
	parts := strings.Fields(s)
	if len(parts) == 0 || len(parts) > 9 {
		return nil, false
	}

	seen := make(map[int]bool)
	ints := make([]int, len(parts))

	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 9 || seen[n] {
			return nil, false
		}
		seen[n] = true
		ints[i] = n
	}
	return ints, true
}
func compartmentNumberToString(compartmentNumber []int) string {
	s := []string{}
	for _, n := range compartmentNumber {
		s = append(s, strconv.Itoa(n))
	}
	return strings.Join(s, " ")
}

func remove[T comparable](l []T, item T) []T {
	out := make([]T, 0)
	for _, element := range l {
		if element != item {
			out = append(out, element)
		}
	}
	return out
}

func loadCities() error {
	data, err := os.ReadFile("cities.json")
	if err != nil {
		log.Println("Error: reading file: ", err)
		return err
	}

	err = json.Unmarshal(data, &cities)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	return nil
}

func getCitiesWithPrefix(prefix string) []string {
	var result []string
	lowerPrefix := strings.ToLower(prefix)
	for city, _ := range cities {
		if strings.HasPrefix(strings.ToLower(city), lowerPrefix) {
			result = append(result, city)
		}
	}
	return result
}

// monitoring forms

var monitoringCancelFuncs = make(map[int]context.CancelFunc)
var monitoringWaitGroup sync.WaitGroup

func startMonitoring(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64, form Form) {
	ctxm, cancel := context.WithCancel(context.Background())
	monitoringCancelFuncs[form.ID] = cancel
	monitoringWaitGroup.Add(1)
	doc, err := fetchHTML(form)
	if err != nil {
		log.Println("Error: getting html")
		return
	}
	initFormState, err := getFromState(doc, form)
	if err != nil {
		log.Print("Error: getting form state 1: ", err)
		return
	}
	go monitorForm(ctxm, ctx, b, update, chatID, form, initFormState)

}

func stopMonitoring(formID int) {
	if cancelFunc, ok := monitoringCancelFuncs[formID]; ok {
		fmt.Printf("Stopping monitoring for form %d...\n", formID)
		cancelFunc()
		delete(monitoringCancelFuncs, formID)
	} else {
		fmt.Printf("No active monitoring found for form %d\n", formID)
	}
}

func fetchHTML(form Form) (*html.Node, error) {
	resp, err := http.Post("https://grandtrain.ru/local/components/oscompany/train.select/ajax.php", "application/x-www-form-urlencoded", strings.NewReader(getFormUrlParams(form)))
	log.Printf("Params: %s", getFormUrlParams(form))
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return nil, err
	}

	return doc, nil
}

func getFormUrlParams(form Form) string {
	return fmt.Sprintf("from=%s&to=%s&forward_date=%s&backward_date=&multimodal=0pagestyle=tav&timeout=10", cities[form.DeparturePoint], cities[form.ArrivalPoint], form.DepartureDate.Format("02.01.2006"))
}

type FormState struct {
	Price string
}

// getTextContent extracts the text content of an HTML node and its children.
func getTextContent(n *html.Node) string {
	var text string
	if n.Type == html.TextNode {
		text += n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += getTextContent(c)
	}
	return strings.TrimSpace(text)
}

// getFromState extracts the price string for the current date from the parsed HTML.
func getFromState(doc *html.Node, form Form) (FormState, error) {
	departureDate := form.DepartureDate.Format("2006-01-02")
	var formState FormState
	var foundPrice bool

	// Function to recursively traverse the HTML nodes.
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			// Check if this is the <a> element we're interested in.
			thisDate, ok := getAttributeValue(n, "data-thisdate")
			if ok && thisDate == departureDate {
				// Found the correct <a> tag.
				priceSpan := findChildWithAttribute(n, "span", "data-table", "Плац")
				if priceSpan != nil {
					formState.Price = getTextContent(priceSpan) // Get the price string directly
					foundPrice = true
					return // Stop traversing once we find the price.
				}
			}
		}
		// Continue traversing the children of the current node.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
			if foundPrice {
				return // Stop traversing if we've found the price.
			}
		}
	}

	traverse(doc) // Start the traversal from the root of the document.

	if !foundPrice {
		return FormState{}, fmt.Errorf("price not found for departure date: %s", departureDate)
	}

	return formState, nil
}

// getAttributeValue retrieves the value of a specific attribute from an HTML node.
func getAttributeValue(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

// findChildWithAttribute finds the first child of a node with a specific attribute value.
func findChildWithAttribute(n *html.Node, tag, attrKey, attrValue string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			val, ok := getAttributeValue(c, attrKey)
			if ok && val == attrValue {
				return c
			}
		}
	}
	return nil
}

func monitorForm(ctxm, ctx context.Context, b *bot.Bot, update *models.Update, chatID int64, form Form, initialFormState FormState) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	defer monitoringWaitGroup.Done()

	log.Printf("Monitoring started for form %d (chat %d)", form.ID, chatID)

	for {
		select {
		case <-ticker.C:
			doc, err := fetchHTML(form)
			if err != nil {
				log.Printf("Error fetching for form %d (chat %d): %v", form.ID, chatID, err)
				continue
			}

			updated := false

			newFormState, err := getFromState(doc, form)
			if err != nil {
				log.Println("Error: getting form state: ", err)
				return
			}

			if newFormState.Price != initialFormState.Price {
				updated = true
				initialFormState.Price = newFormState.Price
			}

			if updated {
				log.Printf("Update detected on form %d (%d)!", form.ID, chatID)
				sendMessage(ctx, b, update, "Изменение")
			}

		case <-ctxm.Done():
			fmt.Printf("Monitoring stopped for form %d (chat %d)\n", form.ID, chatID)
			return
		}
	}
}
