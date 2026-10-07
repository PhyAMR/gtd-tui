package scheduler

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/phuniverse/gtd/internal/calendar"
	"github.com/phuniverse/gtd/internal/config"
	"github.com/phuniverse/gtd/internal/db"
)

type Scheduler struct {
	db     *db.Database
	cal    *calendar.Client
	config *config.Config
}

func New(database *db.Database, calClient *calendar.Client, cfg *config.Config) *Scheduler {
	return &Scheduler{
		db:     database,
		cal:    calClient,
		config: cfg,
	}
}

func (s *Scheduler) AddTask(description, className string, dryRun bool) error {
	task := &db.Task{
		ID:           uuid.New().String(),
		Description:  description,
		ClassName:    className,
		ReviewCount:  0,
		Interval:     1,
		Status:       "pending",
		CreatedAt:    time.Now(),
		NextReviewAt: s.findNextReviewTime(className),
	}

	if dryRun {
		fmt.Printf("[DRY RUN] Would add task: %s for class: %s\n", description, className)
		fmt.Printf("[DRY RUN] Next review at: %s\n", task.NextReviewAt.Format("2006-01-02 15:04"))
		return nil
	}

	if err := s.db.SaveTask(task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	if err := s.attachTaskToCalendar(task); err != nil {
		return fmt.Errorf("failed to attach to calendar: %w", err)
	}

	fmt.Printf("✓ Added task: %s to %s\n", description, className)
	return nil
}

func (s *Scheduler) RemoveTask(description, className string, dryRun bool) error {
	tasks, err := s.db.GetTasksByClass(className)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.Description == description {
			if dryRun {
				fmt.Printf("[DRY RUN] Would remove task: %s\n", description)
				return nil
			}

			if err := s.db.DeleteTask(task.ID); err != nil {
				return err
			}

			fmt.Printf("✓ Removed task: %s\n", description)
			return nil
		}
	}

	return fmt.Errorf("task not found: %s", description)
}

func (s *Scheduler) ListReviews() error {
	today := time.Now()
	start := today.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	events, err := s.cal.GetStudyEvents(start, end)
	if err != nil {
		return err
	}

	fmt.Printf("\nREVIEW SESSIONS - %s\n", today.Format("2006-01-02"))
	fmt.Println("┌──────────────┬──────────────┬─────────────┐")
	fmt.Println("│ TIME         │ CLASS        │ TASKS       │")
	fmt.Println("├──────────────┼──────────────┼─────────────┤")

	for _, event := range events {
		timeStr := fmt.Sprintf("%s-%s", 
			event.Start.Format("15:04"), 
			event.End.Format("15:04"))
		
		tasks, _ := s.db.GetTasksByClass(event.Summary)
		taskCount := len(tasks)

		fmt.Printf("│ %-12s │ %-12s │ %-11s │\n", 
			timeStr, 
			truncate(event.Summary, 12), 
			fmt.Sprintf("%d tasks", taskCount))
	}

	fmt.Println("└──────────────┴──────────────┴─────────────┘")
	return nil
}

func (s *Scheduler) RunPlanScript(dryRun bool) error {
	fmt.Println("Running study scheduler plan script...")
	
	if dryRun {
		fmt.Println("[DRY RUN] Would execute:", s.config.PlanScriptPath)
		return nil
	}

	cmd := exec.Command("go", "run", s.config.PlanScriptPath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return fmt.Errorf("plan script failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println(string(output))
	fmt.Println("\n✓ Plan script completed successfully")
	return nil
}

func (s *Scheduler) findNextReviewTime(className string) time.Time {
	today := time.Now()
	weekEnd := today.AddDate(0, 0, 7)

	events, err := s.cal.GetStudyEvents(today, weekEnd)
	if err != nil {
		return today.Add(24 * time.Hour)
	}

	for _, event := range events {
		if contains(event.Summary, className) {
			return event.Start
		}
	}

	return today.Add(24 * time.Hour)
}

func (s *Scheduler) attachTaskToCalendar(task *db.Task) error {
	return nil
}

func (s *Scheduler) MarkTaskComplete(taskID string, interval int) error {
	task, err := s.db.GetTask(taskID)
	if err != nil {
		return err
	}

	now := time.Now()
	task.Status = "done"
	task.CompletedAt = &now
	task.ReviewCount++
	task.Interval = interval

	if interval == 0 {
		return s.db.SaveTask(task)
	}

	nextReview := s.calculateNextReview(task)
	task.NextReviewAt = nextReview
	task.Status = "pending"

	return s.db.SaveTask(task)
}

func (s *Scheduler) calculateNextReview(task *db.Task) time.Time {
	base := time.Now()
	
	switch task.Interval {
	case 1:
		return base.Add(24 * time.Hour)
	case 2:
		return base.Add(3 * 24 * time.Hour)
	case 3:
		return base.Add(7 * 24 * time.Hour)
	case 4:
		return base.Add(14 * 24 * time.Hour)
	default:
		return base.Add(time.Duration(task.Interval*7) * 24 * time.Hour)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
