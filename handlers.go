package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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

			// sending DeparturePoint
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

			sendButtonList(ctx, b, update, foundCities, fmt.Sprintf("Результаты для \"%s\":", msg), func(ctx context.Context, _ *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
				if err := updateLastForm(chatID, FormUpdate{DeparturePoint: strPtr(string(data))}); err != nil {
					log.Print("Error: start:0 could not update last form", err)
					return
				}
				sendMessage(ctx, b, update, "Выберите пункт назначения.")
				updateSession(chatID, SessionUpdate{Step: intPtr(1)}) // next session step
			})
		case 1: // line: user was asked where to

			// sending ArrivalPoint
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

			sendButtonList(ctx, b, update, foundCities, fmt.Sprintf("Результаты для \"%s\":", msg), func(ctx context.Context, _ *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
				if err := updateLastForm(chatID, FormUpdate{ArrivalPoint: strPtr(string(data))}); err != nil {
					log.Print("Error: start:1,0 could not update last form", err)
					return
				}

				// sending DepartureDate
				sendMessage(ctx, b, update, fmt.Sprintf("Маршрут выбран:\n%s -> %s", form.DeparturePoint, string(data)))
				sendDatePicker(ctx, b, update, "Выберите дату отправления.", func(ctx context.Context, _ *bot.Bot, mes models.MaybeInaccessibleMessage, date time.Time) {
					d := date.Format("2006-01-02")

					if err := updateLastForm(chatID, FormUpdate{DepartureDate: &date}); err != nil {
						log.Print("Error: start:1.1 could not update last form", err)
						return
					}

					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: chatID,
						Text:   "Вы выбрали:  " + d,
					})

					// sending CarriageType
					sendButtonList(ctx, b, update, []string{"Любой", "Плацкарт", "Купе"}, "Какой тип вагона вас устроит?", func(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
						if err := updateLastForm(chatID, FormUpdate{CarriageType: strPtr(string(data))}); err != nil {
							log.Print("Error: start:1.2 could not update last form", err)
							return
						}

						sendMessage(ctx, b, update, "Сколько пассажиров?\n(Введите число от 1 до 6)")
						updateSession(chatID, SessionUpdate{Step: intPtr(2)}) // next session step

					})

				})

			})
		case 2: // line: user replied with amount of passengers

			numberOfPassengers, err := strconv.Atoi(msg)
			if err != nil || !(numberOfPassengers >= 1 && numberOfPassengers <= 6) {
				sendMessage(ctx, b, update, "(Введите число от 1 до 6)")
				return
			}

			if err := updateLastForm(chatID, FormUpdate{NumberOfPassengers: intPtr(numberOfPassengers)}); err != nil {
				log.Print("Error: start:2 could not update last form", err)
				return
			}

			// sending CompartmentNumber
			sendButtonList(ctx, b, update, []string{"Любой", "Не боковой", "Выбрать"}, "Какой отсек мест?", func(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
				compartmentNumber := []int{1, 2, 3, 4, 5, 6, 8, 9}
				switch string(data) {
				case "Любой":
					compartmentNumber = []int{1, 2, 3, 4, 5, 6, 8, 9}
				case "Не боковой":
					compartmentNumber = []int{2, 3, 4, 5, 6, 8}
				case "Выбрать":
					sendMessage(ctx, b, update, "Перечислите отсек(и) через пробел (1-9)")
					updateSession(chatID, SessionUpdate{Step: intPtr(3)}) // next session step
					return
				default:
					log.Println("Error: unknown CompartmentNumber state")
					return
				}

				if err := updateLastForm(chatID, FormUpdate{CompartmentNumber: &compartmentNumber}); err != nil {
					log.Print("Error: start:2.0 could not update last form", err)
					return
				}

				// sending ShelfType
				sendShelfTypeHandler(ctx, b, update, chatID)

			})

		case 3: // user chose CompartmentNumber

			parsedCompartmentNumber, isValid := stringToCompartmentNumber(msg)
			if !isValid {
				log.Println(parsedCompartmentNumber)
				sendMessage(ctx, b, update, "Перечислите отсек(и) через пробел (1-9)")
				return
			}

			if err := updateLastForm(chatID, FormUpdate{CompartmentNumber: &parsedCompartmentNumber}); err != nil {
				log.Print("Error: start:3.0 could not update last form", err)
				return
			}

			// sending  ShelfType
			sendShelfTypeHandler(ctx, b, update, chatID)

		case 4: // user sent number for bottom shelf
			form, err := getLastForm(chatID)
			if err != nil {
				log.Print("Error: start:4 could not get last(current) form", err)
				return
			}

			numberOfPassengersBottomShefl, err := strconv.Atoi(msg)
			if err != nil || !(numberOfPassengersBottomShefl >= 0 && numberOfPassengersBottomShefl <= 6) {
				sendMessage(ctx, b, update, fmt.Sprintf("(Введите число от 0 до %d)", form.NumberOfPassengers))
				return
			}

			if err := updateLastForm(chatID, FormUpdate{NumberOfPassengersBottomShefl: &numberOfPassengersBottomShefl}); err != nil {
				log.Print("Error: start:4.1 could not update last form", err)
				return
			}

			if err := updateLastForm(chatID, FormUpdate{NumberOfPassengersTopShefl: intPtr(int(form.NumberOfPassengers - numberOfPassengersBottomShefl))}); err != nil {
				log.Print("Error: start:4.2 could not update last form", err)
				return
			}

			sendMessage(ctx, b, update, fmt.Sprintf("Нижние полки: %d\nВерхние полки: %d", numberOfPassengersBottomShefl, form.NumberOfPassengers-numberOfPassengersBottomShefl))
			sendTrackPriceChangeHandler(ctx, b, update, chatID)

		case 5: // TODO

			updateSession(chatID, SessionUpdate{Command: strPtr("none"), Step: intPtr(0)}) // next session step
		default:
			log.Println("Error: unknown session step state")
			return
		}
	default:
		log.Println("Error: unknown command state")
		return
	}
}

func sendShelfTypeHandler(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64) {
	sendButtonList(ctx, b, update, []string{"Любое", "Указать нижние", "Указать верхние"}, "Какое размещение вас устроит?", func(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
		if err := updateLastForm(chatID, FormUpdate{ShelfType: strPtr(string(data))}); err != nil {
			log.Print("Error: start:sendShelfTypeHandler could not update last form", err)
			return
		}

		form, err := getLastForm(chatID)
		if err != nil {
			log.Print("Error: start:sendShelfTypeHandler could not get last(current) form", err)
			return
		}

		if string(data) != "Любое" {
			sendMessage(ctx, b, update, fmt.Sprintf("Укажите количество пассажиров для нижней полки:\n(Введите число от 0 до %d)", form.NumberOfPassengers))
			updateSession(chatID, SessionUpdate{Step: intPtr(4)}) // next session step
			return
		}

		sendTrackPriceChangeHandler(ctx, b, update, chatID)

	})
}

func sendTrackPriceChangeHandler(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64) {
	sendButtonList(ctx, b, update, []string{"Да", "Нет"}, "Отслеживать изменение цены?", func(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
		trackPriceChange := false
		if string(data) == "Да" {
			trackPriceChange = true
		}

		if err := updateLastForm(chatID, FormUpdate{TrackPriceChange: &trackPriceChange}); err != nil {
			log.Print("Error: start:trackPriceChange could not update last form", err)
			return
		}

		sendSuggestSimilarSeatsHandler(ctx, b, update, chatID)
	})
}

func sendSuggestSimilarSeatsHandler(ctx context.Context, b *bot.Bot, update *models.Update, chatID int64) {
	sendButtonList(ctx, b, update, []string{"Да", "Нет"}, "Предлагать похожие места?", func(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
		suggestSimilarSeats := false
		if string(data) == "Да" {
			suggestSimilarSeats = true
		}

		if err := updateLastForm(chatID, FormUpdate{SuggestSimilarSeats: &suggestSimilarSeats}); err != nil {
			log.Print("Error: start:suggestSimilarSeats could not update last form", err)
			return
		}

		form, err := getLastForm(chatID)
		if err != nil {
			log.Print("Error: start:suggestSimilarSeats could not get last(current) form", err)
			return
		}

		sendFormSaved(ctx, b, update)
		updateSession(chatID, SessionUpdate{Command: strPtr("none"), Step: intPtr(0)}) // next session step
		startMonitoring(ctx, b, update, chatID, form)
	})
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

// when user typed `/list`
func listHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	hasSession, err := userHasSession(chatID)
	if err != nil {
		log.Print("Error: could not check if user has session: ", err)
		return
	}

	if !hasSession {
		sendNoForms(ctx, b, update)
		return
	}

	session, err := getSession(chatID)
	if err != nil {
		log.Println("Error: could not get session: ", err)
		return
	}

	if session.Command != "none" {
		sendResposeIsInvalid(ctx, b, update)
	} else {

		sendMessage(ctx, b, update, "Список всех отслеживаемых форм:")
		for _, form := range session.Forms {
			seats := ""
			if form.ShelfType == "Любое" {
				seats = "Любые места"
			} else {
				seats = fmt.Sprintf("Нижних полок: %d\nВерхних полок: %d", form.NumberOfPassengersBottomShefl, form.NumberOfPassengersTopShefl)
			}
			formOptions := []string{}
			if form.TrackPriceChange {
				formOptions = append(formOptions, "Отслеживать цену")
			}
			if form.SuggestSimilarSeats {
				formOptions = append(formOptions, "Предлагать похожие места")
			} else {
				formOptions = append(formOptions, "Только выбранные места")
			}

			sendMessage(ctx, b, update, fmt.Sprintf("Отслеживаемый маршрут: \n%s → %s\nДата: %s\nТип Вагона: %s\nКоличество Пассажиров: %d\nОтсек: %s\n%s\n%s\nНомер формы: %d", form.DeparturePoint, form.ArrivalPoint, fmt.Sprintf("%d %d %d", form.DepartureDate.Day(), form.DepartureDate.Month(), form.DepartureDate.Year()), form.CarriageType, form.NumberOfPassengers, compartmentNumberToString(form.CompartmentNumber), seats, strings.Join(formOptions, ",\n"), form.ID))
		}
	}
}

// when user typed `/status`
func statusHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	hasSession, err := userHasSession(chatID)
	if err != nil {
		log.Print("Error: could not check if user has session: ", err)
		return
	}

	if !hasSession {
		sendNoForms(ctx, b, update)
		return
	}

	session, err := getSession(chatID)
	if err != nil {
		log.Println("Error: could not get session: ", err)
		return
	}

	if session.Command != "none" {
		sendResposeIsInvalid(ctx, b, update)
	} else {

		// sendMessage(ctx, b, update, "Список всех отслеживаемых форм:")
		for _, formStatus := range session.FormsStatus {

			sendMessage(ctx, b, update, fmt.Sprintf("Билеты на %s: \nПлацкарт: %s", formStatus.Date.Format("02 01 2006"), formStatus.Price))
		}
	}
}
