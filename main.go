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
type ListEventsMcp struct {
	Limit int `json:"list"`
}
type DeleteEventMcp struct {
	Name    string
	MaxDate time.Time
	MinDate time.Time
}

var config *oauth2.Config

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
	creatEvent(ctx, event)
	eventReulst := fmt.Sprintf("Event created: %s\n", event.HtmlLink)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: eventReulst}},
	}, nil, nil
}

func listEventsByLimit(ctx context.Context, req *mcp.CallToolRequest, input ListEventsMcp) (*mcp.CallToolResult, any, error) {
	log.Println("Listing events")
	if input.Limit < 1 {
		log.Fatal("0 limit is not allowed")
	}
	events := listEvents(ctx, input.Limit)
	var builder strings.Builder

	for i, event := range events {
		fmt.Fprintf(&builder, "%d. %s\n", i+1, event)
	}

	event_text := builder.String()
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: event_text}},
	}, nil, nil
}

func deleteEventByName(ctx context.Context, req *mcp.CallToolRequest, input DeleteEventMcp) (*mcp.CallToolResult, any, error) {
	log.Println("Listing events")
	result := deleteEvent(ctx, input)
	resp := input.Name
	if result {
		resp = resp + " successfully deleted\n"
	} else {
		resp = resp + " couldn't not be deleted\n"
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: resp}},
	}, nil, nil
}

func creatEvent(ctx context.Context, event *calendar.Event) {

	calendarId := "primary"
	calendarService, err := createCalService(ctx)
	//
	if err != nil {
		log.Fatalf("Error while creating calendar service")
	}
	event, err = calendarService.Events.Insert(calendarId, event).Do()
	if err != nil {
		log.Fatalf("Unable to create event. %v\n", err)
	}

	log.Printf("Event Created from calendar:%v\n", event.HtmlLink)

}
func deleteEvent(ctx context.Context, event_data DeleteEventMcp) bool {

	calendarId := "primary"
	calendarService, err := createCalService(ctx)

	if err != nil {
		log.Fatalf("Error while creating calendar service")
		return false
	}
	events, err := calendarService.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(event_data.MinDate.Format(time.RFC3339)).TimeMax(event_data.MaxDate.Format(time.RFC3339)).
		MaxResults(1).OrderBy("startTime").Q(event_data.Name).Do()

	if err != nil {
		log.Fatalf("Error while searching calendar event ")
	}
	err = calendarService.Events.Delete(calendarId, events.Items[0].Id).Do()
	if err != nil {
		log.Fatalf("Unable to delete a event. %v\n", err)
		return false
	}

	log.Printf("Event Delete from calendar:%v\n", events.Items[0].Summary)

	return true
}
func login(ctx context.Context) (*oauth2.Token, error) {
	configdir, _ := os.UserConfigDir()
	refresh_token_path := filepath.Join(configdir, "sechduler", "refresh_token.txt")
	path := filepath.Join(configdir, "sechduler")
	_, err := os.Stat(path)

	if err != nil {
		err := os.Mkdir(path, 0755)
		if err != nil {
			log.Fatal(err)
		}

	}

	refe_token_data, rerr := os.ReadFile(refresh_token_path)

	if rerr == nil {

		log.Printf("Reading the tokens form file\n refresh_token: (%s)\n", string(refe_token_data))

		initToken := &oauth2.Token{
			RefreshToken: string(refe_token_data),
		}
		ts := config.TokenSource(ctx, initToken)
		token, err := ts.Token()
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		// fmt.Printf("New token found %s\n", token.AccessToken)
		// fmt.Printf("New token found %v\n", token.ExpiresIn)

		rerr := os.WriteFile(refresh_token_path, []byte(token.RefreshToken), 0644)
		if rerr != nil {
			log.Fatal(rerr)
		}
		log.Printf("Logged In sucessfully")
		return token, nil
	}

	url := config.AuthCodeURL("state")
	log.Printf("%s", url)

	log.Println("Paste the code from browser: ")
	var code string
	fmt.Scan(&code)
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	rerr = os.WriteFile(refresh_token_path, []byte(token.RefreshToken), 0644)
	if rerr != nil {
		log.Fatal(rerr)
	}
	return token, nil
}

func createCalService(ctx context.Context) (*calendar.Service, error) {
	token, err := login(ctx)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	calendarService, err := calendar.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))
	//
	if err != nil {
		log.Fatalf("Error while creating calendar service")
		return nil, err
	}
	return calendarService, nil
}

func listEvents(ctx context.Context, limit int) []string {

	t := time.Now().Format(time.RFC3339)

	calendarService, err := createCalService(ctx)
	//
	if err != nil {
		log.Fatalf("Error while creating calendar service")
	}
	events, err := calendarService.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(int64(limit)).OrderBy("startTime").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}
	log.Println("Upcoming events:")

	event_slice := make([]string, limit)
	if len(events.Items) == 0 {
		log.Println("No upcoming events found.")
	} else {

		for index := range limit {
			item := events.Items[index]

			date := item.Start.DateTime
			if date != "" {
				t, err := time.Parse(time.RFC3339, date)
				if err != nil {
					log.Fatalf("Error while parsing time %v", err)
				}
				date = t.Format("02-01-2006 03:04:05 PM")
			} else {
				date = item.Start.Date
			}
			value := fmt.Sprintf("%v (%v)\n", item.Summary, date)
			if len(value) == 0 {
				continue
			}
			event_slice[index] = value
		}
	}
	return event_slice
}

func main() {
	exe, err := os.Executable()

	if err != nil {
		log.Fatal(err)
		return
	}

	exeDir := filepath.Dir(exe)
	file := filepath.Join(exeDir, ".env")

	if _, err := os.Stat(file); err == nil {

		if err := godotenv.Load(file); err != nil {
			log.Fatalf("Error while loading env file %v", err)
			return
		}

		log.Printf("Env file loaded sucessfully")

	} else {

		log.Println("No env file found. Pass:")
		log.Println("GOOGLE_CLIENT_ID")
		log.Println("GOOGLE_CLIENT_SECRET")

	}

	google_client_id := os.Getenv("GOOGLE_CLIENT_ID")
	google_client_secret := os.Getenv("GOOGLE_CLIENT_SECRET")

	config = &oauth2.Config{
		ClientID:     google_client_id,
		ClientSecret: google_client_secret,
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
		Name:        "listEvents",
		Description: "Lists all the future events from now",
	}, listEventsByLimit)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "deleteEventByNameAndDate",
		Description: "Deletes the event by name and start and end date",
	}, deleteEventByName)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// ctx := context.Background()
//
// event := &calendar.Event{
// 	Summary:     "test",
// 	Location:    "India",
// 	Description: "Delete as soon as you see",
// 	Start: &calendar.EventDateTime{
// 		DateTime: time.Now().Format(time.RFC3339),
// 		TimeZone: "Asia/Kolkata",
// 	},
// 	End: &calendar.EventDateTime{
// 		DateTime: time.Now().Format(time.RFC3339),
// 		TimeZone: "Asia/Kolkata",
// 	},
// }
// creatEvent(ctx, event)
