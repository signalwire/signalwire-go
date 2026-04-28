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
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/pkg/rest"
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
	room, err := client.Video.Rooms.Create(map[string]any{
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
	rooms, err := client.Video.Rooms.List(nil)
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
	token, err := client.Video.RoomTokens.Create(map[string]any{
		"room_name":   "daily-standup",
		"user_name":   "alice",
		"permissions": []string{"room.self.audio_mute", "room.self.video_mute"},
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Token failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		tokenStr, _ := token["token"].(string)
		if len(tokenStr) > 40 {
			tokenStr = tokenStr[:40]
		}
		fmt.Printf("  Token: %s...\n", tokenStr)
	}

	// --- Sessions ---

	// 4. List room sessions
	fmt.Println("\nListing room sessions...")
	sessions, err := client.Video.RoomSessions.List(nil)
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
					detail, err := client.Video.RoomSessions.Get(sid)
					if err == nil {
						fmt.Printf("  Session: %v (%v)\n", detail["name"], detail["status"])
					}

					members, err := client.Video.RoomSessions.ListMembers(sid, nil)
					if err == nil {
						if d, ok := members["data"].([]any); ok {
							fmt.Printf("  Members: %d\n", len(d))
						}
					}

					events, err := client.Video.RoomSessions.ListEvents(sid, nil)
					if err == nil {
						if d, ok := events["data"].([]any); ok {
							fmt.Printf("  Events: %d\n", len(d))
						}
					}

					recs, err := client.Video.RoomSessions.ListRecordings(sid, nil)
					if err == nil {
						if d, ok := recs["data"].([]any); ok {
							fmt.Printf("  Recordings: %d\n", len(d))
						}
					}
				}
			}
		}
	}

	// --- Room Recordings ---

	// 6. List and get room recordings
	fmt.Println("\nListing room recordings...")
	roomRecs, err := client.Video.RoomRecordings.List(nil)
	if err == nil {
		if data, ok := roomRecs["data"].([]any); ok {
			limit := 3
			if len(data) < limit {
				limit = len(data)
			}
			for _, rr := range data[:limit] {
				if m, ok := rr.(map[string]any); ok {
					fmt.Printf("  - Recording %s: %vs\n", m["id"], m["duration"])
				}
			}

			if len(data) > 0 {
				if first, ok := data[0].(map[string]any); ok {
					if recID, ok := first["id"].(string); ok {
						recDetail, err := client.Video.RoomRecordings.Get(recID)
						if err == nil {
							fmt.Printf("  Recording detail: %vs\n", recDetail["duration"])
						}

						recEvents, err := client.Video.RoomRecordings.ListEvents(recID, nil)
						if err == nil {
							if d, ok := recEvents["data"].([]any); ok {
								fmt.Printf("  Recording events: %d\n", len(d))
							}
						}
					}
				}
			}
		}
	}

	// --- Video Conferences ---

	// 7. Create a video conference
	fmt.Println("\nCreating video conference...")
	var confID string
	conf, err := client.Video.Conferences.Create(map[string]any{
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
		tokens, err := client.Video.Conferences.ListConferenceTokens(confID, nil)
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Conference tokens failed: %d\n", restErr.StatusCode)
			}
		} else if data, ok := tokens["data"].([]any); ok {
			for _, t := range data {
				if m, ok := t.(map[string]any); ok {
					fmt.Printf("  - Token: %v\n", m["id"])
				}
			}
		}
	}

	// 9. Create a stream on the conference
	var streamID string
	if confID != "" {
		fmt.Println("\nCreating stream on conference...")
		stream, err := client.Video.Conferences.CreateStream(confID, map[string]any{
			"url": "rtmp://live.example.com/stream-key",
		})
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  Stream creation failed (expected in demo): %d\n", restErr.StatusCode)
			}
		} else {
			streamID = stream["id"].(string)
			fmt.Printf("  Created stream: %s\n", streamID)
		}
	}

	// 10. Get and update stream
	if streamID != "" {
		fmt.Printf("\nManaging stream %s...\n", streamID)
		sDetail, err := client.Video.Streams.Get(streamID)
		if err == nil {
			fmt.Printf("  Stream URL: %v\n", sDetail["url"])
		}

		_, err = client.Video.Streams.Update(streamID, map[string]any{
			"url": "rtmp://backup.example.com/stream-key",
		})
		if err == nil {
			fmt.Println("  Stream URL updated")
		} else if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Stream ops failed: %d\n", restErr.StatusCode)
		}
	}

	// 11. Clean up
	fmt.Println("\nCleaning up...")
	if streamID != "" {
		if _, err := client.Video.Streams.Delete(streamID); err == nil {
			fmt.Printf("  Deleted stream %s\n", streamID)
		} else if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Stream delete failed: %d\n", restErr.StatusCode)
		}
	}
	if confID != "" {
		client.Video.Conferences.Delete(confID)
		fmt.Printf("  Deleted conference %s\n", confID)
	}
	client.Video.Rooms.Delete(roomID)
	fmt.Printf("  Deleted room %s\n", roomID)
}
