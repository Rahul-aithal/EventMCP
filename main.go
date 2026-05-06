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

func listEventsWithLimit(ctx context.Context, req *mcp.CallToolRequest, input ListEventsMcp) (*mcp.CallToolResult, any, error) {
	log.Println("Listing events")
	events := listEvents(ctx, input.Limit)
	event_text := strings.Join(events, "\n-")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: event_text}},
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
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}
	log.Println("Upcoming events:")
	if limit > len(events.Items) {
		limit = len(events.Items)
	}

	event_slice := make([]string, limit)
	if len(events.Items) == 0 {
		log.Println("No upcoming events found.")
	} else {

		for index := range limit {
			item := events.Items[index]
			date := item.Start.DateTime
			if date == "" {
				date = item.Start.Date
			}
			event_slice = append(event_slice, fmt.Sprintf("%v (%v)\n", item.Summary, date))
		}
	}
	return event_slice
}

func main() {
	file, ferr := filepath.Abs("code/scheduler/.env")

	if ferr == nil {
		err := godotenv.Load(file)
		if err != nil {
			log.Fatalf("Error while loading env file %v", err)
			return
		}
		log.Printf("Env file loaded sucessfully")
	}
	if ferr != nil {
		log.Println("No env file found make sure env is iassed while calling the mcp,\nGOOGLE_CLIENT_ID: \nGOOGLE_CLIENT_SECRET: \nare the names")
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
		Name:    "Sechduler",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "addEvent",
		Description: "Adds the event to calendar",
	}, addEvent)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "listEvents",
		Description: "Lists all the future events from now",
	}, listEventsWithLimit)

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
