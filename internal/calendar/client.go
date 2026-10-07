package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phuniverse/gtd/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Client struct {
	service         *calendar.Service
	classCalendarID string
	studyCalendarID string
}

type Event struct {
	ID          string
	Summary     string
	Description string
	Start       time.Time
	End         time.Time
	Location    string
	CalendarID  string
}

func NewClient(cfg *config.Config) (*Client, error) {
	ctx := context.Background()

	b, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w\nPlease set up Google API credentials first", err)
	}

	oauthConfig, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	token, err := tokenFromFile(cfg.TokenPath)
	if err != nil {
		token = getTokenFromWeb(oauthConfig)
		saveToken(cfg.TokenPath, token)
	}

	client := oauthConfig.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Calendar service: %w", err)
	}

	return &Client{
		service:         srv,
		classCalendarID: cfg.ClassCalendarID,
		studyCalendarID: cfg.StudyCalendarID,
	}, nil
}

func (c *Client) GetEvents(calendarID string, start, end time.Time) ([]*Event, error) {
	events, err := c.service.Events.List(calendarID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	var result []*Event
	for _, item := range events.Items {
		startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		if item.Start.DateTime == "" {
			startTime, _ = time.Parse("2006-01-02", item.Start.Date)
		}

		endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)
		if item.End.DateTime == "" {
			endTime, _ = time.Parse("2006-01-02", item.End.Date)
		}

		result = append(result, &Event{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			Start:       startTime,
			End:         endTime,
			Location:    item.Location,
			CalendarID:  calendarID,
		})
	}

	return result, nil
}

func (c *Client) GetClassEvents(start, end time.Time) ([]*Event, error) {
	return c.GetEvents(c.classCalendarID, start, end)
}

func (c *Client) GetStudyEvents(start, end time.Time) ([]*Event, error) {
	return c.GetEvents(c.studyCalendarID, start, end)
}

func (c *Client) CreateEvent(calendarID string, event *Event) (string, error) {
	calEvent := &calendar.Event{
		Summary:     event.Summary,
		Description: event.Description,
		Location:    event.Location,
		Start: &calendar.EventDateTime{
			DateTime: event.Start.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: event.End.Format(time.RFC3339),
		},
		ColorId: "9",
	}

	created, err := c.service.Events.Insert(calendarID, calEvent).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return created.Id, nil
}

func (c *Client) UpdateEvent(calendarID, eventID string, event *Event) error {
	calEvent := &calendar.Event{
		Summary:     event.Summary,
		Description: event.Description,
		Location:    event.Location,
		Start: &calendar.EventDateTime{
			DateTime: event.Start.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: event.End.Format(time.RFC3339),
		},
	}

	_, err := c.service.Events.Update(calendarID, eventID, calEvent).Do()
	return err
}

func (c *Client) DeleteEvent(calendarID, eventID string) error {
	return c.service.Events.Delete(calendarID, eventID).Do()
}

func (c *Client) ListClasses() error {
	today := time.Now()
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 7)

	events, err := c.GetClassEvents(weekStart, weekEnd)
	if err != nil {
		return err
	}

	fmt.Println("\nCLASSES THIS WEEK")
	fmt.Println("─────────────────────────────────────────")
	
	classMap := make(map[string]bool)
	for _, event := range events {
		if !classMap[event.Summary] {
			fmt.Printf("• %s\n", event.Summary)
			classMap[event.Summary] = true
		}
	}
	
	fmt.Printf("\nTotal: %d unique classes\n", len(classMap))
	return nil
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)
	fmt.Print("Enter authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		fmt.Printf("Unable to read authorization code: %v\n", err)
		os.Exit(1)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		fmt.Printf("Unable to retrieve token from web: %v\n", err)
		os.Exit(1)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Printf("Unable to create token directory: %v\n", err)
		return
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("Unable to cache oauth token: %v\n", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
