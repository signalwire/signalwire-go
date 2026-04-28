//go:build ignore

// Example: room_and_sip
//
// Room and SIP configuration. Demonstrates JoinRoom, JoinConference,
// and SipRefer helpers on SwaigFunctionResult for multi-party
// communication and SIP call transfers.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	fmt.Println("Room and SIP Examples")
	fmt.Println()

	// ---- Basic room join ----
	fmt.Println("=== Basic Room Join ===")
	roomJoin := swaig.NewFunctionResult("Joining the support team room").
		JoinRoom("support_team_room").
		Say("Welcome to the support team collaboration room")
	printResult(roomJoin)

	// ---- Conference room with metadata ----
	fmt.Println("=== Conference Room ===")
	conference := swaig.NewFunctionResult("Setting up daily standup meeting").
		JoinRoom("daily_standup_room").
		SetMetadata(map[string]any{
			"meeting_type":   "daily_standup",
			"participant_id": "user_123",
			"role":           "scrum_master",
		}).
		UpdateGlobalData(map[string]any{
			"meeting_active": true,
			"room_name":      "daily_standup_room",
		}).
		Say("You have joined the daily standup meeting")
	printResult(conference)

	// ---- Basic SIP REFER ----
	fmt.Println("=== Basic SIP REFER ===")
	sipRefer := swaig.NewFunctionResult("Transferring your call to support").
		Say("Please hold while I transfer you to our support specialist").
		SipRefer("sip:support@company.com")
	printResult(sipRefer)

	// ---- Advanced SIP REFER with metadata ----
	fmt.Println("=== Advanced SIP REFER ===")
	advSip := swaig.NewFunctionResult("Transferring to technical support").
		SetMetadata(map[string]any{
			"transfer_type":   "technical_support",
			"priority":        "high",
			"original_caller": "+15551234567",
		}).
		Say("Connecting you to our senior technical specialist").
		SipRefer("sip:tech-specialist@pbx.company.com:5060").
		UpdateGlobalData(map[string]any{
			"transfer_completed":   true,
			"transfer_destination": "tech-specialist@pbx.company.com",
		})
	printResult(advSip)

	// ---- Join conference ----
	fmt.Println("=== Join Conference ===")
	joinConf := swaig.NewFunctionResult("Joining team conference").
		JoinConference("daily_standup", nil).
		Say("Welcome to the daily standup conference")
	printResult(joinConf)

	fmt.Println("Key Features Demonstrated:")
	fmt.Println("- RELAY room joining for multi-party communication")
	fmt.Println("- SIP REFER for call transfers in SIP environments")
	fmt.Println("- Audio conferences with JoinConference")
	fmt.Println("- Metadata tracking for participants and transfers")
	fmt.Println("- Global data management for workflow state")
}

func printResult(fr *swaig.FunctionResult) {
	data, _ := json.MarshalIndent(fr.ToMap(), "", "  ")
	fmt.Println(string(data))
	fmt.Println()
}
