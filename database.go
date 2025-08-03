package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Reminder struct {
	UserID   int64  `json:"user_id"`
	Category string `json:"category"`
	Date     string `json:"date"`
}

func getDBPath() string {
	if runtime.GOOS == "windows" {
		return "reminders.db" // folosit local
	}
	return "/root/data/reminders.db" // folosit în Docker/Linux
}

func InitDB() *bolt.DB {
	path := getDBPath()
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("reminders"))
		return err
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("✅ DB Path:", path)
	return db
}

func AddReminder(db *bolt.DB, userID int64, category, date string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("reminders"))
		key := fmt.Sprintf("%d_%s", userID, category)
		data, _ := json.Marshal(Reminder{UserID: userID, Category: category, Date: date})
		return b.Put([]byte(key), data)
	})
}

func ListReminders(db *bolt.DB, userID int64) ([]Reminder, error) {
	var reminders []Reminder
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("reminders"))
		return b.ForEach(func(_, v []byte) error {
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

func RemoveReminder(db *bolt.DB, userID int64, category string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("reminders"))
		key := fmt.Sprintf("%d_%s", userID, category)
		if b.Get([]byte(key)) == nil {
			return fmt.Errorf("Reminder inexistent")
		}
		return b.Delete([]byte(key))
	})
}

func GetStatusInfo(db *bolt.DB, userID int64) (int, string, error) {
	count := 0
	nearestDate := ""
	var nearestTime time.Time

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("reminders"))
		return b.ForEach(func(_, v []byte) error {
			var r Reminder
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if r.UserID == userID {
				count++
				expDate, err := time.Parse("02-01-2006", r.Date)
				if err == nil {
					if nearestTime.IsZero() || expDate.Before(nearestTime) {
						nearestTime = expDate
						nearestDate = r.Date
					}
				}
			}
			return nil
		})
	})
	return count, nearestDate, err
}
