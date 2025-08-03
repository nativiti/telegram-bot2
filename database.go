package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Reminder struct {
	UserID   int64  `json:"user_id"`
	Category string `json:"category"`
	Date     string `json:"date"`
}

func InitDB() *bolt.DB {
	db, err := bolt.Open("reminders.db", 0600, nil)
	if err != nil {
		log.Fatalf("Eroare DB: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("reminders"))
		return err
	})
	if err != nil {
		log.Fatalf("Eroare creare bucket: %v", err)
	}
	return db
}

func ListReminders(db *bolt.DB, userID int64) ([]Reminder, error) {
	var reminders []Reminder

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("reminders"))
		return b.ForEach(func(k, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if r.UserID == userID {
				reminders = append(reminders, r)
			}
			return nil
		})
	})
	return reminders, err
}

func AddReminder(db *bolt.DB, userID int64, category, date string) error {
	id := fmt.Sprintf("%d-%d", userID, time.Now().UnixNano())
	reminder := Reminder{
		UserID:   userID,
		Category: category,
		Date:     date,
	}

	data, err := json.Marshal(reminder)
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("reminders"))
		return b.Put([]byte(id), data)
	})
}
