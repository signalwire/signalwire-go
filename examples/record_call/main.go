//go:build ignore

// Example: record_call
//
// Call recording configuration. Demonstrates different recording setups
// using SwaigFunctionResult helpers: basic recording, advanced stereo
// recording, voicemail, and stop recording.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	fmt.Println("Record Call Examples")
	fmt.Println()

	// ---- Basic recording ----
	fmt.Println("=== Basic Recording ===")
	basic := swaig.NewFunctionResult("Starting basic call recording").
		RecordCall("", false, "mp3", "both", nil).
		Say("This call is now being recorded")
	printResult(basic)

	// ---- Advanced stereo recording ----
	fmt.Println("=== Advanced Stereo Recording ===")
	advanced := swaig.NewFunctionResult("Starting advanced call recording").
		RecordCall("support_call_001", true, "mp3", "both", nil).
		Say("This call is being recorded for quality and training purposes")
	printResult(advanced)

	// ---- Voicemail recording ----
	fmt.Println("=== Voicemail Recording ===")
	voicemail := swaig.NewFunctionResult("Please leave your message after the beep").
		RecordCall("voicemail_123", false, "wav", "speak", nil).
		SetEndOfSpeechTimeout(2000)
	printResult(voicemail)

	// ---- Stop recording ----
	fmt.Println("=== Stop Recording ===")
	stopRec := swaig.NewFunctionResult("Ending call recording").
		StopRecordCall("support_call_001").
		Say("Thank you for calling. Your feedback is important to us.")
	printResult(stopRec)

	// ---- Complete customer service workflow ----
	fmt.Println("=== Customer Service Workflow ===")
	startWorkflow := swaig.NewFunctionResult("Transferring you to a customer service agent").
		RecordCall("cs_transfer_001", true, "mp3", "both", nil).
		UpdateGlobalData(map[string]any{"recording_id": "cs_transfer_001"}).
		Say("Please hold while I connect you to an agent")
	printResult(startWorkflow)

	endWorkflow := swaig.NewFunctionResult("Call recording stopped").
		StopRecordCall("cs_transfer_001").
		RemoveGlobalData([]string{"recording_id"}).
		Say("Thank you for calling. Have a wonderful day!")
	printResult(endWorkflow)

	fmt.Println("Key Features Demonstrated:")
	fmt.Println("- Basic and advanced recording configurations")
	fmt.Println("- Stereo recording with format selection")
	fmt.Println("- Voicemail-style recording with direction control")
	fmt.Println("- Stop recording by control ID")
	fmt.Println("- Complete workflow with global data tracking")
}

func printResult(fr *swaig.FunctionResult) {
	data, _ := json.MarshalIndent(fr.ToMap(), "", "  ")
	fmt.Println(string(data))
	fmt.Println()
}
