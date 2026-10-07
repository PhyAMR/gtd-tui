package main

import (
	"fmt"
	"os"

	"github.com/phuniverse/gtd/internal/calendar"
	"github.com/phuniverse/gtd/internal/config"
	"github.com/phuniverse/gtd/internal/db"
	"github.com/phuniverse/gtd/internal/scheduler"
	"github.com/phuniverse/gtd/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	dryRun  bool
	class   string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "gtd",
	Short:   "GTD - A minimal-distraction spaced-repetition study assistant",
	Version: version,
	Long: `GTD is a TUI application that helps you manage your study schedule
using Google Calendar and spaced repetition techniques.`,
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		database, err := db.New(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		calClient, err := calendar.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create calendar client: %w", err)
		}

		sched := scheduler.New(database, calClient, cfg)

		return ui.RunDashboard(database, calClient, sched, cfg)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List items (reviews or classes)",
}

var listReviewsCmd = &cobra.Command{
	Use:   "-r",
	Short: "List today's review sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		database, err := db.New(cfg.DBPath)
		if err != nil {
			return err
		}
		defer database.Close()

		calClient, err := calendar.NewClient(cfg)
		if err != nil {
			return err
		}

		sched := scheduler.New(database, calClient, cfg)
		return sched.ListReviews()
	},
}

var listClassesCmd = &cobra.Command{
	Use:   "-c",
	Short: "List all classes from Class Calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		calClient, err := calendar.NewClient(cfg)
		if err != nil {
			return err
		}

		return calClient.ListClasses()
	},
}

var addCmd = &cobra.Command{
	Use:   "add [task description]",
	Short: "Add a task to an upcoming review",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskDesc := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		database, err := db.New(cfg.DBPath)
		if err != nil {
			return err
		}
		defer database.Close()

		calClient, err := calendar.NewClient(cfg)
		if err != nil {
			return err
		}

		sched := scheduler.New(database, calClient, cfg)
		return sched.AddTask(taskDesc, class, dryRun)
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm [task description]",
	Short: "Remove a task from an upcoming review",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskDesc := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		database, err := db.New(cfg.DBPath)
		if err != nil {
			return err
		}
		defer database.Close()

		calClient, err := calendar.NewClient(cfg)
		if err != nil {
			return err
		}

		sched := scheduler.New(database, calClient, cfg)
		return sched.RemoveTask(taskDesc, class, dryRun)
	},
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Run the study scheduler planning script",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		database, err := db.New(cfg.DBPath)
		if err != nil {
			return err
		}
		defer database.Close()

		calClient, err := calendar.NewClient(cfg)
		if err != nil {
			return err
		}

		sched := scheduler.New(database, calClient, cfg)
		return sched.RunPlanScript(dryRun)
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(planCmd)

	listCmd.AddCommand(listReviewsCmd)
	listCmd.AddCommand(listClassesCmd)

	addCmd.Flags().StringVarP(&class, "class", "c", "", "Class/lecture name")
	addCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	addCmd.MarkFlagRequired("class")

	rmCmd.Flags().StringVarP(&class, "class", "c", "", "Class/lecture name")
	rmCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	rmCmd.MarkFlagRequired("class")

	planCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
}
