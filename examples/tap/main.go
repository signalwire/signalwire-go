//go:build ignore

// Example: tap
//
// TAP configuration for media monitoring. Demonstrates using the Tap
// and StopTap helpers on SwaigFunctionResult to stream call audio
// over WebSocket and RTP for real-time analysis or monitoring.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	fmt.Println("Tap Examples")
	fmt.Println()

	// ---- Basic WebSocket tap ----
	fmt.Println("=== Basic WebSocket Tap ===")
	wsTap := swaig.NewFunctionResult("Starting call monitoring").
		Tap("wss://monitoring.company.com/audio-stream", "", "", "", 0, "").
		Say("Call monitoring is now active")
	printResult(wsTap)

	// ---- Basic RTP tap ----
	fmt.Println("=== Basic RTP Tap ===")
	rtpTap := swaig.NewFunctionResult("Starting RTP monitoring").
		Tap("rtp://192.168.1.100:5004", "", "", "", 0, "").
		UpdateGlobalData(map[string]any{"rtp_monitoring": true})
	printResult(rtpTap)

	// ---- Advanced compliance monitoring ----
	fmt.Println("=== Advanced Compliance Monitoring ===")
	compliance := swaig.NewFunctionResult("Setting up compliance monitoring").
		Tap("wss://compliance.company.com/secure-stream",
			"compliance_tap_001", "both", "PCMA", 0, "").
		SetMetadata(map[string]any{
			"compliance_session": true,
			"agent_id":           "agent_123",
			"recording_purpose":  "regulatory_compliance",
		}).
		Say("This call may be monitored for compliance purposes")
	printResult(compliance)

	// ---- Customer service quality monitoring ----
	fmt.Println("=== Customer Service Monitoring ===")
	csMonitor := swaig.NewFunctionResult("Initialising quality monitoring").
		Tap("wss://quality.company.com/cs-monitoring",
			"cs_quality_monitor", "speak", "", 0, "").
		UpdateGlobalData(map[string]any{
			"quality_monitoring": true,
		}).
		Say("Welcome to customer service. How can I help you today?")
	printResult(csMonitor)

	// ---- Stop specific tap ----
	fmt.Println("=== Stop Specific Tap ===")
	stopTap := swaig.NewFunctionResult("Ending compliance monitoring").
		StopTap("compliance_tap_001").
		UpdateGlobalData(map[string]any{"compliance_session": false}).
		Say("Compliance monitoring has been deactivated")
	printResult(stopTap)

	// ---- Stop most recent tap ----
	fmt.Println("=== Stop Most Recent Tap ===")
	stopRecent := swaig.NewFunctionResult("Ending monitoring session").
		StopTap("").
		Say("Call monitoring has been stopped")
	printResult(stopRecent)

	fmt.Println("Key Features Demonstrated:")
	fmt.Println("- WebSocket and RTP tap streaming")
	fmt.Println("- Compliance and quality monitoring")
	fmt.Println("- Direction control (both, speak)")
	fmt.Println("- Codec selection (PCMA, PCMU)")
	fmt.Println("- Stop tap by control ID")
	fmt.Println("- Metadata and global data tracking")
}

func printResult(fr *swaig.FunctionResult) {
	data, _ := json.MarshalIndent(fr.ToMap(), "", "  ")
	fmt.Println(string(data))
	fmt.Println()
}
