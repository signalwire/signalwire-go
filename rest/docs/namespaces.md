# All Namespaces

Reference for every namespace beyond Fabric, Calling, and Compat (which have their own pages).

## Phone Numbers

```go
// List your phone numbers
numbers, _ := client.PhoneNumbers.List(nil)
numbers, _ = client.PhoneNumbers.List(map[string]string{"name": "Main"})

// Search available numbers to purchase
available, _ := client.PhoneNumbers.Search(map[string]string{"area_code": "512", "number_type": "local"})

// Purchase a number
number, _ := client.PhoneNumbers.Create(map[string]any{"number": "+15551234567"})

// Get / update / release
number, _ = client.PhoneNumbers.Get("pn-uuid")
client.PhoneNumbers.Update("pn-uuid", map[string]any{"name": "Support Line"})
client.PhoneNumbers.Delete("pn-uuid")
```

## Addresses

```go
addresses, _ := client.Addresses.List(nil)
address, _ := client.Addresses.Create(map[string]any{
	"label": "Office", "street": "123 Main St", "city": "Austin", "state": "TX",
})
address, _ = client.Addresses.Get("addr-uuid")
client.Addresses.Delete("addr-uuid")
```

## Queues

```go
queues, _ := client.Queues.List(nil)
queue, _ := client.Queues.Create(map[string]any{"name": "Support"})
queue, _ = client.Queues.Get("q-uuid")
client.Queues.Update("q-uuid", map[string]any{"name": "VIP Support"})
client.Queues.Delete("q-uuid")

// Members
members, _ := client.Queues.ListMembers("q-uuid")
nextMember, _ := client.Queues.GetNextMember("q-uuid")
member, _ := client.Queues.GetMember("q-uuid", "member-uuid")
```

## Recordings

```go
recordings, _ := client.Recordings.List(nil)
recording, _ := client.Recordings.Get("rec-uuid")
client.Recordings.Delete("rec-uuid")
```

## Number Groups

```go
groups, _ := client.NumberGroups.List(nil)
group, _ := client.NumberGroups.Create(map[string]any{"name": "Marketing"})
group, _ = client.NumberGroups.Get("ng-uuid")
client.NumberGroups.Update("ng-uuid", map[string]any{"name": "Sales"})
client.NumberGroups.Delete("ng-uuid")

// Memberships
memberships, _ := client.NumberGroups.ListMemberships("ng-uuid")
client.NumberGroups.AddMembership("ng-uuid", "pn-uuid")
membership, _ := client.NumberGroups.GetMembership("mem-uuid")
client.NumberGroups.DeleteMembership("mem-uuid")
```

## Verified Caller IDs

```go
callers, _ := client.VerifiedCallers.List(nil)
caller, _ := client.VerifiedCallers.Create(map[string]any{
	"phone_number": "+15551234567", "name": "Office",
})
caller, _ = client.VerifiedCallers.Get("vc-uuid")
client.VerifiedCallers.Update("vc-uuid", map[string]any{"name": "Main Office"})
client.VerifiedCallers.Delete("vc-uuid")

// Verification flow
client.VerifiedCallers.RedialVerification("vc-uuid")
client.VerifiedCallers.SubmitVerification("vc-uuid", "123456")
```

## SIP Profile

Singleton resource -- no ID needed:

```go
profile, _ := client.SIPProfile.Get()
client.SIPProfile.Update(map[string]any{"username": "myproject", "password": "newsecret"})
```

## Phone Number Lookup

```go
info, _ := client.Lookup.PhoneNumber("+15551234567", nil)
info, _ = client.Lookup.PhoneNumber("+15551234567", map[string]string{"include": "carrier,cnam"})
```

Note: carrier and CNAM lookups are billable.

## Short Codes

```go
codes, _ := client.ShortCodes.List(nil)
code, _ := client.ShortCodes.Get("sc-uuid")
client.ShortCodes.Update("sc-uuid", map[string]any{"name": "Alerts"})
```

## Imported Phone Numbers

```go
client.ImportedNumbers.Create(map[string]any{"number": "+15559999999", "carrier": "external"})
```

## MFA (Multi-Factor Authentication)

```go
// Request a verification code via SMS
result, _ := client.MFA.SMS(map[string]any{
	"to":      "+15551234567",
	"from":    "+15559876543",
	"message": "Your code is {code}",
})
requestID, _ := result["id"].(string)

// Or via phone call
result, _ = client.MFA.Call(map[string]any{
	"to":   "+15551234567",
	"from": "+15559876543",
})

// Verify the code
result, _ = client.MFA.Verify(requestID, "123456")
```

## 10DLC Campaign Registry

```go
// Brands
brands, _ := client.Registry.Brands.List(nil)
brand, _ := client.Registry.Brands.Create(map[string]any{"name": "My Brand", "ein": "12-3456789"})
brand, _ = client.Registry.Brands.Get("brand-uuid")

// Campaigns under a brand
campaigns, _ := client.Registry.Brands.ListCampaigns("brand-uuid")
campaign, _ := client.Registry.Brands.CreateCampaign("brand-uuid", map[string]any{"description": "Alerts"})

// Campaign management
campaign, _ = client.Registry.Campaigns.Get("camp-uuid")
client.Registry.Campaigns.Update("camp-uuid", map[string]any{"description": "Updated alerts"})

// Number assignments
numbers, _ := client.Registry.Campaigns.ListNumbers("camp-uuid")
orders, _ := client.Registry.Campaigns.ListOrders("camp-uuid")
order, _ := client.Registry.Campaigns.CreateOrder("camp-uuid", map[string]any{
	"phone_number_ids": []string{"pn-1"},
})
order, _ = client.Registry.Orders.Get("order-uuid")
client.Registry.Numbers.Delete("number-assignment-uuid")
```

## Datasphere

```go
// Documents
docs, _ := client.Datasphere.Documents.List(nil)
doc, _ := client.Datasphere.Documents.Create(map[string]any{
	"url": "https://example.com/doc.pdf", "tags": []string{"support"},
})
doc, _ = client.Datasphere.Documents.Get("doc-uuid")
client.Datasphere.Documents.Update("doc-uuid", map[string]any{
	"tags": []string{"support", "billing"},
})
client.Datasphere.Documents.Delete("doc-uuid")

// Semantic search
results, _ := client.Datasphere.Documents.Search(map[string]any{
	"query_string": "How do I reset my password?",
	"tags":         []string{"support"},
	"count":        5,
})

// Chunks
chunks, _ := client.Datasphere.Documents.ListChunks("doc-uuid")
chunk, _ := client.Datasphere.Documents.GetChunk("doc-uuid", "chunk-uuid")
client.Datasphere.Documents.DeleteChunk("doc-uuid", "chunk-uuid")
```

## Video

```go
// Rooms
rooms, _ := client.Video.Rooms.List(nil)
room, _ := client.Video.Rooms.Create(map[string]any{"name": "standup", "max_members": 10})
room, _ = client.Video.Rooms.Get("room-uuid")
client.Video.Rooms.Update("room-uuid", map[string]any{"max_members": 20})
client.Video.Rooms.Delete("room-uuid")
client.Video.Rooms.ListStreams("room-uuid")
client.Video.Rooms.CreateStream("room-uuid", map[string]any{"url": "rtmp://example.com/live"})

// Room tokens
token, _ := client.Video.RoomTokens.Create(map[string]any{
	"room_name": "standup", "user_name": "alice",
})

// Room sessions
sessions, _ := client.Video.RoomSessions.List(map[string]string{"room_name": "standup"})
session, _ := client.Video.RoomSessions.Get("session-uuid")
events, _ := client.Video.RoomSessions.ListEvents("session-uuid")
members, _ := client.Video.RoomSessions.ListMembers("session-uuid")
recordings, _ := client.Video.RoomSessions.ListRecordings("session-uuid")

// Room recordings
recs, _ := client.Video.RoomRecordings.List(nil)
rec, _ := client.Video.RoomRecordings.Get("rec-uuid")
client.Video.RoomRecordings.Delete("rec-uuid")
events, _ = client.Video.RoomRecordings.ListEvents("rec-uuid")

// Conferences
confs, _ := client.Video.Conferences.List(nil)
conf, _ := client.Video.Conferences.Create(map[string]any{"name": "all-hands", "quality": "720p"})
conf, _ = client.Video.Conferences.Get("conf-uuid")
client.Video.Conferences.Update("conf-uuid", map[string]any{"quality": "1080p"})
client.Video.Conferences.Delete("conf-uuid")
tokens, _ := client.Video.Conferences.ListConferenceTokens("conf-uuid")
client.Video.Conferences.ListStreams("conf-uuid")
client.Video.Conferences.CreateStream("conf-uuid", map[string]any{"url": "rtmp://example.com/live"})

// Conference tokens
token, _ = client.Video.ConferenceTokens.Get("token-uuid")
client.Video.ConferenceTokens.Reset("token-uuid")

// Streams
stream, _ := client.Video.Streams.Get("stream-uuid")
client.Video.Streams.Update("stream-uuid", map[string]any{"url": "rtmp://example.com/new"})
client.Video.Streams.Delete("stream-uuid")
```

## Logs

All log endpoints are read-only.

```go
// Message logs
logs, _ := client.Logs.Messages.List(map[string]string{"include_deleted": "true"})
log, _ := client.Logs.Messages.Get("log-uuid")

// Voice logs (with events)
logs, _ = client.Logs.Voice.List(nil)
log, _ = client.Logs.Voice.Get("log-uuid")
events, _ := client.Logs.Voice.ListEvents("log-uuid")

// Fax logs
logs, _ = client.Logs.Fax.List(nil)
log, _ = client.Logs.Fax.Get("log-uuid")

// Conference logs
logs, _ = client.Logs.Conferences.List(nil)
```

## Project Tokens

```go
token, _ := client.Project.Tokens.Create(map[string]any{
	"name":        "ci-token",
	"permissions": []string{"calling", "messaging", "numbers"},
})
client.Project.Tokens.Update("token-uuid", map[string]any{"name": "renamed-token"})
client.Project.Tokens.Delete("token-uuid")
```

## PubSub Tokens

```go
token, _ := client.PubSub.CreateToken(map[string]any{
	"ttl": 60,
	"channels": []map[string]any{
		{"name": "updates", "read": true, "write": false},
	},
	"member_id": "user-123",
})
```

## Chat Tokens

```go
token, _ := client.Chat.CreateToken(map[string]any{
	"ttl": 60,
	"channels": []map[string]any{
		{"name": "support", "read": true, "write": true},
	},
	"member_id": "user-123",
})
```
