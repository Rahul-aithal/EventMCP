# Scheduler

Small Google Calendar MCP server built in Go. A personal side project for quick calendar automation.

## Features

- Add events
- List upcoming events
- Delete events by name and date window
- Google OAuth2 login
- MCP over stdio
- Uses the MCP Go SDK
- Loads `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` from `.env`
- Stores a refresh token locally

## Quick setup

1. Create a Google OAuth client (Desktop app).
2. Put a `.env` file next to the built binary, or export the env vars.
3. Add your credentials:

```
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
```

On first run, the server prints an auth URL and asks for the code.

## Build / run

```
go build -o scheduler .
./scheduler
```

Or:

```
GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... go run .
```

## MCP config example

```
{
  "mcpServers": {
    "scheduler": {
      "command": "/absolute/path/to/scheduler"
    }
  }
}
```

## Example tool usage

```
addEvent {
  "name": "Coffee chat",
  "start_date_time": "2026-05-07T10:00:00+05:30",
  "end_date_time": "2026-05-07T10:30:00+05:30"
}
```

```
listUpcomingEvents { "limit": 5 }
```

```
deleteEventByNameAndDate {
  "name": "Coffee chat",
  "min_date": "2026-05-07T00:00:00+05:30",
  "max_date": "2026-05-08T00:00:00+05:30"
}
```

## Project structure

- `main.go` — MCP server, tools, OAuth, calendar logic
- `go.mod` — Go dependencies
- `.env` — local credentials (not committed)

## Notes

- Refresh token is stored in your user config dir (e.g. `~/.config/sechduler/refresh_token.txt` on Linux).
- Uses your primary Google Calendar.
- Event timestamps use the Asia/Kolkata timezone.
