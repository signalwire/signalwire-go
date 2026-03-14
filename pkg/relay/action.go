package relay

import (
	"context"
	"fmt"
	"sync"
)

// Action represents a long-running operation on a call, such as playing
// audio, recording, or collecting input. Callers can Wait for completion,
// check status, or register a completion callback.
type Action struct {
	controlID   string
	call        *Call
	done        chan struct{}
	result      *RelayEvent
	completed   bool
	mu          sync.Mutex
	onCompleted func(*RelayEvent)
}

// newAction creates a new Action tied to a specific call and control ID.
func newAction(call *Call, controlID string) *Action {
	return &Action{
		controlID: controlID,
		call:      call,
		done:      make(chan struct{}),
	}
}

// ControlID returns the control identifier for this action.
func (a *Action) ControlID() string {
	return a.controlID
}

// Wait blocks until the action completes or the context is cancelled.
func (a *Action) Wait(ctx context.Context) (*RelayEvent, error) {
	select {
	case <-a.done:
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Stop sends a stop command to the SignalWire server for this action.
func (a *Action) Stop() error {
	if a.call == nil || a.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := a.call.client.execute("calling.stop", map[string]any{
		"node_id":    a.call.nodeID,
		"call_id":    a.call.callID,
		"control_id": a.controlID,
	})
	return err
}

// IsDone returns true if the action has completed.
func (a *Action) IsDone() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.completed
}

// Result returns the final event that resolved this action, or nil if pending.
func (a *Action) Result() *RelayEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.result
}

// Completed returns whether the action finished.
func (a *Action) Completed() bool {
	return a.IsDone()
}

// OnCompleted registers a callback invoked when the action completes.
func (a *Action) OnCompleted(fn func(*RelayEvent)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onCompleted = fn
}

// resolve is called internally when the server signals that the action
// has finished. It stores the result, marks the action as done, fires
// the completion callback, and closes the done channel.
func (a *Action) resolve(event *RelayEvent) {
	a.mu.Lock()
	if a.completed {
		a.mu.Unlock()
		return
	}
	a.result = event
	a.completed = true
	cb := a.onCompleted
	a.mu.Unlock()

	if cb != nil {
		cb(event)
	}
	close(a.done)
}

// ---------------------------------------------------------------------------
// Specialised action types
// ---------------------------------------------------------------------------

// PlayAction represents a long-running play operation with media controls.
type PlayAction struct {
	*Action
}

// newPlayAction creates a new PlayAction.
func newPlayAction(call *Call, controlID string) *PlayAction {
	return &PlayAction{Action: newAction(call, controlID)}
}

// Pause pauses the currently playing media.
func (pa *PlayAction) Pause() error {
	if pa.call == nil || pa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := pa.call.client.execute("calling.play.pause", map[string]any{
		"node_id":    pa.call.nodeID,
		"call_id":    pa.call.callID,
		"control_id": pa.controlID,
	})
	return err
}

// Resume resumes paused media playback.
func (pa *PlayAction) Resume() error {
	if pa.call == nil || pa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := pa.call.client.execute("calling.play.resume", map[string]any{
		"node_id":    pa.call.nodeID,
		"call_id":    pa.call.callID,
		"control_id": pa.controlID,
	})
	return err
}

// Volume adjusts playback volume by the given dB offset.
func (pa *PlayAction) Volume(db float64) error {
	if pa.call == nil || pa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := pa.call.client.execute("calling.play.volume", map[string]any{
		"node_id":    pa.call.nodeID,
		"call_id":    pa.call.callID,
		"control_id": pa.controlID,
		"volume":     db,
	})
	return err
}

// RecordAction represents a long-running record operation.
type RecordAction struct {
	*Action
}

// newRecordAction creates a new RecordAction.
func newRecordAction(call *Call, controlID string) *RecordAction {
	return &RecordAction{Action: newAction(call, controlID)}
}

// DetectAction represents a long-running detect operation (e.g. machine detection).
type DetectAction struct {
	*Action
}

// newDetectAction creates a new DetectAction.
func newDetectAction(call *Call, controlID string) *DetectAction {
	return &DetectAction{Action: newAction(call, controlID)}
}

// CollectAction represents a play-and-collect operation.
type CollectAction struct {
	*Action
}

// newCollectAction creates a new CollectAction.
func newCollectAction(call *Call, controlID string) *CollectAction {
	return &CollectAction{Action: newAction(call, controlID)}
}

// StandaloneCollectAction represents a standalone collect (without play).
type StandaloneCollectAction struct {
	*Action
}

// newStandaloneCollectAction creates a new StandaloneCollectAction.
func newStandaloneCollectAction(call *Call, controlID string) *StandaloneCollectAction {
	return &StandaloneCollectAction{Action: newAction(call, controlID)}
}

// FaxAction represents a long-running fax send/receive operation.
type FaxAction struct {
	*Action
}

// newFaxAction creates a new FaxAction.
func newFaxAction(call *Call, controlID string) *FaxAction {
	return &FaxAction{Action: newAction(call, controlID)}
}

// TapAction represents a long-running tap operation.
type TapAction struct {
	*Action
}

// newTapAction creates a new TapAction.
func newTapAction(call *Call, controlID string) *TapAction {
	return &TapAction{Action: newAction(call, controlID)}
}

// StreamAction represents a long-running media stream operation.
type StreamAction struct {
	*Action
}

// newStreamAction creates a new StreamAction.
func newStreamAction(call *Call, controlID string) *StreamAction {
	return &StreamAction{Action: newAction(call, controlID)}
}

// PayAction represents a long-running pay operation.
type PayAction struct {
	*Action
}

// newPayAction creates a new PayAction.
func newPayAction(call *Call, controlID string) *PayAction {
	return &PayAction{Action: newAction(call, controlID)}
}

// TranscribeAction represents a long-running transcription operation.
type TranscribeAction struct {
	*Action
}

// newTranscribeAction creates a new TranscribeAction.
func newTranscribeAction(call *Call, controlID string) *TranscribeAction {
	return &TranscribeAction{Action: newAction(call, controlID)}
}

// AIAction represents a long-running AI operation on a call.
type AIAction struct {
	*Action
}

// newAIAction creates a new AIAction.
func newAIAction(call *Call, controlID string) *AIAction {
	return &AIAction{Action: newAction(call, controlID)}
}
