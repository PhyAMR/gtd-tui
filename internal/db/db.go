package db

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	TasksBucket    = []byte("tasks")
	EventsBucket   = []byte("events")
	HistoryBucket  = []byte("history")
	SettingsBucket = []byte("settings")
)

type Database struct {
	db *bolt.DB
}

type Task struct {
	ID            string    `json:"id"`
	Description   string    `json:"description"`
	ClassName     string    `json:"class_name"`
	ReviewCount   int       `json:"review_count"`
	Interval      int       `json:"interval"`
	EventID       string    `json:"event_id"`
	Status        string    `json:"status"` // pending, done, skipped
	CreatedAt     time.Time `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	NextReviewAt  time.Time `json:"next_review_at"`
}

type PomodoroLog struct {
	ID          string    `json:"id"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Type        string    `json:"type"` // work, short_break, long_break
	Interrupted bool      `json:"interrupted"`
}

func New(path string) (*Database, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{TasksBucket, EventsBucket, HistoryBucket, SettingsBucket} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) SaveTask(task *Task) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(TasksBucket)
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		return b.Put([]byte(task.ID), data)
	})
}

func (d *Database) GetTask(id string) (*Task, error) {
	var task Task
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(TasksBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("task not found")
		}
		return json.Unmarshal(data, &task)
	})
	return &task, err
}

func (d *Database) GetAllTasks() ([]*Task, error) {
	var tasks []*Task
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(TasksBucket)
		return b.ForEach(func(k, v []byte) error {
			var task Task
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			tasks = append(tasks, &task)
			return nil
		})
	})
	return tasks, err
}

func (d *Database) GetTasksByClass(className string) ([]*Task, error) {
	var tasks []*Task
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(TasksBucket)
		return b.ForEach(func(k, v []byte) error {
			var task Task
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			if task.ClassName == className {
				tasks = append(tasks, &task)
			}
			return nil
		})
	})
	return tasks, err
}

func (d *Database) GetTasksDueToday() ([]*Task, error) {
	var tasks []*Task
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(TasksBucket)
		return b.ForEach(func(k, v []byte) error {
			var task Task
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			if task.Status == "pending" && 
			   task.NextReviewAt.After(today) && 
			   task.NextReviewAt.Before(tomorrow) {
				tasks = append(tasks, &task)
			}
			return nil
		})
	})
	return tasks, err
}

func (d *Database) DeleteTask(id string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(TasksBucket)
		return b.Delete([]byte(id))
	})
}

func (d *Database) SavePomodoroLog(log *PomodoroLog) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(HistoryBucket)
		data, err := json.Marshal(log)
		if err != nil {
			return err
		}
		return b.Put([]byte(log.ID), data)
	})
}

func (d *Database) GetRecentPomodoros(count int) ([]*PomodoroLog, error) {
	var logs []*PomodoroLog
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(HistoryBucket)
		c := b.Cursor()

		for k, v := c.Last(); k != nil && len(logs) < count; k, v = c.Prev() {
			var log PomodoroLog
			if err := json.Unmarshal(v, &log); err != nil {
				continue
			}
			logs = append(logs, &log)
		}
		return nil
	})
	return logs, err
}

func (d *Database) SaveEventMapping(eventID, taskID string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(EventsBucket)
		return b.Put([]byte(eventID), []byte(taskID))
	})
}

func (d *Database) GetEventMapping(eventID string) (string, error) {
	var taskID string
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(EventsBucket)
		data := b.Get([]byte(eventID))
		if data == nil {
			return fmt.Errorf("mapping not found")
		}
		taskID = string(data)
		return nil
	})
	return taskID, err
}
