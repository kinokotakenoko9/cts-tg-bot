package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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
	// CORE LOGIC HERE
	switch session.Command {
	case "none":
		sendInfo(ctx, b, update)
	case "start":
		switch session.Step {
		case 0: // line: user was asked where from

			// show availible cities
			foundCities := getCitiesWithPrefix(msg)
			if len(foundCities) == 0 {
				sendMessage(ctx, b, update, "Ничего не найдено. Попробуйте ещё раз.")
				break
			}
			if len(foundCities) >= 6 {
				sendMessage(ctx, b, update, "Слишком много результатов. Попробуйте ещё раз.")
				break
			}

			sendButtonList(ctx, b, update, foundCities, fmt.Sprintf("Результаты для \"%s\":", msg), func(ctx context.Context, bot *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
				if err := updateLastForm(chatID, FormUpdate{DeparturePoint: strPtr(string(data))}); err != nil {
					log.Print("Error: start:0 could not update last form", err)
					return
				}
				sendMessage(ctx, b, update, "Выберите пункт назначения.")
				updateSession(chatID, SessionUpdate{Step: intPtr(1)}) // next session step
			})
		case 1: // line: user was asked where to

			// show availible cities
			foundCities := getCitiesWithPrefix(msg)
			if len(foundCities) == 0 {
				sendMessage(ctx, b, update, "Ничего не найдено. Попробуйте ещё раз.")
				break
			}
			if len(foundCities) >= 6 {
				sendMessage(ctx, b, update, "Слишком много результатов. Попробуйте ещё раз.")
				break
			}

			form, err := getLastForm(chatID)
			if err != nil {
				log.Print("Error: start:1 could not get last(current) form", err)
				return
			}

			foundCities = remove(foundCities, form.DeparturePoint)

			sendButtonList(ctx, b, update, foundCities, fmt.Sprintf("Результаты для \"%s\":", msg), func(ctx context.Context, bot *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
				if err := updateLastForm(chatID, FormUpdate{ArrivalPoint: strPtr(string(data))}); err != nil {
					log.Print("Error: start:1 could not update last form", err)
					return
				}

				sendMessage(ctx, b, update, fmt.Sprintf("Маршрут выбран:\n%s -> %s", form.DeparturePoint, string(data)))
				updateSession(chatID, SessionUpdate{Command: strPtr("none"), Step: intPtr(0)}) // next session step
			})
		default:
			log.Println("Error: unknown session step state")
			return
		}
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
