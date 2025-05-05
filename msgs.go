package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/datepicker"
	"github.com/go-telegram/ui/keyboard/inline"
)

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
	sendMessage(ctx, b, update, "Введите команду /start, чтобы начать.") // TODO: better msg?
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
