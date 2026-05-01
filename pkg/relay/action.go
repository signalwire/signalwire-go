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
	controlID string
	call      *Call
	done      chan struct{}
	result    *RelayEvent
	completed bool
	mu        sync.Mutex
	// terminalEvent is the calling.* event_type that resolves this
	// action. Empty string means "any event for this control_id resolves
	// it" — the legacy behavior. Specific actions set this so e.g. a
	// PlayAndCollect action listens on calling.call.collect, not
	// calling.call.play. Mirrors Python's terminal_event constructor
	// arg at relay/call.py:Action.__init__.
	terminalEvent string
	// terminalStates is the optional set of state values that resolve
	// the action. Empty means "any state on the terminal_event resolves".
	// PlayAction uses {"finished","error"} per Python — non-finished
	// states like "playing"/"paused" don't resolve.
	terminalStates map[string]bool
	onCompleted    func(*RelayEvent)
}

// newAction creates a new Action tied to a specific call and control ID.
// Matches Python's Action(call, control_id, terminal_event="", terminal_states=()).
func newAction(call *Call, controlID string) *Action {
	return &Action{
		controlID: controlID,
		call:      call,
		done:      make(chan struct{}),
	}
}

// matchesTerminal returns true when an event for this action's
// control_id should resolve it. Mirrors Python Action._matches_terminal:
// the event_type must equal terminalEvent (when set), and the state
// must be in terminalStates (when set).
func (a *Action) matchesTerminal(event *RelayEvent) bool {
	if a.terminalEvent != "" && event.EventType != a.terminalEvent {
		return false
	}
	if len(a.terminalStates) > 0 {
		state := event.GetString("state")
		if !a.terminalStates[state] {
			return false
		}
	}
	return true
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

// newPlayAction creates a new PlayAction. PlayAction listens on
// calling.call.play and resolves on the {finished,error} terminal
// states — mirrors Python PlayAction(call, control_id) at
// relay/call.py:94-96.
func newPlayAction(call *Call, controlID string) *PlayAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallPlay
	a.terminalStates = map[string]bool{"finished": true, "error": true}
	return &PlayAction{Action: a}
}

// Stop sends calling.play.stop to halt the active play operation.
func (pa *PlayAction) Stop() error {
	if pa.call == nil || pa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := pa.call.client.execute("calling.play.stop", map[string]any{
		"node_id":    pa.call.nodeID,
		"call_id":    pa.call.callID,
		"control_id": pa.controlID,
	})
	return err
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

// newRecordAction creates a new RecordAction. RecordAction listens on
// calling.call.record and resolves on {finished,error}.
func newRecordAction(call *Call, controlID string) *RecordAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallRecord
	a.terminalStates = map[string]bool{"finished": true, "error": true}
	return &RecordAction{Action: a}
}

// Stop sends calling.record.stop to halt the active recording.
func (ra *RecordAction) Stop() error {
	if ra.call == nil || ra.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ra.call.client.execute("calling.record.stop", map[string]any{
		"node_id":    ra.call.nodeID,
		"call_id":    ra.call.callID,
		"control_id": ra.controlID,
	})
	return err
}

// Pause pauses the active recording. An optional behavior string may be
// provided (e.g. "silence" or "skip") to control how the gap is handled.
// Pass no argument — or "" — to omit behavior, matching Python's
// pause(behavior: Optional[str] = None) signature.
func (ra *RecordAction) Pause(behavior ...string) error {
	if ra.call == nil || ra.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	params := map[string]any{
		"node_id":    ra.call.nodeID,
		"call_id":    ra.call.callID,
		"control_id": ra.controlID,
	}
	if len(behavior) > 0 && behavior[0] != "" {
		params["behavior"] = behavior[0]
	}
	_, err := ra.call.client.execute("calling.record.pause", params)
	return err
}

// Resume resumes a paused recording.
func (ra *RecordAction) Resume() error {
	if ra.call == nil || ra.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ra.call.client.execute("calling.record.resume", map[string]any{
		"node_id":    ra.call.nodeID,
		"call_id":    ra.call.callID,
		"control_id": ra.controlID,
	})
	return err
}

// DetectAction represents a long-running detect operation (e.g. machine detection).
type DetectAction struct {
	*Action
}

// newDetectAction creates a new DetectAction. DetectAction listens on
// calling.call.detect and — per the Python gotcha at
// RELAY_IMPLEMENTATION_GUIDE — resolves on the FIRST event carrying a
// non-empty `detect` payload, not on a state(finished). The "detect
// payload present" check happens in resolveAction's matchesTerminal
// branch. Empty terminalStates means any matching event_type resolves.
func newDetectAction(call *Call, controlID string) *DetectAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallDetect
	return &DetectAction{Action: a}
}

// Stop sends calling.detect.stop to halt the active detect operation.
func (da *DetectAction) Stop() error {
	if da.call == nil || da.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := da.call.client.execute("calling.detect.stop", map[string]any{
		"node_id":    da.call.nodeID,
		"call_id":    da.call.callID,
		"control_id": da.controlID,
	})
	return err
}

// CollectAction represents a play-and-collect operation.
type CollectAction struct {
	*Action
}

// newCollectAction creates a new CollectAction (used by play_and_collect).
// CollectAction listens on calling.call.collect — NOT calling.call.play —
// per RELAY_IMPLEMENTATION_GUIDE.md and Python relay/call.py:154-156:
// a play(finished) earlier in the timeline must NOT resolve this action.
func newCollectAction(call *Call, controlID string) *CollectAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallCollect
	return &CollectAction{Action: a}
}

// Stop sends calling.play_and_collect.stop to halt the play-and-collect operation.
func (ca *CollectAction) Stop() error {
	if ca.call == nil || ca.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ca.call.client.execute("calling.play_and_collect.stop", map[string]any{
		"node_id":    ca.call.nodeID,
		"call_id":    ca.call.callID,
		"control_id": ca.controlID,
	})
	return err
}

// Volume adjusts the playback volume by the given dB offset during a
// play-and-collect operation.
func (ca *CollectAction) Volume(db float64) error {
	if ca.call == nil || ca.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ca.call.client.execute("calling.play_and_collect.volume", map[string]any{
		"node_id":    ca.call.nodeID,
		"call_id":    ca.call.callID,
		"control_id": ca.controlID,
		"volume":     db,
	})
	return err
}

// StartInputTimers starts the initial_timeout timer on an active collect,
// equivalent to Python's CollectAction.start_input_timers().
func (ca *CollectAction) StartInputTimers() error {
	if ca.call == nil || ca.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ca.call.client.execute("calling.collect.start_input_timers", map[string]any{
		"node_id":    ca.call.nodeID,
		"call_id":    ca.call.callID,
		"control_id": ca.controlID,
	})
	return err
}

// StandaloneCollectAction represents a standalone collect (without play).
type StandaloneCollectAction struct {
	*Action
}

// newStandaloneCollectAction creates a new StandaloneCollectAction.
// Listens on calling.call.collect; resolves on any matching event for
// this control_id (no state filter — per Python).
func newStandaloneCollectAction(call *Call, controlID string) *StandaloneCollectAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallCollect
	return &StandaloneCollectAction{Action: a}
}

// Stop sends calling.collect.stop to halt the standalone collect operation.
func (sca *StandaloneCollectAction) Stop() error {
	if sca.call == nil || sca.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := sca.call.client.execute("calling.collect.stop", map[string]any{
		"node_id":    sca.call.nodeID,
		"call_id":    sca.call.callID,
		"control_id": sca.controlID,
	})
	return err
}

// StartInputTimers starts the initial_timeout timer on an active standalone
// collect, equivalent to Python's StandaloneCollectAction.start_input_timers().
func (sca *StandaloneCollectAction) StartInputTimers() error {
	if sca.call == nil || sca.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := sca.call.client.execute("calling.collect.start_input_timers", map[string]any{
		"node_id":    sca.call.nodeID,
		"call_id":    sca.call.callID,
		"control_id": sca.controlID,
	})
	return err
}

// FaxAction represents a long-running fax send/receive operation.
// methodPrefix distinguishes "send_fax" from "receive_fax" and is used to
// build the operation-specific stop command (e.g. "calling.send_fax.stop").
type FaxAction struct {
	*Action
	methodPrefix string
}

// newFaxAction creates a new FaxAction for the given method prefix
// ("send_fax" or "receive_fax"), matching Python's FaxAction(call,
// control_id, method_prefix). FaxAction listens on calling.call.fax.
func newFaxAction(call *Call, controlID string, methodPrefix string) *FaxAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallFax
	return &FaxAction{
		Action:       a,
		methodPrefix: methodPrefix,
	}
}

// Stop sends "calling.{methodPrefix}.stop" (e.g. "calling.send_fax.stop" or
// "calling.receive_fax.stop") to halt the active fax operation.
func (fa *FaxAction) Stop() error {
	if fa.call == nil || fa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := fa.call.client.execute("calling."+fa.methodPrefix+".stop", map[string]any{
		"node_id":    fa.call.nodeID,
		"call_id":    fa.call.callID,
		"control_id": fa.controlID,
	})
	return err
}

// TapAction represents a long-running tap operation.
type TapAction struct {
	*Action
}

// newTapAction creates a new TapAction. Listens on calling.call.tap.
func newTapAction(call *Call, controlID string) *TapAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallTap
	return &TapAction{Action: a}
}

// Stop sends calling.tap.stop to halt the active tap operation.
func (ta *TapAction) Stop() error {
	if ta.call == nil || ta.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ta.call.client.execute("calling.tap.stop", map[string]any{
		"node_id":    ta.call.nodeID,
		"call_id":    ta.call.callID,
		"control_id": ta.controlID,
	})
	return err
}

// StreamAction represents a long-running media stream operation.
type StreamAction struct {
	*Action
}

// newStreamAction creates a new StreamAction. Listens on calling.call.stream.
func newStreamAction(call *Call, controlID string) *StreamAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallStream
	return &StreamAction{Action: a}
}

// Stop sends calling.stream.stop to halt the active stream operation.
func (sa *StreamAction) Stop() error {
	if sa.call == nil || sa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := sa.call.client.execute("calling.stream.stop", map[string]any{
		"node_id":    sa.call.nodeID,
		"call_id":    sa.call.callID,
		"control_id": sa.controlID,
	})
	return err
}

// PayAction represents a long-running pay operation.
type PayAction struct {
	*Action
}

// newPayAction creates a new PayAction. Listens on calling.call.pay.
func newPayAction(call *Call, controlID string) *PayAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallPay
	return &PayAction{Action: a}
}

// Stop sends calling.pay.stop to halt the active pay operation.
func (pa *PayAction) Stop() error {
	if pa.call == nil || pa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := pa.call.client.execute("calling.pay.stop", map[string]any{
		"node_id":    pa.call.nodeID,
		"call_id":    pa.call.callID,
		"control_id": pa.controlID,
	})
	return err
}

// TranscribeAction represents a long-running transcription operation.
type TranscribeAction struct {
	*Action
}

// newTranscribeAction creates a new TranscribeAction.
// Listens on calling.call.transcribe.
func newTranscribeAction(call *Call, controlID string) *TranscribeAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallTranscribe
	return &TranscribeAction{Action: a}
}

// Stop sends calling.transcribe.stop to halt the active transcription.
func (ta *TranscribeAction) Stop() error {
	if ta.call == nil || ta.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := ta.call.client.execute("calling.transcribe.stop", map[string]any{
		"node_id":    ta.call.nodeID,
		"call_id":    ta.call.callID,
		"control_id": ta.controlID,
	})
	return err
}

// AIAction represents a long-running AI operation on a call.
type AIAction struct {
	*Action
}

// newAIAction creates a new AIAction. Listens on calling.call.ai.
func newAIAction(call *Call, controlID string) *AIAction {
	a := newAction(call, controlID)
	a.terminalEvent = EventCallingCallAI
	return &AIAction{Action: a}
}

// Stop sends calling.ai.stop to halt the active AI session.
func (aa *AIAction) Stop() error {
	if aa.call == nil || aa.call.client == nil {
		return fmt.Errorf("action not associated with a call or client")
	}
	_, err := aa.call.client.execute("calling.ai.stop", map[string]any{
		"node_id":    aa.call.nodeID,
		"call_id":    aa.call.callID,
		"control_id": aa.controlID,
	})
	return err
}
