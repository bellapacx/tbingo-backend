package main

import (
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func startTelegramBot() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		user := update.Message.From

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🎉 Welcome to TBingo!\nSend your phone number to join.")
				bot.Send(msg)

				// Send keyboard to request contact
				contactBtn := tgbotapi.NewKeyboardButtonContact("📱 Share Phone Number")
				keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{contactBtn})
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Tap the button below to share your phone number:")
				msg.ReplyMarkup = keyboard
				bot.Send(msg)
			}
		}

		// Handle contact
		if update.Message.Contact != nil {
			phone := update.Message.Contact.PhoneNumber
			chatID := update.Message.Chat.ID

			msg := tgbotapi.NewMessage(chatID, "✅ Phone number received: "+phone+"\nPlease wait while we join you to the Bingo round.")

			bot.Send(msg)

			// Call your backend /join
			// You can make HTTP POST to localhost:8080/join or your Render URL here
			go joinBingoServer(phone, strconv.FormatInt(chatID, 10))
		}
	}
}
