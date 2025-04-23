package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/keyboard/inline"
)

// handle all non-command messages
func messageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	hasSession, err := userHasSession(chatID)
	if err != nil {
		log.Println("Error: could not check if user has session: ", err)
		return
	}

	if !hasSession {
		sendInfo(ctx, b, update)
		return
	}

	session, err := getSession(chatID)
	msg := update.Message.Text

	if err != nil {
		log.Println("Error: could not get session: ", err)
		return
	}

	switch session.Command {
	case "none":
		sendInfo(ctx, b, update)
		break
	case "start":
		switch session.Step {
		case 0: // story: user was asked where from
			if err := updateLastForm(chatID, FormUpdate{DeparturePoint: strPtr(msg)}); err != nil {
				log.Print("Error: start:0 could not update last form", err)
				return
			}
			updateSession(chatID, SessionUpdate{Step: intPtr(1)})
			sendMessage(ctx, b, update, "Выберите пункт назначения.")
			break
		case 1: // story: user was asked where to
			if err := updateLastForm(chatID, FormUpdate{ArrivalPoint: strPtr(msg)}); err != nil {
				log.Print("Error: start:1 could not update last form", err)
				return
			}
			form, err := getLastForm(chatID)
			if err != nil {
				log.Print("Error: start:1 could not get last(current) form", err)
				return
			}

			sendMessage(ctx, b, update, fmt.Sprintf("Маршрут выбран:\n%s -> %s", form.DeparturePoint, form.ArrivalPoint))
			updateSession(chatID, SessionUpdate{Command: strPtr("none"), Step: intPtr(0)})
			break
		default:
			log.Println("Error: unknown session step state")
			return
		}
		break
	default:
		log.Println("Error: unknown command state")
		return
	}
}

// when user typed `/start`
func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	hasSession, err := userHasSession(chatID)
	if err != nil {
		log.Print("Error: could not check if user has session: ", err)
		return
	}

	if !hasSession {
		if err := createSession(chatID); err != nil {
			log.Print("Error: could not create new session: ", err)
			return
		}
	}

	session, err := getSession(chatID)
	if err != nil {
		log.Println("Error: could not get session: ", err)
		return
	}

	if session.Command != "none" {
		sendResposeIsInvalid(ctx, b, update)
	} else {
		updateSession(chatID, SessionUpdate{Command: strPtr("start"), Step: intPtr(0)})
		insertEmptyForm(chatID)

		sendMessage(ctx, b, update, "Откуда вы хотите отправиться?")
	}
}

// button

func initInlineKeyboard(b *bot.Bot) {

}

func handlerInlineKeyboard(ctx context.Context, b *bot.Bot, update *models.Update) {
	demoInlineKeyboard := inline.New(b, inline.NoDeleteAfterClick()).
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
