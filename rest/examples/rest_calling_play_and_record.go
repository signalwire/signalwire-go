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

	// 1. Dial an outbound call
	fmt.Println("Dialing outbound call...")
	callID := "demo-call-id"
	callURL := "https://example.com/call-handler"
	call, err := client.Calling.Dial(context.Background(), namespaces.CallingNamespaceDialParams{
		From: "+15559876543",
		To:   "+15551234567",
		URL:  &callURL,
	})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Dial failed (expected in demo): %d\n", restErr.StatusCode)
		} else {
			fmt.Printf("  Dial failed: %v\n", err)
		}
	} else {
		if m, ok := (*call).(map[string]any); ok {
			if id, ok := m["id"].(string); ok {
				callID = id
			}
		}
		fmt.Printf("  Call initiated: %s\n", callID)
	}

	// 2. Play TTS audio
	fmt.Println("\nPlaying TTS on call...")
	_, err = client.Calling.Play(context.Background(), callID, namespaces.CallingNamespacePlayParams{
		Play: []map[string]any{{"type": "tts", "text": "Welcome to SignalWire."}},
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
		fn    func() (*namespaces.CallResponse, error)
	}{
		{"Pause", func() (*namespaces.CallResponse, error) {
			return client.Calling.PlayPause(context.Background(), callID, namespaces.CallingNamespacePlayPauseParams{})
		}},
		{"Resume", func() (*namespaces.CallResponse, error) {
			return client.Calling.PlayResume(context.Background(), callID, namespaces.CallingNamespacePlayResumeParams{})
		}},
		{"Volume +2dB", func() (*namespaces.CallResponse, error) {
			return client.Calling.PlayVolume(context.Background(), callID, namespaces.CallingNamespacePlayVolumeParams{Volume: 2.0})
		}},
		{"Stop", func() (*namespaces.CallResponse, error) {
			return client.Calling.PlayStop(context.Background(), callID, namespaces.CallingNamespacePlayStopParams{})
		}},
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
	_, err = client.Calling.Record(context.Background(), callID, namespaces.CallingNamespaceRecordParams{Extras: map[string]any{"beep": true, "format": "mp3"}})
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
		fn    func() (*namespaces.CallResponse, error)
	}{
		{"Pause", func() (*namespaces.CallResponse, error) {
			return client.Calling.RecordPause(context.Background(), callID, namespaces.CallingNamespaceRecordPauseParams{})
		}},
		{"Resume", func() (*namespaces.CallResponse, error) {
			return client.Calling.RecordResume(context.Background(), callID, namespaces.CallingNamespaceRecordResumeParams{})
		}},
		{"Stop", func() (*namespaces.CallResponse, error) {
			return client.Calling.RecordStop(context.Background(), callID, namespaces.CallingNamespaceRecordStopParams{})
		}},
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
	_, err = client.Calling.Transcribe(context.Background(), callID, namespaces.CallingNamespaceTranscribeParams{Extras: map[string]any{"language": "en-US"}})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Transcribe failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Transcription started")
		client.Calling.TranscribeStop(context.Background(), callID, namespaces.CallingNamespaceTranscribeStopParams{})
		fmt.Println("  Transcription stopped")
	}

	// 7. Denoise the call
	fmt.Println("\nEnabling denoise...")
	_, err = client.Calling.Denoise(context.Background(), callID, namespaces.CallingNamespaceDenoiseParams{})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  Denoise failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Denoise started")
		client.Calling.DenoiseStop(context.Background(), callID, namespaces.CallingNamespaceDenoiseStopParams{})
		fmt.Println("  Denoise stopped")
	}

	// 8. End the call
	fmt.Println("\nEnding call...")
	hangupReason := namespaces.HangupReasonHangup
	_, err = client.Calling.End(context.Background(), callID, namespaces.CallingNamespaceEndParams{Reason: &hangupReason})
	if err != nil {
		if restErr, ok := err.(*rest.SignalWireRestError); ok {
			fmt.Printf("  End call failed (expected in demo): %d\n", restErr.StatusCode)
		}
	} else {
		fmt.Println("  Call ended")
	}
}
