# All Namespaces

Reference for every namespace beyond Fabric and Calling (which have their own pages).

The examples below import the resource-parameter structs from
`github.com/signalwire/signalwire-go/pkg/rest/namespaces` and use a small helper
to set optional pointer fields:

<!-- snippet: no-compile illustrative API signature (reference only) -->
```go
func ptr[T any](v T) *T { return &v }
```

<!-- snippet-setup -->
```go
import (
	"context"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/namespaces"
)

// Shared context the fragments below assume: a constructed REST client.
// (The `ptr` helper above is illustrative; runnable fragments take the address
// of a local variable instead.)
var client, err = rest.NewRestClient("project", "token", "space")

var (
	_ = client
	_ = err
	_ = namespaces.Uuid("")
	_ = context.Background
)
```

## Phone Numbers

```go
// List your phone numbers
numbers, err := client.PhoneNumbers.List(context.Background(), nil)
numbers, err = client.PhoneNumbers.List(context.Background(), map[string]string{"name": "Main"})

// Search available numbers to purchase
available, err := client.PhoneNumbers.Search(context.Background(), map[string]string{
	"areacode":   "512",
	"number_type": "local",
})

// Purchase a number
number, err := client.PhoneNumbers.Create(context.Background(), map[string]any{"number": "+15551234567"})

// Get / update / release
number, err = client.PhoneNumbers.Get(context.Background(), "pn-uuid")
_, err = client.PhoneNumbers.Update(context.Background(), "pn-uuid", map[string]any{"name": "Support Line"})
_, err = client.PhoneNumbers.Delete(context.Background(), "pn-uuid")

_, _, _ = numbers, available, number
```

## Addresses

```go
addresses, err := client.Addresses.List(context.Background(), nil)
address, err := client.Addresses.Create(context.Background(), namespaces.AddressesNamespaceCreateParams{
	Label:      "Office",
	StreetName: "Main St",
	City:       "Austin",
	State:      "TX",
})
address, err = client.Addresses.Get(context.Background(), "addr-uuid", nil)
_, err = client.Addresses.Delete(context.Background(), "addr-uuid")

_, _ = addresses, address
```

## Queues

```go
queues, err := client.Queues.List(context.Background(), nil)
queue, err := client.Queues.Create(context.Background(), map[string]any{"name": "Support"})
queue, err = client.Queues.Get(context.Background(), "q-uuid")
_, err = client.Queues.Update(context.Background(), "q-uuid", map[string]any{"name": "VIP Support"})
_, err = client.Queues.Delete(context.Background(), "q-uuid")

// Members
members, err := client.Queues.ListMembers(context.Background(), "q-uuid", nil)
nextMember, err := client.Queues.GetNextMember(context.Background(), "q-uuid", nil)
member, err := client.Queues.GetMember(context.Background(), "q-uuid", "member-uuid", nil)

_, _, _, _ = queues, queue, members, nextMember
_ = member
```

## Recordings

```go
recordings, err := client.Recordings.List(context.Background(), nil)
recording, err := client.Recordings.Get(context.Background(), "rec-uuid", nil)
_, err = client.Recordings.Delete(context.Background(), "rec-uuid")

_, _ = recordings, recording
```

## Number Groups

```go
groups, err := client.NumberGroups.List(context.Background(), nil)
group, err := client.NumberGroups.Create(context.Background(), map[string]any{"name": "Marketing"})
group, err = client.NumberGroups.Get(context.Background(), "ng-uuid")
_, err = client.NumberGroups.Update(context.Background(), "ng-uuid", map[string]any{"name": "Sales"})
_, err = client.NumberGroups.Delete(context.Background(), "ng-uuid")

// Memberships
memberships, err := client.NumberGroups.ListMemberships(context.Background(), "ng-uuid", nil)
_, err = client.NumberGroups.AddMembership(context.Background(), "ng-uuid", namespaces.NumberGroupsNamespaceAddMembershipParams{
	Extras: map[string]any{"phone_number_id": "pn-uuid"},
})
membership, err := client.NumberGroups.GetMembership(context.Background(), "mem-uuid", nil)
_, err = client.NumberGroups.DeleteMembership(context.Background(), "mem-uuid")

_, _, _ = groups, group, memberships
_ = membership
```

## Verified Caller IDs

```go
callers, err := client.VerifiedCallers.List(context.Background(), nil)
caller, err := client.VerifiedCallers.Create(context.Background(), map[string]any{
	"phone_number": "+15551234567",
	"name":         "Office",
})
caller, err = client.VerifiedCallers.Get(context.Background(), "vc-uuid")
_, err = client.VerifiedCallers.Update(context.Background(), "vc-uuid", map[string]any{"name": "Main Office"})
_, err = client.VerifiedCallers.Delete(context.Background(), "vc-uuid")

// Verification flow
_, err = client.VerifiedCallers.RedialVerification(context.Background(), "vc-uuid")
_, err = client.VerifiedCallers.SubmitVerification(context.Background(), "vc-uuid", namespaces.VerifiedCallersNamespaceSubmitVerificationParams{
	VerificationCode: "123456",
})

_, _ = callers, caller
```

## SIP Profile

Singleton resource -- no ID needed:

```go
profile, err := client.SIPProfile.Get(context.Background(), nil)
_, err = client.SIPProfile.Update(context.Background(), namespaces.SIPProfileNamespaceUpdateParams{
	Extras: map[string]any{"username": "myproject", "password": "newsecret"},
})

_ = profile
```

## Phone Number Lookup

```go
info, err := client.Lookup.PhoneNumber(context.Background(), "+15551234567", nil)
info, err = client.Lookup.PhoneNumber(context.Background(), "+15551234567", map[string]string{"include": "carrier,cnam"})

_ = info
```

Note: carrier and CNAM lookups are billable.

## Short Codes

```go
codes, err := client.ShortCodes.List(context.Background(), nil)
code, err := client.ShortCodes.Get(context.Background(), "sc-uuid", nil)
_, err = client.ShortCodes.Update(context.Background(), "sc-uuid", namespaces.ShortCodesNamespaceUpdateParams{Name: "Alerts"})

_, _ = codes, code
```

## Imported Phone Numbers

```go
_, err = client.ImportedNumbers.Create(context.Background(), namespaces.ImportedNumbersNamespaceCreateParams{
	Number:     "+15559999999",
	NumberType: "external",
})
```

## MFA (Multi-Factor Authentication)

```go
// Request a verification code via SMS
from := "+15559876543"
message := "Your code is {code}"
result, err := client.MFA.SMS(context.Background(), namespaces.MFANamespaceSMSParams{
	To:      "+15551234567",
	From:    &from,
	Message: &message,
})
requestID := string(result.ID)

// Or via phone call
result, err = client.MFA.Call(context.Background(), namespaces.MFANamespaceCallParams{
	To:   "+15551234567",
	From: &from,
})

// Verify the code
verified, err := client.MFA.Verify(context.Background(), requestID, namespaces.MFANamespaceVerifyParams{Token: "123456"})

_, _ = result, verified
```

## 10DLC Campaign Registry

```go
// Brands
brands, err := client.Registry.Brands.List(context.Background(), nil)
brand, err := client.Registry.Brands.Create(context.Background(), map[string]any{"name": "My Brand", "ein": "12-3456789"})
brand, err = client.Registry.Brands.Get(context.Background(), "brand-uuid", nil)

// Campaigns under a brand
campaigns, err := client.Registry.Brands.ListCampaigns(context.Background(), "brand-uuid", nil)
campaign, err := client.Registry.Brands.CreateCampaign(context.Background(), "brand-uuid", map[string]any{"description": "Alerts"})

// Campaign management
campaign, err = client.Registry.Campaigns.Get(context.Background(), "camp-uuid", nil)
_, err = client.Registry.Campaigns.Update(context.Background(), "camp-uuid", namespaces.RegistryCampaignsUpdateParams{
	Extras: map[string]any{"description": "Updated alerts"},
})

// Number assignments
numbers, err := client.Registry.Campaigns.ListNumbers(context.Background(), "camp-uuid", nil)
orders, err := client.Registry.Campaigns.ListOrders(context.Background(), "camp-uuid", nil)
order, err := client.Registry.Campaigns.CreateOrder(context.Background(), "camp-uuid", namespaces.RegistryCampaignsCreateOrderParams{
	PhoneNumbers: []string{"pn-1"},
})
order, err = client.Registry.Orders.Get(context.Background(), "order-uuid", nil)
_, err = client.Registry.Numbers.Delete(context.Background(), "number-assignment-uuid")

_, _, _ = brands, brand, campaigns
_, _, _ = campaign, numbers, orders
_ = order
```

## Datasphere

```go
// Documents
docs, err := client.Datasphere.Documents.List(context.Background(), nil)
doc, err := client.Datasphere.Documents.Create(context.Background(), map[string]any{
	"url":  "https://example.com/doc.pdf",
	"tags": []string{"support"},
})
doc, err = client.Datasphere.Documents.Get(context.Background(), "doc-uuid")
_, err = client.Datasphere.Documents.Update(context.Background(), "doc-uuid", map[string]any{"tags": []string{"support", "billing"}})
_, err = client.Datasphere.Documents.Delete(context.Background(), "doc-uuid")

// Semantic search
count := 5
results, err := client.Datasphere.Documents.Search(context.Background(), namespaces.DatasphereDocumentsSearchParams{
	QueryString: "How do I reset my password?",
	Tags:        []string{"support"},
	Count:       &count,
})

// Chunks
chunks, err := client.Datasphere.Documents.ListChunks(context.Background(), "doc-uuid", nil)
chunk, err := client.Datasphere.Documents.GetChunk(context.Background(), "doc-uuid", "chunk-uuid", nil)
_, err = client.Datasphere.Documents.DeleteChunk(context.Background(), "doc-uuid", "chunk-uuid")

_, _, _, _ = docs, doc, results, chunks
_ = chunk
```

## Video

```go
// Rooms
rooms, err := client.Video.Rooms.List(context.Background(), nil)
room, err := client.Video.Rooms.Create(context.Background(), map[string]any{"name": "standup", "max_members": 10})
room, err = client.Video.Rooms.Get(context.Background(), "room-uuid")
_, err = client.Video.Rooms.Update(context.Background(), "room-uuid", map[string]any{"max_members": 20})
_, err = client.Video.Rooms.Delete(context.Background(), "room-uuid")
_, err = client.Video.Rooms.ListStreams(context.Background(), "room-uuid", nil)
_, err = client.Video.Rooms.CreateStream(context.Background(), "room-uuid", namespaces.VideoRoomsCreateStreamParams{URL: "rtmp://example.com/live"})

// Room tokens
userName := "alice"
token, err := client.Video.RoomTokens.Create(context.Background(), namespaces.VideoRoomTokensCreateParams{
	RoomName: "standup",
	UserName: &userName,
})

// Room sessions
sessions, err := client.Video.RoomSessions.List(context.Background(), map[string]string{"room_name": "standup"})
session, err := client.Video.RoomSessions.Get(context.Background(), "session-uuid")
events, err := client.Video.RoomSessions.ListEvents(context.Background(), "session-uuid", nil)
members, err := client.Video.RoomSessions.ListMembers(context.Background(), "session-uuid", nil)
recordings, err := client.Video.RoomSessions.ListRecordings(context.Background(), "session-uuid", nil)

// Room recordings
recs, err := client.Video.RoomRecordings.List(context.Background(), nil)
rec, err := client.Video.RoomRecordings.Get(context.Background(), "rec-uuid", nil)
_, err = client.Video.RoomRecordings.Delete(context.Background(), "rec-uuid")
recEvents, err := client.Video.RoomRecordings.ListEvents(context.Background(), "rec-uuid", nil)
_ = recEvents

// Conferences
confs, err := client.Video.Conferences.List(context.Background(), nil)
conf, err := client.Video.Conferences.Create(context.Background(), map[string]any{"name": "all-hands", "quality": "720p"})
conf, err = client.Video.Conferences.Get(context.Background(), "conf-uuid")
_, err = client.Video.Conferences.Update(context.Background(), "conf-uuid", map[string]any{"quality": "1080p"})
_, err = client.Video.Conferences.Delete(context.Background(), "conf-uuid")
tokens, err := client.Video.Conferences.ListConferenceTokens(context.Background(), "conf-uuid", nil)
_, err = client.Video.Conferences.ListStreams(context.Background(), "conf-uuid", nil)
_, err = client.Video.Conferences.CreateStream(context.Background(), "conf-uuid", namespaces.VideoConferencesCreateStreamParams{URL: "rtmp://example.com/live"})

// Conference tokens
confToken, err := client.Video.ConferenceTokens.Get(context.Background(), "token-uuid", nil)
_ = confToken
_, err = client.Video.ConferenceTokens.Reset(context.Background(), "token-uuid")

// Streams
stream, err := client.Video.Streams.Get(context.Background(), "stream-uuid", nil)
_, err = client.Video.Streams.Update(context.Background(), "stream-uuid", namespaces.VideoStreamsUpdateParams{URL: "rtmp://example.com/new"})
_, err = client.Video.Streams.Delete(context.Background(), "stream-uuid")

_, _, _, _ = rooms, room, token, sessions
_, _, _, _ = session, events, members, recordings
_, _, _, _ = recs, rec, confs, conf
_, _ = tokens, stream
```

## Logs

All log endpoints are read-only.

```go
// Message logs
logs, err := client.Logs.Messages.List(context.Background(), map[string]string{"include_deleted": "true"})
log, err := client.Logs.Messages.Get(context.Background(), "log-uuid")

// Voice logs (with events)
voiceLogs, err := client.Logs.Voice.List(context.Background(), nil)
voiceLog, err := client.Logs.Voice.Get(context.Background(), "log-uuid")
events, err := client.Logs.Voice.ListEvents(context.Background(), "log-uuid", nil)

// Fax logs
faxLogs, err := client.Logs.Fax.List(context.Background(), nil)
faxLog, err := client.Logs.Fax.Get(context.Background(), "log-uuid")

// Conference logs
confLogs, err := client.Logs.Conferences.List(context.Background(), nil)

_, _, _, _ = logs, log, voiceLogs, voiceLog
_, _, _, _ = events, faxLogs, faxLog, confLogs
```

## Project Tokens

```go
token, err := client.Project.Tokens.Create(context.Background(), namespaces.ProjectTokensCreateParams{
	Name:        "ci-token",
	Permissions: []namespaces.TokenPermission{"calling", "messaging", "numbers"},
})
renamed := "renamed-token"
_, err = client.Project.Tokens.Update(context.Background(), "token-uuid", namespaces.ProjectTokensUpdateParams{Name: &renamed})
_, err = client.Project.Tokens.Delete(context.Background(), "token-uuid")

_ = token
```

## PubSub Tokens

```go
memberID := "user-123"
token, err := client.PubSub.CreateToken(context.Background(), namespaces.PubSubNamespaceCreateTokenParams{
	Ttl: 60,
	Channels: namespaces.PubSubChannels{
		"updates": map[string]any{"read": true, "write": false},
	},
	MemberID: &memberID,
})

_ = token
```

## Chat Tokens

```go
memberID := "user-123"
token, err := client.Chat.CreateToken(context.Background(), namespaces.ChatNamespaceCreateTokenParams{
	Ttl: 60,
	Channels: namespaces.ChatChannel{
		"support": map[string]any{"read": true, "write": true},
	},
	MemberID: &memberID,
})

_ = token
