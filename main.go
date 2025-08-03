package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var pendingDate = make(map[int64]string) // stocăm date în așteptare per user

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		fmt.Println("❌ BOT_TOKEN nu este setat")
		return
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("❌ Eroare la conectarea cu Telegram: %v", err)
	}

	log.Printf("✅ Bot pornit: %s", bot.Self.UserName)

	db := InitDB()
	defer db.Close()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	dateRegex := regexp.MustCompile(`^\d{2}-\d{2}-\d{4}$`)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.Chat.ID
		text := update.Message.Text

		// Dacă așteptăm categoria după dată
		if date, ok := pendingDate[userID]; ok {
			err := AddReminder(db, userID, text, date)
			if err != nil {
				msg := tgbotapi.NewMessage(userID, "❌ Eroare la salvare")
				bot.Send(msg)
				continue
			}
			delete(pendingDate, userID)
			msg := tgbotapi.NewMessage(userID, fmt.Sprintf("✅ Am salvat %s pentru data %s", text, date))
			bot.Send(msg)
			continue
		}
		if text == "/list" {
			reminders, err := ListReminders(db, userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "❌ Eroare la citirea remindere-lor"))
				continue
			}

			if len(reminders) == 0 {
				bot.Send(tgbotapi.NewMessage(userID, "ℹ️ Nu ai remindere salvate"))
				continue
			}

			msgText := "📋 Remindere salvate:\n"
			for _, r := range reminders {
				expDate, err := time.Parse("02-01-2006", r.Date)
				if err != nil {
					msgText += fmt.Sprintf("- %s (%s) - ❌ Data invalidă\n", r.Category, r.Date)
					continue
				}

				zileRamase := int(time.Until(expDate).Hours() / 24)
				if zileRamase < 0 {
					msgText += fmt.Sprintf("- %s (%s) - ⚠️ Expirat acum %d zile\n", r.Category, r.Date, -zileRamase)
				} else {
					msgText += fmt.Sprintf("- %s (%s) - %d zile rămase\n", r.Category, r.Date, zileRamase)
				}
			}
			bot.Send(tgbotapi.NewMessage(userID, msgText))
			continue
		}
		// Dacă mesajul este o dată
		if dateRegex.MatchString(text) {
			// validăm data
			_, err := time.Parse("02-01-2006", text)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "❌ Format invalid. Folosește DD-MM-YYYY"))
				continue
			}
			pendingDate[userID] = text
			bot.Send(tgbotapi.NewMessage(userID, "ℹ️ Ce este această dată? (ex: ITP, Asigurare, Rovinietă)"))
		} else {
			bot.Send(tgbotapi.NewMessage(userID, "📅 Trimite o dată (DD-MM-YYYY) pentru a crea un reminder"))
		}
	}
}
