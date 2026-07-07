# All Namespaces

Reference for every namespace beyond Fabric and Calling (which have their own pages).

The examples below import the resource-parameter structs from
`github.com/signalwire/signalwire-go/pkg/rest/namespaces` and use a small helper
to set optional pointer fields:

```go
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func floatPtr(f float64) *float64 { return &f }
```

## Phone Numbers

```go
// List your phone numbers
numbers, err := client.PhoneNumbers.List(nil)
numbers, err = client.PhoneNumbers.List(map[string]string{"name": "Main"})

// Search available numbers to purchase
available, err := client.PhoneNumbers.Search(map[string]string{
	"area_code":   "512",
	"number_type": "local",
})

// Purchase a number
number, err := client.PhoneNumbers.Create(map[string]any{"number": "+15551234567"})

// Get / update / release
number, err = client.PhoneNumbers.Get("pn-uuid")
_, err = client.PhoneNumbers.Update("pn-uuid", map[string]any{"name": "Support Line"})
_, err = client.PhoneNumbers.Delete("pn-uuid")
```

## Addresses

```go
addresses, err := client.Addresses.List(nil)
address, err := client.Addresses.Create(namespaces.AddressesNamespaceCreateParams{
	Label:      "Office",
	StreetName: "Main St",
	City:       "Austin",
	State:      "TX",
})
address, err = client.Addresses.Get("addr-uuid", nil)
_, err = client.Addresses.Delete("addr-uuid")
```

## Queues

```go
queues, err := client.Queues.List(nil)
queue, err := client.Queues.Create(map[string]any{"name": "Support"})
queue, err = client.Queues.Get("q-uuid")
_, err = client.Queues.Update("q-uuid", map[string]any{"name": "VIP Support"})
_, err = client.Queues.Delete("q-uuid")

// Members
members, err := client.Queues.ListMembers("q-uuid", nil)
nextMember, err := client.Queues.GetNextMember("q-uuid", nil)
member, err := client.Queues.GetMember("q-uuid", "member-uuid", nil)
```

## Recordings

```go
recordings, err := client.Recordings.List(nil)
recording, err := client.Recordings.Get("rec-uuid", nil)
_, err = client.Recordings.Delete("rec-uuid")
```

## Number Groups

```go
groups, err := client.NumberGroups.List(nil)
group, err := client.NumberGroups.Create(map[string]any{"name": "Marketing"})
group, err = client.NumberGroups.Get("ng-uuid")
_, err = client.NumberGroups.Update("ng-uuid", map[string]any{"name": "Sales"})
_, err = client.NumberGroups.Delete("ng-uuid")

// Memberships
memberships, err := client.NumberGroups.ListMemberships("ng-uuid", nil)
_, err = client.NumberGroups.AddMembership("ng-uuid", namespaces.NumberGroupsNamespaceAddMembershipParams{
	Extras: map[string]any{"phone_number_id": "pn-uuid"},
})
membership, err := client.NumberGroups.GetMembership("mem-uuid", nil)
_, err = client.NumberGroups.DeleteMembership("mem-uuid")
```

## Verified Caller IDs

```go
callers, err := client.VerifiedCallers.List(nil)
caller, err := client.VerifiedCallers.Create(map[string]any{
	"phone_number": "+15551234567",
	"name":         "Office",
})
caller, err = client.VerifiedCallers.Get("vc-uuid")
_, err = client.VerifiedCallers.Update("vc-uuid", map[string]any{"name": "Main Office"})
_, err = client.VerifiedCallers.Delete("vc-uuid")

// Verification flow
_, err = client.VerifiedCallers.RedialVerification("vc-uuid")
_, err = client.VerifiedCallers.SubmitVerification("vc-uuid", namespaces.VerifiedCallersNamespaceSubmitVerificationParams{
	VerificationCode: "123456",
})
```

## SIP Profile

Singleton resource -- no ID needed:

```go
profile, err := client.SIPProfile.Get(nil)
_, err = client.SIPProfile.Update(namespaces.SIPProfileNamespaceUpdateParams{
	Extras: map[string]any{"username": "myproject", "password": "newsecret"},
})
```

## Phone Number Lookup

```go
info, err := client.Lookup.PhoneNumber("+15551234567", nil)
info, err = client.Lookup.PhoneNumber("+15551234567", map[string]string{"include": "carrier,cnam"})
```

Note: carrier and CNAM lookups are billable.

## Short Codes

```go
codes, err := client.ShortCodes.List(nil)
code, err := client.ShortCodes.Get("sc-uuid", nil)
_, err = client.ShortCodes.Update("sc-uuid", namespaces.ShortCodesNamespaceUpdateParams{Name: "Alerts"})
```

## Imported Phone Numbers

```go
_, err := client.ImportedNumbers.Create(namespaces.ImportedNumbersNamespaceCreateParams{
	Number:     "+15559999999",
	NumberType: "external",
})
```

## MFA (Multi-Factor Authentication)

```go
// Request a verification code via SMS
result, err := client.MFA.SMS(namespaces.MFANamespaceSMSParams{
	To:      "+15551234567",
	From:    strPtr("+15559876543"),
	Message: strPtr("Your code is {code}"),
})
requestID := string(result.Id)

// Or via phone call
result, err = client.MFA.Call(namespaces.MFANamespaceCallParams{
	To:   "+15551234567",
	From: strPtr("+15559876543"),
})

// Verify the code
verified, err := client.MFA.Verify(requestID, namespaces.MFANamespaceVerifyParams{Token: "123456"})
```

## 10DLC Campaign Registry

```go
// Brands
brands, err := client.Registry.Brands.List(nil)
brand, err := client.Registry.Brands.Create(map[string]any{"name": "My Brand", "ein": "12-3456789"})
brand, err = client.Registry.Brands.Get("brand-uuid", nil)

// Campaigns under a brand
campaigns, err := client.Registry.Brands.ListCampaigns("brand-uuid", nil)
campaign, err := client.Registry.Brands.CreateCampaign("brand-uuid", map[string]any{"description": "Alerts"})

// Campaign management
campaign, err = client.Registry.Campaigns.Get("camp-uuid", nil)
_, err = client.Registry.Campaigns.Update("camp-uuid", namespaces.RegistryCampaignsUpdateParams{
	Extras: map[string]any{"description": "Updated alerts"},
})

// Number assignments
numbers, err := client.Registry.Campaigns.ListNumbers("camp-uuid", nil)
orders, err := client.Registry.Campaigns.ListOrders("camp-uuid", nil)
order, err := client.Registry.Campaigns.CreateOrder("camp-uuid", namespaces.RegistryCampaignsCreateOrderParams{
	PhoneNumbers: []string{"pn-1"},
})
order, err = client.Registry.Orders.Get("order-uuid", nil)
_, err = client.Registry.Numbers.Delete("number-assignment-uuid")
```

## Datasphere

```go
// Documents
docs, err := client.Datasphere.Documents.List(nil)
doc, err := client.Datasphere.Documents.Create(map[string]any{
	"url":  "https://example.com/doc.pdf",
	"tags": []string{"support"},
})
doc, err = client.Datasphere.Documents.Get("doc-uuid")
_, err = client.Datasphere.Documents.Update("doc-uuid", map[string]any{"tags": []string{"support", "billing"}})
_, err = client.Datasphere.Documents.Delete("doc-uuid")

// Semantic search
results, err := client.Datasphere.Documents.Search(namespaces.DatasphereDocumentsSearchParams{
	QueryString: "How do I reset my password?",
	Tags:        []string{"support"},
	Count:       intPtr(5),
})

// Chunks
chunks, err := client.Datasphere.Documents.ListChunks("doc-uuid", nil)
chunk, err := client.Datasphere.Documents.GetChunk("doc-uuid", "chunk-uuid", nil)
_, err = client.Datasphere.Documents.DeleteChunk("doc-uuid", "chunk-uuid")
```

## Video

```go
// Rooms
rooms, err := client.Video.Rooms.List(nil)
room, err := client.Video.Rooms.Create(map[string]any{"name": "standup", "max_members": 10})
room, err = client.Video.Rooms.Get("room-uuid")
_, err = client.Video.Rooms.Update("room-uuid", map[string]any{"max_members": 20})
_, err = client.Video.Rooms.Delete("room-uuid")
_, err = client.Video.Rooms.ListStreams("room-uuid", nil)
_, err = client.Video.Rooms.CreateStream("room-uuid", namespaces.VideoRoomsCreateStreamParams{Url: "rtmp://example.com/live"})

// Room tokens
token, err := client.Video.RoomTokens.Create(namespaces.VideoRoomTokensCreateParams{
	RoomName: "standup",
	UserName: strPtr("alice"),
})

// Room sessions
sessions, err := client.Video.RoomSessions.List(map[string]string{"room_name": "standup"})
session, err := client.Video.RoomSessions.Get("session-uuid")
events, err := client.Video.RoomSessions.ListEvents("session-uuid", nil)
members, err := client.Video.RoomSessions.ListMembers("session-uuid", nil)
recordings, err := client.Video.RoomSessions.ListRecordings("session-uuid", nil)

// Room recordings
recs, err := client.Video.RoomRecordings.List(nil)
rec, err := client.Video.RoomRecordings.Get("rec-uuid", nil)
_, err = client.Video.RoomRecordings.Delete("rec-uuid")
recEvents, err := client.Video.RoomRecordings.ListEvents("rec-uuid", nil)
_ = recEvents

// Conferences
confs, err := client.Video.Conferences.List(nil)
conf, err := client.Video.Conferences.Create(map[string]any{"name": "all-hands", "quality": "720p"})
conf, err = client.Video.Conferences.Get("conf-uuid")
_, err = client.Video.Conferences.Update("conf-uuid", map[string]any{"quality": "1080p"})
_, err = client.Video.Conferences.Delete("conf-uuid")
tokens, err := client.Video.Conferences.ListConferenceTokens("conf-uuid", nil)
_, err = client.Video.Conferences.ListStreams("conf-uuid", nil)
_, err = client.Video.Conferences.CreateStream("conf-uuid", namespaces.VideoConferencesCreateStreamParams{Url: "rtmp://example.com/live"})

// Conference tokens
confToken, err := client.Video.ConferenceTokens.Get("token-uuid", nil)
_ = confToken
_, err = client.Video.ConferenceTokens.Reset("token-uuid")

// Streams
stream, err := client.Video.Streams.Get("stream-uuid", nil)
_, err = client.Video.Streams.Update("stream-uuid", namespaces.VideoStreamsUpdateParams{Url: "rtmp://example.com/new"})
_, err = client.Video.Streams.Delete("stream-uuid")
```

## Logs

All log endpoints are read-only.

```go
// Message logs
logs, err := client.Logs.Messages.List(map[string]string{"include_deleted": "true"})
log, err := client.Logs.Messages.Get("log-uuid")

// Voice logs (with events)
voiceLogs, err := client.Logs.Voice.List(nil)
voiceLog, err := client.Logs.Voice.Get("log-uuid")
events, err := client.Logs.Voice.ListEvents("log-uuid", nil)

// Fax logs
faxLogs, err := client.Logs.Fax.List(nil)
faxLog, err := client.Logs.Fax.Get("log-uuid")

// Conference logs
confLogs, err := client.Logs.Conferences.List(nil)
```

## Project Tokens

```go
token, err := client.Project.Tokens.Create(namespaces.ProjectTokensCreateParams{
	Name:        "ci-token",
	Permissions: []namespaces.TokenPermission{"calling", "messaging", "numbers"},
})
_, err = client.Project.Tokens.Update("token-uuid", namespaces.ProjectTokensUpdateParams{Name: strPtr("renamed-token")})
_, err = client.Project.Tokens.Delete("token-uuid")
```

## PubSub Tokens

```go
token, err := client.PubSub.CreateToken(namespaces.PubSubNamespaceCreateTokenParams{
	Ttl: 60,
	Channels: namespaces.PubSubChannels{
		"updates": map[string]any{"read": true, "write": false},
	},
	MemberId: strPtr("user-123"),
})
```

## Chat Tokens

```go
token, err := client.Chat.CreateToken(namespaces.ChatNamespaceCreateTokenParams{
	Ttl: 60,
	Channels: namespaces.ChatChannel{
		"support": map[string]any{"read": true, "write": true},
	},
	MemberId: strPtr("user-123"),
})
```
