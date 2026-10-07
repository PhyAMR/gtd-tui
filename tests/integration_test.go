//go:build integration
// +build integration

package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phuniverse/gtd/internal/calendar"
	"github.com/phuniverse/gtd/internal/config"
	"github.com/phuniverse/gtd/internal/db"
	"github.com/phuniverse/gtd/internal/scheduler"
)

func TestIntegrationFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DBPath:          filepath.Join(tmpDir, "test.db"),
		TokenPath:       filepath.Join(tmpDir, "tokens.json"),
		CredentialsPath: os.Getenv("GTD_TEST_CREDENTIALS"),
		ClassCalendarID: os.Getenv("GTD_TEST_CLASS_CAL"),
		StudyCalendarID: os.Getenv("GTD_TEST_STUDY_CAL"),
		ReviewRatio:     0.75,
	}

	if cfg.CredentialsPath == "" {
		t.Skip("GTD_TEST_CREDENTIALS not set, skipping integration test")
	}

	// Initialize components
	database, err := db.New(cfg.DBPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	calClient, err := calendar.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create calendar client: %v", err)
	}

	sched := scheduler.New(database, calClient, cfg)

	// Test: Add a task
	t.Run("AddTask", func(t *testing.T) {
		err := sched.AddTask("Test task", "TestClass", false)
		if err != nil {
			t.Errorf("AddTask failed: %v", err)
		}

		tasks, err := database.GetTasksByClass("TestClass")
		if err != nil {
			t.Errorf("GetTasksByClass failed: %v", err)
		}

		if len(tasks) == 0 {
			t.Error("Expected task to be saved")
		}
	})

	// Test: Mark task complete
	t.Run("MarkTaskComplete", func(t *testing.T) {
		tasks, _ := database.GetTasksByClass("TestClass")
		if len(tasks) == 0 {
			t.Skip("No tasks to mark complete")
		}

		task := tasks[0]
		err := sched.MarkTaskComplete(task.ID, 3)
		if err != nil {
			t.Errorf("MarkTaskComplete failed: %v", err)
		}

		updated, err := database.GetTask(task.ID)
		if err != nil {
			t.Errorf("GetTask failed: %v", err)
		}

		if updated.ReviewCount != 1 {
			t.Errorf("ReviewCount = %d, want 1", updated.ReviewCount)
		}

		if updated.Interval != 3 {
			t.Errorf("Interval = %d, want 3", updated.Interval)
		}
	})

	// Test: Calendar sync
	t.Run("CalendarSync", func(t *testing.T) {
		today := time.Now()
		start := today.Truncate(24 * time.Hour)
		end := start.Add(24 * time.Hour)

		events, err := calClient.GetStudyEvents(start, end)
		if err != nil {
			t.Errorf("GetStudyEvents failed: %v", err)
		}

		t.Logf("Found %d study events", len(events))
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		tasks, _ := database.GetTasksByClass("TestClass")
		for _, task := range tasks {
			database.DeleteTask(task.ID)
		}
	})
}

func TestIntegrationCalendarOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	credPath := os.Getenv("GTD_TEST_CREDENTIALS")
	if credPath == "" {
		t.Skip("GTD_TEST_CREDENTIALS not set")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		TokenPath:       filepath.Join(tmpDir, "tokens.json"),
		CredentialsPath: credPath,
		ClassCalendarID: os.Getenv("GTD_TEST_CLASS_CAL"),
		StudyCalendarID: os.Getenv("GTD_TEST_STUDY_CAL"),
	}

	calClient, err := calendar.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create calendar client: %v", err)
	}

	t.Run("CreateAndDeleteEvent", func(t *testing.T) {
		event := &calendar.Event{
			Summary:     "Test Event",
			Description: "Integration test event",
			Start:       time.Now().Add(1 * time.Hour),
			End:         time.Now().Add(2 * time.Hour),
		}

		eventID, err := calClient.CreateEvent(cfg.StudyCalendarID, event)
		if err != nil {
			t.Fatalf("CreateEvent failed: %v", err)
		}

		defer func() {
			if err := calClient.DeleteEvent(cfg.StudyCalendarID, eventID); err != nil {
				t.Errorf("DeleteEvent failed: %v", err)
			}
		}()

		if eventID == "" {
			t.Error("Expected non-empty event ID")
		}
	})
}
