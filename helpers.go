package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/datepicker"
	"github.com/go-telegram/ui/keyboard/inline"
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
