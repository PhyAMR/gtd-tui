package db

import (
	"os"
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	tmpfile := "/tmp/gtd_test.db"
	defer os.Remove(tmpfile)

	db, err := New(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	t.Run("SaveAndGetTask", func(t *testing.T) {
		task := &Task{
			ID:           "test-1",
			Description:  "Test task",
			ClassName:    "TestClass",
			ReviewCount:  0,
			Interval:     1,
			Status:       "pending",
			CreatedAt:    time.Now(),
			NextReviewAt: time.Now().Add(24 * time.Hour),
		}

		if err := db.SaveTask(task); err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}

		retrieved, err := db.GetTask("test-1")
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}

		if retrieved.Description != task.Description {
			t.Errorf("Retrieved description = %q, want %q", retrieved.Description, task.Description)
		}
	})

	t.Run("GetTasksByClass", func(t *testing.T) {
		task := &Task{
			ID:           "test-2",
			Description:  "Another task",
			ClassName:    "TestClass",
			ReviewCount:  0,
			Interval:     1,
			Status:       "pending",
			CreatedAt:    time.Now(),
			NextReviewAt: time.Now().Add(24 * time.Hour),
		}

		db.SaveTask(task)

		tasks, err := db.GetTasksByClass("TestClass")
		if err != nil {
			t.Fatalf("Failed to get tasks by class: %v", err)
		}

		if len(tasks) < 2 {
			t.Errorf("Expected at least 2 tasks, got %d", len(tasks))
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		if err := db.DeleteTask("test-1"); err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		_, err := db.GetTask("test-1")
		if err == nil {
			t.Error("Expected error when getting deleted task")
		}
	})
}
