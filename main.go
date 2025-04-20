package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/keyboard/inline"
)

// Represents user data form
type Form struct {
	ID                            int
	DeparturePoint                string
	ArrivalPoint                  string
	DepartureDate                 time.Time
	CarriageType                  string
	NumberOfPassengers            int    // invariant: 1..6
	CompartmentNumber             int    // invariant: 1..9
	ShelfType                     string // invariant: one of "any", "top", "bottom"
	NumberOfPassengersTopShefl    int    // invariant: <= NumberOfPassengers
	NumberOfPassengersBottomShefl int    // invariant: <= NumberOfPassengers
	TrackPriceChange              bool
	SuggestSimilarSeats           bool
	IsComplete                    bool // shows if user finished form or some fields arent filled
}

type Session struct {
	Step    int
	Command string
	Form    *Form
}

type Survey interface { // interface for surveys
}

var demoInlineKeyboard *inline.Keyboard

var db *badger.DB

func main() {
	var err error

	// init context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// init database
	db, err = badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Println("Error: could not open db: ", err)
		return
	}
	defer db.Close()

	opts := []bot.Option{
		bot.WithDefaultHandler(messageHandler),
	}

	// read bot token
	token, err := os.ReadFile("./token.txt")
	if err != nil {
		log.Println("Error: could not read token bot: ", err)
		return
	}

	b, err := bot.New(string(token), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		log.Println("Error: could not create bot: ", err)
		return
	}

	initInlineKeyboard(b)

	// my handlers for app itself
	b.RegisterHandler(bot.HandlerTypeMessageText, "survey", bot.MatchTypeCommand, surveyHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "shoot", bot.MatchTypeCommand, shootHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "btn", bot.MatchTypeCommand, handlerInlineKeyboard)
	// my handlers end

	b.Start(ctx)
}

// handle all non-command messages
func messageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func surveyHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(fmt.Sprintf("form:%d:0", update.Message.Chat.ID)), []byte("42"))
		return err
	})
	if err != nil {
		log.Println("Error: could not update database: ", err)
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "survey here",
	})

}

func shootHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(fmt.Sprintf("form:%d:", update.Message.Chat.ID))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				fmt.Printf("key=%s, value=%s\n", k, v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Error: could not view database", err)
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "shoot here",
		ParseMode: models.ParseModeMarkdown,
	})
}

// button

func initInlineKeyboard(b *bot.Bot) {
	demoInlineKeyboard = inline.New(b, inline.WithPrefix("inline")).
		Row().
		Button("Row 1, Btn 1", []byte("1-1"), onInlineKeyboardSelect).
		Button("Row 1, Btn 2", []byte("1-2"), onInlineKeyboardSelect).
		Row().
		Button("Row 2, Btn 1", []byte("2-1"), onInlineKeyboardSelect).
		Button("Row 2, Btn 2", []byte("2-2"), onInlineKeyboardSelect).
		Button("Row 2, Btn 3", []byte("2-3"), onInlineKeyboardSelect).
		Row().
		Button("Row 3, Btn 1", []byte("3-1"), onInlineKeyboardSelect).
		Button("Row 3, Btn 2", []byte("3-2"), onInlineKeyboardSelect).
		Button("Row 3, Btn 3", []byte("3-3"), onInlineKeyboardSelect).
		Button("Row 3, Btn 4", []byte("3-4"), onInlineKeyboardSelect).
		Row().
		Button("Cancel", []byte("cancel"), onInlineKeyboardSelect)
}

func handlerInlineKeyboard(ctx context.Context, b *bot.Bot, update *models.Update) {

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Select the variant",
		ReplyMarkup: demoInlineKeyboard,
	})
}

func onInlineKeyboardSelect(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: mes.Message.Chat.ID,
		Text:   "You selected: " + string(data),
	})
}
