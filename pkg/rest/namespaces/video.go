// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ---------- VideoRooms ----------

// VideoRooms provides video room management with stream sub-resources.
type VideoRooms struct {
	*CrudResource
}

// ListStreams lists streams for a video room.
func (r *VideoRooms) ListStreams(roomID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(roomID, "streams"), params)
}

// CreateStream creates a stream for a video room.
func (r *VideoRooms) CreateStream(roomID string, kwargs map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(roomID, "streams"), kwargs, nil)
}

// ---------- VideoRoomTokens ----------

// VideoRoomTokens provides video room token generation.
type VideoRoomTokens struct {
	Resource
}

// Create creates a video room token.
func (r *VideoRoomTokens) Create(kwargs map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, kwargs, nil)
}

// ---------- VideoRoomSessions ----------

// VideoRoomSessions provides video room session management.
type VideoRoomSessions struct {
	Resource
}

// List lists all room sessions.
func (r *VideoRoomSessions) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a specific room session.
func (r *VideoRoomSessions) Get(sessionID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sessionID), nil)
}

// ListEvents lists events for a room session.
func (r *VideoRoomSessions) ListEvents(sessionID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sessionID, "events"), params)
}

// ListMembers lists members in a room session.
func (r *VideoRoomSessions) ListMembers(sessionID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sessionID, "members"), params)
}

// ListRecordings lists recordings for a room session.
func (r *VideoRoomSessions) ListRecordings(sessionID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sessionID, "recordings"), params)
}

// ---------- VideoRoomRecordings ----------

// VideoRoomRecordings provides video room recording management.
type VideoRoomRecordings struct {
	Resource
}

// List lists all room recordings.
func (r *VideoRoomRecordings) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a specific room recording.
func (r *VideoRoomRecordings) Get(recordingID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(recordingID), nil)
}

// Delete removes a room recording. It returns the parsed response body
// (or an empty map for 204 No Content) and any error.
func (r *VideoRoomRecordings) Delete(recordingID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(recordingID))
}

// ListEvents lists events for a room recording.
func (r *VideoRoomRecordings) ListEvents(recordingID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(recordingID, "events"), params)
}

// ---------- VideoConferences ----------

// VideoConferences provides video conference management with tokens and streams.
type VideoConferences struct {
	*CrudResource
}

// ListConferenceTokens lists tokens for a conference.
func (r *VideoConferences) ListConferenceTokens(conferenceID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(conferenceID, "conference_tokens"), params)
}

// ListStreams lists streams for a conference.
func (r *VideoConferences) ListStreams(conferenceID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(conferenceID, "streams"), params)
}

// CreateStream creates a stream for a conference.
func (r *VideoConferences) CreateStream(conferenceID string, kwargs map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(conferenceID, "streams"), kwargs, nil)
}

// ---------- VideoConferenceTokens ----------

// VideoConferenceTokens provides video conference token management.
type VideoConferenceTokens struct {
	Resource
}

// Get retrieves a conference token.
func (r *VideoConferenceTokens) Get(tokenID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(tokenID), nil)
}

// Reset resets a conference token.
func (r *VideoConferenceTokens) Reset(tokenID string) (map[string]any, error) {
	return r.HTTP.Post(r.Path(tokenID, "reset"), nil, nil)
}

// ---------- VideoStreams ----------

// VideoStreams provides video stream management.
type VideoStreams struct {
	Resource
}

// Get retrieves a video stream.
func (r *VideoStreams) Get(streamID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(streamID), nil)
}

// Update modifies a video stream.
func (r *VideoStreams) Update(streamID string, kwargs map[string]any) (map[string]any, error) {
	return r.HTTP.Put(r.Path(streamID), kwargs)
}

// Delete removes a video stream. It returns the parsed response body
// (or an empty map for 204 No Content) and any error.
func (r *VideoStreams) Delete(streamID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(streamID))
}

// ---------- VideoNamespace ----------

// VideoNamespace groups all Video API resources.
type VideoNamespace struct {
	Rooms            *VideoRooms
	RoomTokens       *VideoRoomTokens
	RoomSessions     *VideoRoomSessions
	RoomRecordings   *VideoRoomRecordings
	Conferences      *VideoConferences
	ConferenceTokens *VideoConferenceTokens
	Streams          *VideoStreams
}

// NewVideoNamespace creates a new VideoNamespace with all sub-resources initialized.
func NewVideoNamespace(client HTTPClient) *VideoNamespace {
	base := "/api/video"
	return &VideoNamespace{
		Rooms:            &VideoRooms{NewCrudResourcePUT(client, base+"/rooms")},
		RoomTokens:       &VideoRoomTokens{Resource{HTTP: client, Base: base + "/room_tokens"}},
		RoomSessions:     &VideoRoomSessions{Resource{HTTP: client, Base: base + "/room_sessions"}},
		RoomRecordings:   &VideoRoomRecordings{Resource{HTTP: client, Base: base + "/room_recordings"}},
		Conferences:      &VideoConferences{NewCrudResourcePUT(client, base+"/conferences")},
		ConferenceTokens: &VideoConferenceTokens{Resource{HTTP: client, Base: base + "/conference_tokens"}},
		Streams:          &VideoStreams{Resource{HTTP: client, Base: base + "/streams"}},
	}
}
