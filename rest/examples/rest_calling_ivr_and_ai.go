//go:build ignore

// Example: IVR input collection, AI operations, and advanced call control.
//
// NOTE: These commands require an active call. The callID used here is
// illustrative -- in production you would obtain it from a dial response or
// inbound call event.
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

const callID = "demo-call-id"

func safeCall(label string, fn func() (map[string]any, error)) map[string]any {
	result, err := fn()
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  %s: failed (%d)\n", label, restErr.StatusCode)
		} else {
			fmt.Printf("  %s: failed (%v)\n", label, err)
		}
		return nil
	}
	fmt.Printf("  %s: OK\n", label)
	return result
}

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Collect DTMF input
	fmt.Println("Collecting DTMF input...")
	safeCall("Collect", func() (map[string]any, error) {
		return client.Calling.Collect(callID, map[string]any{
			"digits": map[string]any{"max": 4, "terminators": "#"},
			"play":   []map[string]any{{"type": "tts", "text": "Enter your PIN followed by pound."}},
		})
	})
	safeCall("Start input timers", func() (map[string]any, error) {
		return client.Calling.CollectStartInputTimers(callID, nil)
	})
	safeCall("Stop collect", func() (map[string]any, error) {
		return client.Calling.CollectStop(callID, nil)
	})

	// 2. Answering machine detection
	fmt.Println("\nDetecting answering machine...")
	safeCall("Detect", func() (map[string]any, error) {
		return client.Calling.Detect(callID, map[string]any{"type": "machine"})
	})
	safeCall("Stop detect", func() (map[string]any, error) {
		return client.Calling.DetectStop(callID, nil)
	})

	// 3. AI operations
	fmt.Println("\nAI agent operations...")
	safeCall("AI message", func() (map[string]any, error) {
		return client.Calling.AIMessage(callID, map[string]any{
			"message": "The customer wants to check their balance.",
		})
	})
	safeCall("AI hold", func() (map[string]any, error) {
		return client.Calling.AIHold(callID, nil)
	})
	safeCall("AI unhold", func() (map[string]any, error) {
		return client.Calling.AIUnhold(callID, nil)
	})
	safeCall("AI stop", func() (map[string]any, error) {
		return client.Calling.AIStop(callID, nil)
	})

	// 4. Live transcription and translation
	fmt.Println("\nLive transcription and translation...")
	safeCall("Live transcribe", func() (map[string]any, error) {
		return client.Calling.LiveTranscribe(callID, map[string]any{"language": "en-US"})
	})
	safeCall("Live translate", func() (map[string]any, error) {
		return client.Calling.LiveTranslate(callID, map[string]any{"language": "es"})
	})

	// 5. Tap (media fork)
	fmt.Println("\nTap (media fork)...")
	safeCall("Tap start", func() (map[string]any, error) {
		return client.Calling.Tap(callID, map[string]any{
			"tap":    map[string]any{"type": "audio", "direction": "both"},
			"device": map[string]any{"type": "rtp", "addr": "192.168.1.100", "port": 9000},
		})
	})
	safeCall("Tap stop", func() (map[string]any, error) {
		return client.Calling.TapStop(callID, nil)
	})

	// 6. Stream (WebSocket)
	fmt.Println("\nStream (WebSocket)...")
	safeCall("Stream start", func() (map[string]any, error) {
		return client.Calling.Stream(callID, map[string]any{"url": "wss://example.com/audio-stream"})
	})
	safeCall("Stream stop", func() (map[string]any, error) {
		return client.Calling.StreamStop(callID, nil)
	})

	// 7. User event
	fmt.Println("\nSending user event...")
	safeCall("User event", func() (map[string]any, error) {
		return client.Calling.UserEvent(callID, map[string]any{
			"event_name": "agent_note",
			"data":       map[string]any{"note": "VIP caller"},
		})
	})

	// 8. SIP refer
	fmt.Println("\nSIP refer...")
	safeCall("SIP refer", func() (map[string]any, error) {
		return client.Calling.Refer(callID, map[string]any{"sip_uri": "sip:support@example.com"})
	})

	// 9. Fax stop commands
	fmt.Println("\nFax stop commands...")
	safeCall("Send fax stop", func() (map[string]any, error) {
		return client.Calling.SendFaxStop(callID, nil)
	})
	safeCall("Receive fax stop", func() (map[string]any, error) {
		return client.Calling.ReceiveFaxStop(callID, nil)
	})

	// 10. Transfer and disconnect
	fmt.Println("\nTransfer and disconnect...")
	safeCall("Transfer", func() (map[string]any, error) {
		return client.Calling.Transfer(callID, map[string]any{"dest": "+15559999999"})
	})
	safeCall("Update call", func() (map[string]any, error) {
		return client.Calling.Update(map[string]any{
			"id":       callID,
			"metadata": map[string]any{"priority": "high"},
		})
	})
	safeCall("Disconnect", func() (map[string]any, error) {
		return client.Calling.Disconnect(callID, nil)
	})
}
