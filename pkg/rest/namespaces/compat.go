// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

import "fmt"

// ---------- CompatAccounts ----------

// CompatAccounts provides compat account/subproject management.
type CompatAccounts struct {
	Resource
}

// List lists all compat accounts.
func (r *CompatAccounts) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Create creates a new compat account.
func (r *CompatAccounts) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data, nil)
}

// Get retrieves a compat account by SID.
func (r *CompatAccounts) Get(sid string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sid), nil)
}

// Update updates a compat account by SID.
func (r *CompatAccounts) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// ---------- CompatCalls ----------

// CompatCalls provides compat call management with recording and stream sub-resources.
type CompatCalls struct {
	*CrudResource
}

// Update updates a call (uses POST per Twilio compat).
func (r *CompatCalls) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// StartRecording starts recording on a call.
func (r *CompatCalls) StartRecording(callSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(callSID, "Recordings"), data, nil)
}

// UpdateRecording updates a recording on a call.
func (r *CompatCalls) UpdateRecording(callSID, recordingSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(callSID, "Recordings", recordingSID), data, nil)
}

// StartStream starts a stream on a call.
func (r *CompatCalls) StartStream(callSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(callSID, "Streams"), data, nil)
}

// StopStream stops a stream on a call.
func (r *CompatCalls) StopStream(callSID, streamSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(callSID, "Streams", streamSID), data, nil)
}

// ---------- CompatMessages ----------

// CompatMessages provides compat message management with media sub-resources.
type CompatMessages struct {
	*CrudResource
}

// Update updates a message (uses POST per Twilio compat).
func (r *CompatMessages) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// ListMedia lists media for a message.
func (r *CompatMessages) ListMedia(messageSID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(messageSID, "Media"), params)
}

// GetMedia retrieves a specific media item from a message.
func (r *CompatMessages) GetMedia(messageSID, mediaSID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(messageSID, "Media", mediaSID), nil)
}

// DeleteMedia deletes a media item from a message.
func (r *CompatMessages) DeleteMedia(messageSID, mediaSID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(messageSID, "Media", mediaSID))
}

// ---------- CompatFaxes ----------

// CompatFaxes provides compat fax management with media sub-resources.
type CompatFaxes struct {
	*CrudResource
}

// Update updates a fax (uses POST per Twilio compat).
func (r *CompatFaxes) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// ListMedia lists media for a fax.
func (r *CompatFaxes) ListMedia(faxSID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(faxSID, "Media"), params)
}

// GetMedia retrieves a specific media item from a fax.
func (r *CompatFaxes) GetMedia(faxSID, mediaSID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(faxSID, "Media", mediaSID), nil)
}

// DeleteMedia deletes a media item from a fax.
func (r *CompatFaxes) DeleteMedia(faxSID, mediaSID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(faxSID, "Media", mediaSID))
}

// ---------- CompatConferences ----------

// CompatConferences provides compat conference management with participants,
// recordings, and streams.
type CompatConferences struct {
	Resource
}

// List lists all conferences.
func (r *CompatConferences) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a conference by SID.
func (r *CompatConferences) Get(sid string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sid), nil)
}

// Update updates a conference.
func (r *CompatConferences) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// Participants

// ListParticipants lists participants in a conference.
func (r *CompatConferences) ListParticipants(conferenceSID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(conferenceSID, "Participants"), params)
}

// GetParticipant retrieves a participant from a conference.
func (r *CompatConferences) GetParticipant(conferenceSID, callSID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(conferenceSID, "Participants", callSID), nil)
}

// UpdateParticipant updates a participant in a conference.
func (r *CompatConferences) UpdateParticipant(conferenceSID, callSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(conferenceSID, "Participants", callSID), data, nil)
}

// RemoveParticipant removes a participant from a conference.
func (r *CompatConferences) RemoveParticipant(conferenceSID, callSID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(conferenceSID, "Participants", callSID))
}

// Conference recordings

// ListRecordings lists recordings for a conference.
func (r *CompatConferences) ListRecordings(conferenceSID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(conferenceSID, "Recordings"), params)
}

// GetRecording retrieves a recording from a conference.
func (r *CompatConferences) GetRecording(conferenceSID, recordingSID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(conferenceSID, "Recordings", recordingSID), nil)
}

// UpdateRecording updates a recording in a conference.
func (r *CompatConferences) UpdateRecording(conferenceSID, recordingSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(conferenceSID, "Recordings", recordingSID), data, nil)
}

// DeleteRecording deletes a recording from a conference.
func (r *CompatConferences) DeleteRecording(conferenceSID, recordingSID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(conferenceSID, "Recordings", recordingSID))
}

// Conference streams

// StartStream starts a stream on a conference.
func (r *CompatConferences) StartStream(conferenceSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(conferenceSID, "Streams"), data, nil)
}

// StopStream stops a stream on a conference.
func (r *CompatConferences) StopStream(conferenceSID, streamSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(conferenceSID, "Streams", streamSID), data, nil)
}

// ---------- CompatPhoneNumbers ----------

// CompatPhoneNumbers provides compat phone number management.
type CompatPhoneNumbers struct {
	Resource
	availableBase string
}

// List lists all incoming phone numbers.
func (r *CompatPhoneNumbers) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Purchase purchases a phone number.
func (r *CompatPhoneNumbers) Purchase(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data, nil)
}

// Get retrieves an incoming phone number by SID.
func (r *CompatPhoneNumbers) Get(sid string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sid), nil)
}

// Update updates an incoming phone number.
func (r *CompatPhoneNumbers) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// Delete releases an incoming phone number.
func (r *CompatPhoneNumbers) Delete(sid string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(sid))
}

// ImportNumber imports an externally-hosted phone number.
func (r *CompatPhoneNumbers) ImportNumber(data map[string]any) (map[string]any, error) {
	path := r.Base[:len(r.Base)-len("/IncomingPhoneNumbers")] + "/ImportedPhoneNumbers"
	return r.HTTP.Post(path, data, nil)
}

// ListAvailableCountries lists countries with available numbers.
func (r *CompatPhoneNumbers) ListAvailableCountries(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.availableBase, params)
}

// SearchLocal searches for available local numbers in a country.
func (r *CompatPhoneNumbers) SearchLocal(country string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(fmt.Sprintf("%s/%s/Local", r.availableBase, country), params)
}

// SearchTollFree searches for available toll-free numbers in a country.
func (r *CompatPhoneNumbers) SearchTollFree(country string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(fmt.Sprintf("%s/%s/TollFree", r.availableBase, country), params)
}

// ---------- CompatApplications ----------

// CompatApplications provides compat application management.
type CompatApplications struct {
	*CrudResource
}

// Update updates an application (uses POST per Twilio compat).
func (r *CompatApplications) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// ---------- CompatLamlBins ----------

// CompatLamlBins provides compat cXML/LaML script management.
type CompatLamlBins struct {
	*CrudResource
}

// Update updates a LaML bin (uses POST per Twilio compat).
func (r *CompatLamlBins) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// ---------- CompatQueues ----------

// CompatQueues provides compat queue management with members.
type CompatQueues struct {
	*CrudResource
}

// Update updates a queue (uses POST per Twilio compat).
func (r *CompatQueues) Update(sid string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(sid), data, nil)
}

// ListMembers lists members of a queue.
func (r *CompatQueues) ListMembers(queueSID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(queueSID, "Members"), params)
}

// GetMember retrieves a member from a queue.
func (r *CompatQueues) GetMember(queueSID, callSID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(queueSID, "Members", callSID), nil)
}

// DequeueMember dequeues a member from a queue.
func (r *CompatQueues) DequeueMember(queueSID, callSID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Path(queueSID, "Members", callSID), data, nil)
}

// ---------- CompatRecordings ----------

// CompatRecordings provides compat recording management.
type CompatRecordings struct {
	Resource
}

// List lists all recordings.
func (r *CompatRecordings) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a recording by SID.
func (r *CompatRecordings) Get(sid string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sid), nil)
}

// Delete removes a recording.
func (r *CompatRecordings) Delete(sid string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(sid))
}

// ---------- CompatTranscriptions ----------

// CompatTranscriptions provides compat transcription management.
type CompatTranscriptions struct {
	Resource
}

// List lists all transcriptions.
func (r *CompatTranscriptions) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a transcription by SID.
func (r *CompatTranscriptions) Get(sid string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(sid), nil)
}

// Delete removes a transcription.
func (r *CompatTranscriptions) Delete(sid string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(sid))
}

// ---------- CompatTokens ----------

// CompatTokens provides compat API token management.
type CompatTokens struct {
	Resource
}

// Create creates a new API token.
func (r *CompatTokens) Create(data map[string]any) (map[string]any, error) {
	return r.HTTP.Post(r.Base, data, nil)
}

// Update modifies an API token.
func (r *CompatTokens) Update(tokenID string, data map[string]any) (map[string]any, error) {
	return r.HTTP.Patch(r.Path(tokenID), data)
}

// Delete removes an API token.
func (r *CompatTokens) Delete(tokenID string) (map[string]any, error) {
	return r.HTTP.Delete(r.Path(tokenID))
}

// ---------- CompatNamespace ----------

// CompatNamespace provides the Twilio-compatible LAML API with AccountSid scoping.
type CompatNamespace struct {
	Accounts       *CompatAccounts
	Calls          *CompatCalls
	Messages       *CompatMessages
	Faxes          *CompatFaxes
	Conferences    *CompatConferences
	PhoneNumbers   *CompatPhoneNumbers
	Applications   *CompatApplications
	LamlBins       *CompatLamlBins
	Queues         *CompatQueues
	Recordings     *CompatRecordings
	Transcriptions *CompatTranscriptions
	Tokens         *CompatTokens
}

// NewCompatNamespace creates a new CompatNamespace with all sub-resources
// scoped to the given account SID.
func NewCompatNamespace(client HTTPClient, accountSID string) *CompatNamespace {
	base := fmt.Sprintf("/api/laml/2010-04-01/Accounts/%s", accountSID)
	incomingBase := base + "/IncomingPhoneNumbers"
	availableBase := base + "/AvailablePhoneNumbers"

	return &CompatNamespace{
		Accounts:       &CompatAccounts{Resource{HTTP: client, Base: "/api/laml/2010-04-01/Accounts"}},
		Calls:          &CompatCalls{NewCrudResource(client, base+"/Calls")},
		Messages:       &CompatMessages{NewCrudResource(client, base+"/Messages")},
		Faxes:          &CompatFaxes{NewCrudResource(client, base+"/Faxes")},
		Conferences:    &CompatConferences{Resource{HTTP: client, Base: base + "/Conferences"}},
		PhoneNumbers:   &CompatPhoneNumbers{Resource: Resource{HTTP: client, Base: incomingBase}, availableBase: availableBase},
		Applications:   &CompatApplications{NewCrudResource(client, base+"/Applications")},
		LamlBins:       &CompatLamlBins{NewCrudResource(client, base+"/LamlBins")},
		Queues:         &CompatQueues{NewCrudResource(client, base+"/Queues")},
		Recordings:     &CompatRecordings{Resource{HTTP: client, Base: base + "/Recordings"}},
		Transcriptions: &CompatTranscriptions{Resource{HTTP: client, Base: base + "/Transcriptions"}},
		Tokens:         &CompatTokens{Resource{HTTP: client, Base: base + "/tokens"}},
	}
}
