package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type AddEventMCP struct {
	Name          string    `json:"name"`
	StartDateTime time.Time `json:"start_date_time"`
	EndDateTime   time.Time `json:"end_date_time"`
}
type ListEventsMCP struct {
	Limit int `json:"limit"`
}
type DeleteEventMCP struct {
	Name    string    `json:"name"`
	MaxDate time.Time `json:"max_date"`
	MinDate time.Time `json:"min_date"`
}

var config *oauth2.Config

// ====================
// MCP Tool Definitions
// ====================
func addEvent(ctx context.Context, req *mcp.CallToolRequest, input AddEventMCP) (*mcp.CallToolResult, any, error) {
	log.Println("Add event Started")
	event := &calendar.Event{
		Summary: input.Name,
		Start: &calendar.EventDateTime{
			DateTime: input.StartDateTime.Format(time.RFC3339),
			TimeZone: "Asia/Kolkata",
		},
		End: &calendar.EventDateTime{
			DateTime: input.EndDateTime.Format(time.RFC3339),
			TimeZone: "Asia/Kolkata",
		},
	}
	err := createEvent(ctx, event)
	if err != nil {
		return nil, nil, err
	}
	eventResult := "Event created: " + event.HtmlLink
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: eventResult}},
	}, nil, nil
}

func listUpcomingEventsByLimit(ctx context.Context, req *mcp.CallToolRequest, input ListEventsMCP) (*mcp.CallToolResult, any, error) {
	log.Println("Listing events")
	if input.Limit < 1 {

		return nil, nil, fmt.Errorf("limit must be greater than 0")
	}
	events, err := listUpcomingEvents(ctx, input.Limit)

	if err != nil {
		return nil, nil, err
	}
	if len(events) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "No upcoming events found.",
				},
			},
		}, nil, nil
	}
	var eventText strings.Builder

	for i, event := range events {
		fmt.Fprintf(&eventText, "%d. %s\n", i+1, event)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: eventText.String()}},
	}, nil, nil
}

func deleteEventByName(ctx context.Context, req *mcp.CallToolRequest, input DeleteEventMCP) (*mcp.CallToolResult, any, error) {
	log.Println("Deleting events")
	result, err := deleteEvent(ctx, input)

	if err != nil {
		return nil, nil, err
	}
	resp := input.Name
	if result {
		resp = resp + " successfully deleted\n"
	} else {
		resp = resp + " could not be deleted\n"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

// ====================
// Google Calendar Logic
// ====================

func createEvent(ctx context.Context, event *calendar.Event) error {
	calendarId := "primary"
	calendarService, err := createCalService(ctx)
	//
	if err != nil {
		return fmt.Errorf("error while creating calendar service: %w", err)
	}
	event, err = calendarService.Events.Insert(calendarId, event).Do()
	if err != nil {
		return fmt.Errorf("unable to create event: %w", err)
	}

	log.Printf("Event Created from calendar:%v\n", event.HtmlLink)
	return nil
}
func deleteEvent(ctx context.Context, eventData DeleteEventMCP) (bool, error) {
	calendarId := "primary"
	calendarService, err := createCalService(ctx)

	if err != nil {
		return false, fmt.Errorf("error while creating calendar service: %w", err)
	}
	events, err := calendarService.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(eventData.MinDate.Format(time.RFC3339)).TimeMax(eventData.MaxDate.Format(time.RFC3339)).
		MaxResults(1).OrderBy("startTime").Q(eventData.Name).Do()

	if err != nil {
		return false, fmt.Errorf("error while searching calendar event: %w", err)
	}
	if len(events.Items) == 0 {
		return false, fmt.Errorf("no items found")
	}
	err = calendarService.Events.Delete(calendarId, events.Items[0].Id).Do()
	if err != nil {
		return false, fmt.Errorf("unable to delete event: %w", err)
	}

	log.Printf("Event Delete from calendar:%v\n", events.Items[0].Summary)

	return true, nil
}

func listUpcomingEvents(ctx context.Context, limit int) ([]string, error) {
	t := time.Now().Format(time.RFC3339)

	calendarService, err := createCalService(ctx)
	//
	if err != nil {
		return nil, fmt.Errorf("error while creating calendar service: %w", err)
	}
	events, err := calendarService.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(int64(limit)).OrderBy("startTime").Do()

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve next ten of the user's events: %w", err)
	}

	eventList := make([]string, 0, len(events.Items))

	for _, item := range events.Items {

		date := item.Start.DateTime
		if date != "" {
			t, err := time.Parse(time.RFC3339, date)
			if err != nil {
				return nil, fmt.Errorf("error while parsing time: %w", err)
			}
			date = t.Format("02-01-2006 03:04:05 PM")
		} else {
			date = item.Start.Date
		}
		value := fmt.Sprintf("%s — %s", item.Summary, date)
		eventList = append(eventList, value)
	}

	return eventList, nil
}

func createCalService(ctx context.Context) (*calendar.Service, error) {
	token, err := login(ctx)
	if err != nil {
		return nil, fmt.Errorf("logging in: %w", err)
	}
	calendarService, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	//
	if err != nil {
		return nil, fmt.Errorf("creating calendar service: %w", err)
	}
	return calendarService, nil
}

// ====================
// OAuth Authentication
// ====================
func login(ctx context.Context) (*oauth2.Token, error) {
	configdir, _ := os.UserConfigDir()

	refreshTokenPath := filepath.Join(configdir, "scheduler", "refresh_token.txt")

	schedulerConfigDir := filepath.Join(configdir, "scheduler")
	_, err := os.Stat(schedulerConfigDir)

	if os.IsNotExist(err) {
		err := os.Mkdir(schedulerConfigDir, 0755)

		if err != nil {
			return nil, err
		}
	}

	refreshTokenData, rerr := os.ReadFile(refreshTokenPath)

	if rerr == nil {

		initToken := &oauth2.Token{
			RefreshToken: string(refreshTokenData),
		}

		ts := config.TokenSource(ctx, initToken)
		token, err := ts.Token()

		if err != nil {
			return nil, err
		}

		rerr := os.WriteFile(refreshTokenPath, []byte(token.RefreshToken), 0644)

		if rerr != nil {
			return nil, rerr
		}

		log.Printf("Logged in successfully")
		return token, nil

	}

	url := config.AuthCodeURL("state")

	log.Println(url)
	log.Println("Paste the authorization code:")

	var code string
	_, serr := fmt.Scan(&code)

	if serr != nil {
		return nil, serr
	}
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	rerr = os.WriteFile(refreshTokenPath, []byte(token.RefreshToken), 0644)
	if rerr != nil {
		return nil, rerr
	}

	return token, nil
}

func main() {
	exe, err := os.Executable()

	if err != nil {
		log.Fatal(err)
		return
	}

	exeDir := filepath.Dir(exe)
	file := filepath.Join(exeDir, ".env")

	if _, err := os.Stat(file); !os.IsNotExist(err) {

		if err := godotenv.Load(file); err != nil {
			log.Fatalf("error while loading env file %v", err)
			return
		}

		log.Printf("Env file loaded successfully")

	} else {

		log.Println("No env file found. Pass:")
		log.Println("GOOGLE_CLIENT_ID")
		log.Println("GOOGLE_CLIENT_SECRET")

	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	config = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Scopes:       []string{"https://www.googleapis.com/auth/calendar"},
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Scheduler",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "addEvent",
		Description: "Adds the event to calendar",
	}, addEvent)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "listUpcomingEvents",
		Description: "Lists all the future events from now",
	}, listUpcomingEventsByLimit)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "deleteEventByNameAndDate",
		Description: "Deletes the event by name and start and end date",
	}, deleteEventByName)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
