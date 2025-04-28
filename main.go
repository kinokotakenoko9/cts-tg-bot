package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/go-telegram/bot"
)

// key: chatID, value: Session in json. invariant: all but last form are complete, the last form is complete or not complete
var db *badger.DB

// availible cities. TODO: could be parsed
var cities map[string]string

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

	// init cities map
	err = loadCities()
	if err != nil {
		log.Println("Error: loading cities: ", err)
		return
	}

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

		log.Println("Error: could not create bot: ", err)
		return
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "list", bot.MatchTypeCommand, listHandler)

	b.Start(ctx)
}
