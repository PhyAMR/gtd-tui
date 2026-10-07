package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// Paths
	DBPath            string `json:"db_path"`
	TokenPath         string `json:"token_path"`
	CredentialsPath   string `json:"credentials_path"`
	PlanScriptPath    string `json:"plan_script_path"`

	// Calendar IDs
	ClassCalendarID   string `json:"class_calendar_id"`
	StudyCalendarID   string `json:"study_calendar_id"`

	// Settings
	ReviewRatio       float64            `json:"review_ratio"`
	TimeWindows       string             `json:"time_windows"`
	SyncIntervalSecs  int                `json:"sync_interval_secs"`
	
	// Pomodoro settings
	PomodoroWorkMins  int                `json:"pomodoro_work_mins"`
	PomodoroBreakMins int                `json:"pomodoro_break_mins"`
	PomodoroLongBreak int                `json:"pomodoro_long_break"`
	
	// UI settings
	Theme             string             `json:"theme"`
	KeyBindings       map[string]string  `json:"key_bindings"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "gtd")

	return &Config{
		DBPath:            filepath.Join(configDir, "gtd.db"),
		TokenPath:         filepath.Join(configDir, "tokens.json"),
		CredentialsPath:   filepath.Join(home, ".credentials", "credentials.json"),
		PlanScriptPath:    filepath.Join(home, "Scripts", "study_scheduler.go"),
		ClassCalendarID:   "primary",
		StudyCalendarID:   "primary",
		ReviewRatio:       0.75,
		TimeWindows:       "9-13,16-20.5",
		SyncIntervalSecs:  300,
		PomodoroWorkMins:  25,
		PomodoroBreakMins: 5,
		PomodoroLongBreak: 15,
		Theme:             "default",
		KeyBindings:       DefaultKeyBindings(),
	}
}

func DefaultKeyBindings() map[string]string {
	return map[string]string{
		"quit":          "q",
		"tab":           "tab",
		"todo_up":       "k",
		"todo_down":     "j",
		"todo_toggle":   "enter",
		"todo_add":      "a",
		"todo_delete":   "d",
		"pomo_start":    "s",
		"pomo_reset":    "r",
		"pomo_skip":     "n",
	}
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "gtd", "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "gtd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}
