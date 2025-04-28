package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v4"
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

func sendButtonList(ctx context.Context, b *bot.Bot, update *models.Update, names []string, text string, onSelect inline.OnSelect) {
	citiesInlineKeyboard := inline.New(b) // TODO: bug here, remove inactive keyboards

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
func bytesToIntsPtr(b []byte) *[]int {
	ints := make([]int, len(b))
	for i, v := range b {
		ints[i] = int(v)
	}
	return &ints
}
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

// interacting with database <start>
func getDBKey(chatID int64) []byte {
	return []byte(fmt.Sprintf("%d", chatID))
}

// ---- sessions ----
func userHasSession(chatID int64) (bool, error) {
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func createSession(chatID int64) error {
	key := getDBKey(chatID)
	session := Session{
		Step:    0,
		Command: "none",
		Forms:   []Form{},
	}

	jsn, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: marshaling new session: ", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsn)
	})
	if err != nil {
		log.Println("Error: could not store new session in DB: ", err)
		return err
	}

	return nil
}

func getSession(chatID int64) (Session, error) {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		log.Println("Error: could not read session from DB while getting session: ", err)
		return Session{}, err
	}

	return session, nil
}

func updateSession(chatID int64, update SessionUpdate) error {
	key := getDBKey(chatID)
	var session Session

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})
	if err != nil {
		log.Println("Error: could not read session from DB while updating session: ", err)
		return err
	}

	if update.Step != nil {
		session.Step = *update.Step
	}
	if update.Command != nil {
		session.Command = *update.Command
	}

	data, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: could not marshal updated session:", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
	if err != nil {
		log.Println("Error: updating session in DB: ", err)
		return err
	}

	return nil
}

// ---- forms ----

// inserts empty form in user session. must have a session, or will cause error
func insertEmptyForm(chatID int64) error {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				log.Println("Error(db): user does not have a session while crearting empty form: ", err)
			}

			log.Println("Error(db): could not get item by key while crearting empty form: ", err)
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})
	if err != nil && err != badger.ErrKeyNotFound {
		log.Println("Error: could not view user database: ", err)
		return err
	}

	session.Forms = append(session.Forms, Form{})

	jsn, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: could not unmarshall session with empty form: ", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsn)
	})
	if err != nil {
		log.Println("Error: updating db while creating empty form: ", err)
		return err
	}

	return nil
}

func updateLastForm(chatID int64, update FormUpdate) error {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})
	if err != nil {
		log.Println("Error: could not get current session: ", err)
		return err
	}

	if len(session.Forms) == 0 {
		log.Println("Error: no forms in session.")
		return fmt.Errorf("no forms in session")
	}

	form := &session.Forms[len(session.Forms)-1]

	if update.DeparturePoint != nil {
		form.DeparturePoint = *update.DeparturePoint
	}
	if update.ArrivalPoint != nil {
		form.ArrivalPoint = *update.ArrivalPoint
	}
	if update.DepartureDate != nil {
		form.DepartureDate = *update.DepartureDate
	}
	if update.CarriageType != nil {
		form.CarriageType = *update.CarriageType
	}
	if update.NumberOfPassengers != nil {
		form.NumberOfPassengers = *update.NumberOfPassengers
	}
	if update.CompartmentNumber != nil {
		form.CompartmentNumber = *update.CompartmentNumber
	}
	if update.ShelfType != nil {
		form.ShelfType = *update.ShelfType
	}
	if update.NumberOfPassengersTopShefl != nil {
		form.NumberOfPassengersTopShefl = *update.NumberOfPassengersTopShefl
	}
	if update.NumberOfPassengersBottomShefl != nil {
		form.NumberOfPassengersBottomShefl = *update.NumberOfPassengersBottomShefl
	}
	if update.TrackPriceChange != nil {
		form.TrackPriceChange = *update.TrackPriceChange
	}
	if update.SuggestSimilarSeats != nil {
		form.SuggestSimilarSeats = *update.SuggestSimilarSeats
	}

	jsn, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: failed to marshal updated session: ", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsn)
	})
	if err != nil {
		log.Println("Error: failed to update session in db: ", err)
		return err
	}

	return nil
}

func getLastForm(chatID int64) (Form, error) {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		log.Println("Error: could not read session from DB while getting last form: ", err)
		return Form{}, err
	}

	if len(session.Forms) == 0 {
		log.Println("Error: session has no forms")
		return Form{}, fmt.Errorf("no forms in session")
	}

	return session.Forms[len(session.Forms)-1], nil
}

// <end>
