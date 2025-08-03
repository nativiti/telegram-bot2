package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var pendingDate = make(map[int64]string)

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

	// 🔹 Pornim serverul web în paralel
	var wg sync.WaitGroup
	wg.Add(1)
	go startWebServer()

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

		// 🔹 HELP
		if text == "/help" || text == "help" {
			helpMsg := `📖 Funcții disponibile:

				/help - Afișează acest mesaj
				/status - Vezi dacă botul este activ și câte remindere ai
				/list - Listează reminderele salvate și zilele rămase
				/remove <categorie> - Șterge un reminder după categorie
				Programat de CHATGPT indrumat de DODE513 firt git-up and down`
			bot.Send(tgbotapi.NewMessage(userID, helpMsg))
			continue
		}

		// 🔹 STATUS
		if text == "/status" {
			total, nearest, err := GetStatusInfo(db, userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "❌ Eroare la citirea statusului"))
				continue
			}

			msg := fmt.Sprintf("🤖 Bot activ\n📦 Remindere salvate: %d", total)
			if nearest != "" {
				msg += fmt.Sprintf("\n⏳ Următoarea expirare: %s", nearest)
			} else {
				msg += "\nℹ️ Nicio expirare setată"
			}

			bot.Send(tgbotapi.NewMessage(userID, msg))
			continue
		}

		// 🔹 LIST
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

		// 🔹 REMOVE
		if len(text) > 8 && text[:8] == "/remove " {
			category := text[8:]
			err := RemoveReminder(db, userID, category)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "❌ Eroare la ștergere sau reminder inexistent"))
			} else {
				bot.Send(tgbotapi.NewMessage(userID, fmt.Sprintf("🗑️ Reminder '%s' a fost șters", category)))
			}
			continue
		}

		// 🔹 Dacă așteptăm categoria după dată
		if date, ok := pendingDate[userID]; ok {
			err := AddReminder(db, userID, text, date)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "❌ Eroare la salvare"))
				continue
			}
			delete(pendingDate, userID)
			bot.Send(tgbotapi.NewMessage(userID, fmt.Sprintf("✅ Am salvat %s pentru data %s", text, date)))
			continue
		}

		// 🔹 Dacă mesajul este o dată
		if dateRegex.MatchString(text) {
			_, err := time.Parse("02-01-2006", text)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(userID, "❌ Format invalid. Folosește DD-MM-YYYY"))
				continue
			}
			pendingDate[userID] = text
			bot.Send(tgbotapi.NewMessage(userID, "ℹ️ Ce este această dată? (ex: ITP, Asigurare, Rovinietă)"))
			continue
		}

		// 🔹 Mesaj implicit
		bot.Send(tgbotapi.NewMessage(userID, "📅 Trimite o dată (DD-MM-YYYY) pentru a crea un reminder"))
	}

	wg.Wait()
}

func startWebServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "✅ Telegram bot running")
	})
	log.Println("🌍 Web server pornit pe portul 8080")
	http.ListenAndServe(":8080", nil)
}
