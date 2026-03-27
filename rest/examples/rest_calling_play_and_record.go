//go:build ignore

// Example: Control an active call with media operations (play, record, transcribe, denoise).
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

func main() {
	client, err := rest.NewRestClient("", "", "")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	// 1. Dial an outbound call
	fmt.Println("Dialing outbound call...")
	callID := "demo-call-id"
	call, err := client.Calling.Dial(map[string]any{
		"from": "+15559876543",
		"to":   "+15551234567",
		"url":  "https://example.com/call-handler",
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Dial failed (expected in demo): %d\n", restErr.StatusCode)
		} else {
			fmt.Printf("  Dial failed: %v\n", err)
		}
	} else {
		if id, ok := call["id"].(string); ok {
			callID = id
		}
		fmt.Printf("  Call initiated: %s\n", callID)
	}

	// 2. Play TTS audio
	fmt.Println("\nPlaying TTS on call...")
	_, err = client.Calling.Play(callID, map[string]any{
		"play": []map[string]any{{"type": "tts", "text": "Welcome to SignalWire."}},
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Play failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Play started")
	}

	// 3. Pause, resume, adjust volume, stop playback
	fmt.Println("\nControlling playback...")
	for _, op := range []struct {
		label string
		fn    func() (map[string]any, error)
	}{
		{"Pause", func() (map[string]any, error) { return client.Calling.PlayPause(callID, nil) }},
		{"Resume", func() (map[string]any, error) { return client.Calling.PlayResume(callID, nil) }},
		{"Volume +2dB", func() (map[string]any, error) {
			return client.Calling.PlayVolume(callID, map[string]any{"volume": 2.0})
		}},
		{"Stop", func() (map[string]any, error) { return client.Calling.PlayStop(callID, nil) }},
	} {
		_, err := op.fn()
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  %s: failed (%d)\n", op.label, restErr.StatusCode)
			}
		} else {
			fmt.Printf("  %s: OK\n", op.label)
		}
	}

	// 4. Record the call
	fmt.Println("\nRecording call...")
	_, err = client.Calling.Record(callID, map[string]any{"beep": true, "format": "mp3"})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Record failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Recording started")
	}

	// 5. Pause, resume, stop recording
	fmt.Println("\nControlling recording...")
	for _, op := range []struct {
		label string
		fn    func() (map[string]any, error)
	}{
		{"Pause", func() (map[string]any, error) { return client.Calling.RecordPause(callID, nil) }},
		{"Resume", func() (map[string]any, error) { return client.Calling.RecordResume(callID, nil) }},
		{"Stop", func() (map[string]any, error) { return client.Calling.RecordStop(callID, nil) }},
	} {
		_, err := op.fn()
		if err != nil {
			if restErr, ok := err.(*rest.SignalWireRestError); ok {
				fmt.Printf("  %s: failed (%d)\n", op.label, restErr.StatusCode)
			}
		} else {
			fmt.Printf("  %s: OK\n", op.label)
		}
	}

	// 6. Transcribe the call
	fmt.Println("\nTranscribing call...")
	_, err = client.Calling.Transcribe(callID, map[string]any{"language": "en-US"})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Transcribe failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Transcription started")
		client.Calling.TranscribeStop(callID, nil)
		fmt.Println("  Transcription stopped")
	}

	// 7. Denoise the call
	fmt.Println("\nEnabling denoise...")
	_, err = client.Calling.Denoise(callID, nil)
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Denoise failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Denoise started")
		client.Calling.DenoiseStop(callID, nil)
		fmt.Println("  Denoise stopped")
	}

	// 8. End the call
	fmt.Println("\nEnding call...")
	_, err = client.Calling.End(callID, map[string]any{"reason": "hangup"})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  End call failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Call ended")
	}
}
