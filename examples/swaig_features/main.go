//go:build ignore

// Example: swaig_features
//
// SwaigFunctionResult actions showcase. Demonstrates the full range of
// FunctionResult helpers: Say, Hangup, Hold, Connect, SendSms,
// UpdateGlobalData, SetMetadata, PlayBackgroundFile, ToggleFunctions,
// SetEndOfSpeechTimeout, SwitchContext, and more.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	fmt.Println("SwaigFunctionResult Actions Showcase")
	fmt.Println()

	// ---- Say action ----
	fmt.Println("=== Say ===")
	sayResult := swaig.NewFunctionResult("Processing complete").
		Say("Here are the results of your query")
	printResult(sayResult)

	// ---- Hangup ----
	fmt.Println("=== Hangup ===")
	hangupResult := swaig.NewFunctionResult("Thank you for calling. Goodbye!").
		Hangup()
	printResult(hangupResult)

	// ---- Hold ----
	fmt.Println("=== Hold ===")
	holdResult := swaig.NewFunctionResult("Placing you on hold while I check").
		Hold(60)
	printResult(holdResult)

	// ---- Connect (transfer call) ----
	fmt.Println("=== Connect ===")
	connectResult := swaig.NewFunctionResult("Transferring you to sales").
		Connect("+15551001001", true, "+15559990000")
	printResult(connectResult)

	// ---- SendSms ----
	fmt.Println("=== SendSms ===")
	smsResult := swaig.NewFunctionResult("Sending confirmation text").
		SendSms("+15551234567", "+15559990000", "Your appointment is confirmed for 3pm.", nil, nil)
	printResult(smsResult)

	// ---- UpdateGlobalData ----
	fmt.Println("=== UpdateGlobalData ===")
	globalResult := swaig.NewFunctionResult("Cart updated").
		UpdateGlobalData(map[string]any{
			"cart_items": []string{"Widget A", "Widget B"},
			"cart_total": 49.98,
		})
	printResult(globalResult)

	// ---- SetMetadata ----
	fmt.Println("=== SetMetadata ===")
	metaResult := swaig.NewFunctionResult("Order processed").
		SetMetadata(map[string]any{
			"order_id": "ORD-12345",
			"status":   "confirmed",
		})
	printResult(metaResult)

	// ---- PlayBackgroundFile ----
	fmt.Println("=== PlayBackgroundFile ===")
	bgResult := swaig.NewFunctionResult("Playing hold music").
		PlayBackgroundFile("https://cdn.example.com/music/hold.mp3", false)
	printResult(bgResult)

	// ---- StopBackgroundFile ----
	fmt.Println("=== StopBackgroundFile ===")
	stopBg := swaig.NewFunctionResult("Stopping music").
		StopBackgroundFile()
	printResult(stopBg)

	// ---- ToggleFunctions ----
	fmt.Println("=== ToggleFunctions ===")
	toggleResult := swaig.NewFunctionResult("Adjusting available tools").
		ToggleFunctions([]map[string]any{
			{"function": "transfer_call", "active": true},
			{"function": "send_sms", "active": false},
		})
	printResult(toggleResult)

	// ---- SetEndOfSpeechTimeout ----
	fmt.Println("=== SetEndOfSpeechTimeout ===")
	timeoutResult := swaig.NewFunctionResult("Adjusted speech detection").
		SetEndOfSpeechTimeout(500)
	printResult(timeoutResult)

	// ---- SwitchContext ----
	fmt.Println("=== SwitchContext ===")
	ctxResult := swaig.NewFunctionResult("Switching to billing context").
		SwitchContext("You are now a billing specialist. Help with payment questions.", "", false, false, false)
	printResult(ctxResult)

	// ---- Method chaining ----
	fmt.Println("=== Combined Actions ===")
	combined := swaig.NewFunctionResult("Order confirmed and scheduled").
		Say("Your order has been confirmed").
		UpdateGlobalData(map[string]any{"order_status": "confirmed"}).
		SendSms("+15551234567", "+15559990000", "Order confirmed! Delivery scheduled.", nil, nil).
		SetMetadata(map[string]any{"confirmation_sent": true})
	printResult(combined)

	fmt.Println("Key Features Demonstrated:")
	fmt.Println("- All major FunctionResult action types")
	fmt.Println("- Method chaining for combining multiple actions")
	fmt.Println("- Call control (hangup, hold, connect, SMS)")
	fmt.Println("- State management (global data, metadata)")
	fmt.Println("- Media control (say, background file)")
	fmt.Println("- AI configuration (timeout, toggle functions, context switch)")
}

func printResult(fr *swaig.FunctionResult) {
	data, _ := json.MarshalIndent(fr.ToMap(), "", "  ")
	fmt.Println(string(data))
	fmt.Println()
}
