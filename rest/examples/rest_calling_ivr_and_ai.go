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
	"context"
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/rest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

const callID = "demo-call-id"

func safeCall(label string, fn func() (*namespaces.CallResponse, error)) *namespaces.CallResponse {
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
	safeCall("Collect", func() (*namespaces.CallResponse, error) {
		return client.Calling.Collect(context.Background(), callID, namespaces.CallingNamespaceCollectParams{Extras: map[string]any{
			"digits": map[string]any{"max": 4, "terminators": "#"},
			"play":   []map[string]any{{"type": "tts", "params": map[string]any{"text": "Enter your PIN followed by pound."}}},
		}})
	})
	safeCall("Start input timers", func() (*namespaces.CallResponse, error) {
		return client.Calling.CollectStartInputTimers(context.Background(), callID, namespaces.CallingNamespaceCollectStartInputTimersParams{})
	})
	safeCall("Stop collect", func() (*namespaces.CallResponse, error) {
		return client.Calling.CollectStop(context.Background(), callID, namespaces.CallingNamespaceCollectStopParams{})
	})

	// 2. Answering machine detection
	fmt.Println("\nDetecting answering machine...")
	safeCall("Detect", func() (*namespaces.CallResponse, error) {
		return client.Calling.Detect(context.Background(), callID, namespaces.CallingNamespaceDetectParams{Extras: map[string]any{"type": "machine"}})
	})
	safeCall("Stop detect", func() (*namespaces.CallResponse, error) {
		return client.Calling.DetectStop(context.Background(), callID, namespaces.CallingNamespaceDetectStopParams{})
	})

	// 3. AI operations
	fmt.Println("\nAI agent operations...")
	safeCall("AI message", func() (*namespaces.CallResponse, error) {
		return client.Calling.AIMessage(context.Background(), callID, namespaces.CallingNamespaceAIMessageParams{Extras: map[string]any{
			"message": "The customer wants to check their balance.",
		}})
	})
	safeCall("AI hold", func() (*namespaces.CallResponse, error) {
		return client.Calling.AIHold(context.Background(), callID, namespaces.CallingNamespaceAIHoldParams{})
	})
	safeCall("AI unhold", func() (*namespaces.CallResponse, error) {
		return client.Calling.AIUnhold(context.Background(), callID, namespaces.CallingNamespaceAIUnholdParams{})
	})
	safeCall("AI stop", func() (*namespaces.CallResponse, error) {
		return client.Calling.AIStop(context.Background(), callID, namespaces.CallingNamespaceAIStopParams{})
	})

	// 4. Live transcription and translation
	fmt.Println("\nLive transcription and translation...")
	safeCall("Live transcribe", func() (*namespaces.CallResponse, error) {
		return client.Calling.LiveTranscribe(context.Background(), callID, namespaces.CallingNamespaceLiveTranscribeParams{Extras: map[string]any{"language": "en-US"}})
	})
	safeCall("Live translate", func() (*namespaces.CallResponse, error) {
		return client.Calling.LiveTranslate(context.Background(), callID, namespaces.CallingNamespaceLiveTranslateParams{Extras: map[string]any{"language": "es"}})
	})

	// 5. Tap (media fork)
	fmt.Println("\nTap (media fork)...")
	safeCall("Tap start", func() (*namespaces.CallResponse, error) {
		return client.Calling.Tap(context.Background(), callID, namespaces.CallingNamespaceTapParams{Extras: map[string]any{
			"tap":    map[string]any{"type": "audio", "direction": "both"},
			"device": map[string]any{"type": "rtp", "addr": "192.168.1.100", "port": 9000},
		}})
	})
	safeCall("Tap stop", func() (*namespaces.CallResponse, error) {
		return client.Calling.TapStop(context.Background(), callID, namespaces.CallingNamespaceTapStopParams{})
	})

	// 6. Stream (WebSocket)
	fmt.Println("\nStream (WebSocket)...")
	safeCall("Stream start", func() (*namespaces.CallResponse, error) {
		return client.Calling.Stream(context.Background(), callID, namespaces.CallingNamespaceStreamParams{Extras: map[string]any{"url": "wss://example.com/audio-stream"}})
	})
	safeCall("Stream stop", func() (*namespaces.CallResponse, error) {
		return client.Calling.StreamStop(context.Background(), callID, namespaces.CallingNamespaceStreamStopParams{})
	})

	// 7. User event
	fmt.Println("\nSending user event...")
	safeCall("User event", func() (*namespaces.CallResponse, error) {
		return client.Calling.UserEvent(context.Background(), callID, namespaces.CallingNamespaceUserEventParams{Extras: map[string]any{
			"event_name": "agent_note",
			"data":       map[string]any{"note": "VIP caller"},
		}})
	})

	// 8. SIP refer
	fmt.Println("\nSIP refer...")
	safeCall("SIP refer", func() (*namespaces.CallResponse, error) {
		return client.Calling.Refer(context.Background(), callID, namespaces.CallingNamespaceReferParams{Extras: map[string]any{"sip_uri": "sip:support@example.com"}})
	})

	// 9. Fax stop commands
	fmt.Println("\nFax stop commands...")
	safeCall("Send fax stop", func() (*namespaces.CallResponse, error) {
		return client.Calling.SendFaxStop(context.Background(), callID, namespaces.CallingNamespaceSendFaxStopParams{})
	})
	safeCall("Receive fax stop", func() (*namespaces.CallResponse, error) {
		return client.Calling.ReceiveFaxStop(context.Background(), callID, namespaces.CallingNamespaceReceiveFaxStopParams{})
	})

	// 10. Transfer and disconnect
	fmt.Println("\nTransfer and disconnect...")
	safeCall("Transfer", func() (*namespaces.CallResponse, error) {
		return client.Calling.Transfer(context.Background(), callID, namespaces.CallingNamespaceTransferParams{Extras: map[string]any{"dest": "+15559999999"}})
	})
	safeCall("Update call", func() (*namespaces.CallResponse, error) {
		return client.Calling.Update(context.Background(), namespaces.CallingNamespaceUpdateParams{Extras: map[string]any{
			"id":       callID,
			"metadata": map[string]any{"priority": "high"},
		}})
	})
	safeCall("Disconnect", func() (*namespaces.CallResponse, error) {
		return client.Calling.Disconnect(context.Background(), callID, namespaces.CallingNamespaceDisconnectParams{})
	})
}
