//go:build ignore

// Example: Video rooms for team standup and conference streaming.
//
// Set these env vars (or pass them directly to NewRestClient):
//
//	SIGNALWIRE_PROJECT_ID   - your SignalWire project ID
//	SIGNALWIRE_API_TOKEN    - your SignalWire API token
//	SIGNALWIRE_SPACE        - your SignalWire space (e.g. example.signalwire.com)
//
// For full HTTP debug output:
//
//	SIGNALWIRE_LOG_LEVEL=debug
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// --- Video Rooms ---

	// 1. Create a video room
	fmt.Println("Creating video room...")
	room, err := client.Video.Rooms.Create(context.Background(), map[string]any{
		"name":         "daily-standup",
		"display_name": "Daily Standup",
		"max_members":  10,
		"layout":       "grid-responsive",
	})
	if err != nil {
		fmt.Printf("  Create room failed: %v\n", err)
		return
	}
	roomID := room["id"].(string)
	fmt.Printf("  Created room: %s\n", roomID)

	// 2. List video rooms
	fmt.Println("\nListing video rooms...")
	rooms, err := client.Video.Rooms.List(context.Background(), nil)
	if err == nil {
		if data, ok := rooms["data"].([]any); ok {
			limit := 5
			if len(data) < limit {
				limit = len(data)
			}
			for _, r := range data[:limit] {
				if m, ok := r.(map[string]any); ok {
					fmt.Printf("  - %s: %v\n", m["id"], m["name"])
				}
			}
		}
	}

	// 3. Generate a join token
	fmt.Println("\nGenerating room token...")
	token, err := client.Video.RoomTokens.Create(context.Background(), namespaces.VideoRoomTokensCreateParams{Extras: map[string]any{
		"room_name":   "daily-standup",
		"user_name":   "alice",
		"permissions": []string{"room.self.audio_mute", "room.self.video_mute"},
	}})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Token failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		tokenStr := token.Token
		if len(tokenStr) > 40 {
			tokenStr = tokenStr[:40]
		}
		fmt.Printf("  Token: %s...\n", tokenStr)
	}

	// --- Sessions ---

	// 4. List room sessions
	fmt.Println("\nListing room sessions...")
	sessions, err := client.Video.RoomSessions.List(context.Background(), nil)
	if err == nil {
		if data, ok := sessions["data"].([]any); ok {
			limit := 3
			if len(data) < limit {
				limit = len(data)
			}
			for _, s := range data[:limit] {
				if m, ok := s.(map[string]any); ok {
					fmt.Printf("  - Session %s: %v\n", m["id"], m["status"])
				}
			}
		}
	}

	// 5. Get session details with members, events, recordings
	if sessions != nil {
		if data, ok := sessions["data"].([]any); ok && len(data) > 0 {
			if first, ok := data[0].(map[string]any); ok {
				if sid, ok := first["id"].(string); ok {
					detail, err := client.Video.RoomSessions.Get(context.Background(), sid)
					if err == nil {
						fmt.Printf("  Session: %v (%v)\n", detail["name"], detail["status"])
					}

					members, err := client.Video.RoomSessions.ListMembers(context.Background(), sid, nil)
					if err == nil {
						fmt.Printf("  Members: %d\n", len(members.Data))
					}

					events, err := client.Video.RoomSessions.ListEvents(context.Background(), sid, nil)
					if err == nil {
						fmt.Printf("  Events: %d\n", len(events.Data))
					}

					recs, err := client.Video.RoomSessions.ListRecordings(context.Background(), sid, nil)
					if err == nil {
						fmt.Printf("  Recordings: %d\n", len(recs.Data))
					}
				}
			}
		}
	}

	// --- Room Recordings ---

	// 6. List and get room recordings
	fmt.Println("\nListing room recordings...")
	roomRecs, err := client.Video.RoomRecordings.List(context.Background(), nil)
	if err == nil {
		data := roomRecs.Data
		limit := 3
		if len(data) < limit {
			limit = len(data)
		}
		for _, rr := range data[:limit] {
			fmt.Printf("  - Recording %s: %vs\n", rr.ID, rr.Duration)
		}

		if len(data) > 0 {
			recID := data[0].ID
			recDetail, err := client.Video.RoomRecordings.Get(context.Background(), recID, nil)
			if err == nil {
				fmt.Printf("  Recording detail: %vs\n", recDetail.Duration)
			}

			recEvents, err := client.Video.RoomRecordings.ListEvents(context.Background(), recID, nil)
			if err == nil {
				fmt.Printf("  Recording events: %d\n", len(recEvents.Data))
			}
		}
	}

	// --- Video Conferences ---

	// 7. Create a video conference
	fmt.Println("\nCreating video conference...")
	var confID string
	conf, err := client.Video.Conferences.Create(context.Background(), map[string]any{
		"name":         "all-hands-stream",
		"display_name": "All Hands Meeting",
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Conference creation failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		confID = conf["id"].(string)
		fmt.Printf("  Created conference: %s\n", confID)
	}

	// 8. List conference tokens
	if confID != "" {
		fmt.Println("\nListing conference tokens...")
		tokens, err := client.Video.Conferences.ListConferenceTokens(context.Background(), confID, nil)
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Conference tokens failed: %d\n", restErr.StatusCode)
			}
		} else {
			for _, t := range tokens.Data {
				fmt.Printf("  - Token: %v\n", t.ID)
			}
		}
	}

	// 9. Create a stream on the conference
	var streamID string
	if confID != "" {
		fmt.Println("\nCreating stream on conference...")
		stream, err := client.Video.Conferences.CreateStream(context.Background(), confID, namespaces.VideoConferencesCreateStreamParams{Extras: map[string]any{
			"url": "rtmp://live.example.com/stream-key",
		}})
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Stream creation failed (expected in demo): %d\n", restErr.StatusCode)
			}
		} else {
			streamID = stream.ID
			fmt.Printf("  Created stream: %s\n", streamID)
		}
	}

	// 10. Get and update stream
	if streamID != "" {
		fmt.Printf("\nManaging stream %s...\n", streamID)
		sDetail, err := client.Video.Streams.Get(context.Background(), streamID, nil)
		if err == nil {
			fmt.Printf("  Stream URL: %v\n", sDetail.URL)
		}

		_, err = client.Video.Streams.Update(context.Background(), streamID, namespaces.VideoStreamsUpdateParams{Extras: map[string]any{
			"url": "rtmp://backup.example.com/stream-key",
		}})
		if err == nil {
			fmt.Println("  Stream URL updated")
		} else if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Stream ops failed: %d\n", restErr.StatusCode)
		}
	}

	// 11. Clean up
	fmt.Println("\nCleaning up...")
	if streamID != "" {
		if _, err := client.Video.Streams.Delete(context.Background(), streamID); err == nil {
			fmt.Printf("  Deleted stream %s\n", streamID)
		} else if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Stream delete failed: %d\n", restErr.StatusCode)
		}
	}
	if confID != "" {
		client.Video.Conferences.Delete(context.Background(), confID)
		fmt.Printf("  Deleted conference %s\n", confID)
	}
	client.Video.Rooms.Delete(context.Background(), roomID)
	fmt.Printf("  Deleted room %s\n", roomID)
}
