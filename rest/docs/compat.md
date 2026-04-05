# Compatibility API

The Compatibility API provides a Twilio-compatible LAML surface at `/api/laml/2010-04-01`. All paths are scoped under `/Accounts/{AccountSid}`, where AccountSid is your project ID.

## Sub-Resources

| Attribute | Description |
|-----------|-------------|
| `Compat.Accounts` | Account/subproject management |
| `Compat.Calls` | Call management + recordings + streams |
| `Compat.Messages` | SMS/MMS management + media |
| `Compat.Faxes` | Fax management + media |
| `Compat.Conferences` | Conference management + participants + recordings + streams |
| `Compat.PhoneNumbers` | Incoming + available phone numbers |
| `Compat.Applications` | Application management |
| `Compat.LamlBins` | cXML/LaML script management |
| `Compat.Queues` | Queue management + members |
| `Compat.Recordings` | Recording management |
| `Compat.Transcriptions` | Transcription management |
| `Compat.Tokens` | API token management |

## Accounts

```go
// List accounts/subprojects
accounts, _ := client.Compat.Accounts.List(nil)

// Create a subproject
sub, _ := client.Compat.Accounts.Create(map[string]any{"FriendlyName": "My Subproject"})

// Get/update an account
account, _ := client.Compat.Accounts.Get("AC-sid")
client.Compat.Accounts.Update("AC-sid", map[string]any{"FriendlyName": "Updated"})
```

## Calls

```go
// List calls
calls, _ := client.Compat.Calls.List(map[string]string{"From": "+15551234567"})

// Create a call
call, _ := client.Compat.Calls.Create(map[string]any{
	"To":   "+15552222222",
	"From": "+15551111111",
	"Url":  "https://example.com/twiml",
})

// Get / update / delete
call, _ = client.Compat.Calls.Get("CA-sid")
client.Compat.Calls.Update("CA-sid", map[string]any{"Status": "completed"})
client.Compat.Calls.Delete("CA-sid")

// Start/update recording on a call
client.Compat.Calls.StartRecording("CA-sid", map[string]any{"channels": "dual"})
client.Compat.Calls.UpdateRecording("CA-sid", "RE-sid", map[string]any{"Status": "paused"})

// Start/stop stream on a call
client.Compat.Calls.StartStream("CA-sid", map[string]any{"Url": "wss://example.com/stream"})
client.Compat.Calls.StopStream("CA-sid", "ST-sid")
```

## Messages

```go
// Send an SMS
msg, _ := client.Compat.Messages.Create(map[string]any{
	"To":   "+15552222222",
	"From": "+15551111111",
	"Body": "Hello from SignalWire!",
})

// List / get / update / delete
messages, _ := client.Compat.Messages.List(nil)
msg, _ = client.Compat.Messages.Get("SM-sid")
client.Compat.Messages.Update("SM-sid", map[string]any{"Body": ""}) // redact
client.Compat.Messages.Delete("SM-sid")

// Media sub-resources
media, _ := client.Compat.Messages.ListMedia("SM-sid")
item, _ := client.Compat.Messages.GetMedia("SM-sid", "ME-sid")
client.Compat.Messages.DeleteMedia("SM-sid", "ME-sid")
```

## Faxes

```go
// Send a fax
fax, _ := client.Compat.Faxes.Create(map[string]any{
	"MediaUrl": "https://example.com/doc.pdf",
	"To":       "+15552222222",
	"From":     "+15551111111",
})

// List / get / cancel / delete
faxes, _ := client.Compat.Faxes.List(nil)
fax, _ = client.Compat.Faxes.Get("FX-sid")
client.Compat.Faxes.Update("FX-sid", map[string]any{"Status": "canceled"})
client.Compat.Faxes.Delete("FX-sid")

// Media sub-resources
media, _ := client.Compat.Faxes.ListMedia("FX-sid")
item, _ := client.Compat.Faxes.GetMedia("FX-sid", "ME-sid")
client.Compat.Faxes.DeleteMedia("FX-sid", "ME-sid")
```

## Conferences

```go
// List / get / update
conferences, _ := client.Compat.Conferences.List(nil)
conf, _ := client.Compat.Conferences.Get("CF-sid")
client.Compat.Conferences.Update("CF-sid", map[string]any{"Status": "completed"})

// Participants
participants, _ := client.Compat.Conferences.ListParticipants("CF-sid")
p, _ := client.Compat.Conferences.GetParticipant("CF-sid", "CA-sid")
client.Compat.Conferences.UpdateParticipant("CF-sid", "CA-sid", map[string]any{"Muted": true})
client.Compat.Conferences.RemoveParticipant("CF-sid", "CA-sid")

// Conference recordings
recs, _ := client.Compat.Conferences.ListRecordings("CF-sid")
rec, _ := client.Compat.Conferences.GetRecording("CF-sid", "RE-sid")
client.Compat.Conferences.UpdateRecording("CF-sid", "RE-sid", map[string]any{"Status": "stopped"})
client.Compat.Conferences.DeleteRecording("CF-sid", "RE-sid")

// Conference streams
client.Compat.Conferences.StartStream("CF-sid", map[string]any{"Url": "wss://example.com/stream"})
client.Compat.Conferences.StopStream("CF-sid", "ST-sid")
```

## Phone Numbers

```go
// List purchased numbers
numbers, _ := client.Compat.PhoneNumbers.List(nil)

// Search available numbers
local, _ := client.Compat.PhoneNumbers.SearchLocal("US", map[string]string{"AreaCode": "512"})
tollFree, _ := client.Compat.PhoneNumbers.SearchTollFree("US", nil)
countries, _ := client.Compat.PhoneNumbers.ListAvailableCountries()

// Purchase / get / update / release
num, _ := client.Compat.PhoneNumbers.Purchase(map[string]any{"PhoneNumber": "+15551234567"})
num, _ = client.Compat.PhoneNumbers.Get("PN-sid")
client.Compat.PhoneNumbers.Update("PN-sid", map[string]any{"VoiceUrl": "https://example.com/voice"})
client.Compat.PhoneNumbers.Delete("PN-sid")

// Import external number
client.Compat.PhoneNumbers.ImportNumber(map[string]any{"PhoneNumber": "+15559999999"})
```

## Applications

```go
apps, _ := client.Compat.Applications.List(nil)
app, _ := client.Compat.Applications.Create(map[string]any{
	"FriendlyName": "My App",
	"VoiceUrl":     "https://example.com/voice",
})
app, _ = client.Compat.Applications.Get("AP-sid")
client.Compat.Applications.Update("AP-sid", map[string]any{"VoiceUrl": "https://example.com/new-voice"})
client.Compat.Applications.Delete("AP-sid")
```

## LaML Bins (cXML Scripts)

```go
bins, _ := client.Compat.LamlBins.List(nil)
b, _ := client.Compat.LamlBins.Create(map[string]any{
	"Name":     "Greeting",
	"Contents": "<Response><Say>Hello</Say></Response>",
})
b, _ = client.Compat.LamlBins.Get("LB-sid")
client.Compat.LamlBins.Update("LB-sid", map[string]any{
	"Contents": "<Response><Say>Updated</Say></Response>",
})
client.Compat.LamlBins.Delete("LB-sid")
```

## Queues

```go
queues, _ := client.Compat.Queues.List(nil)
q, _ := client.Compat.Queues.Create(map[string]any{"FriendlyName": "Support", "MaxSize": 100})
q, _ = client.Compat.Queues.Get("QU-sid")
client.Compat.Queues.Update("QU-sid", map[string]any{"MaxSize": 200})
client.Compat.Queues.Delete("QU-sid")

// Members
members, _ := client.Compat.Queues.ListMembers("QU-sid")
member, _ := client.Compat.Queues.GetMember("QU-sid", "CA-sid")
client.Compat.Queues.DequeueMember("QU-sid", "CA-sid", map[string]any{
	"Url": "https://example.com/dequeue",
})
```

## Recordings & Transcriptions

```go
// Recordings
recs, _ := client.Compat.Recordings.List(nil)
rec, _ := client.Compat.Recordings.Get("RE-sid")
client.Compat.Recordings.Delete("RE-sid")

// Transcriptions
txns, _ := client.Compat.Transcriptions.List(nil)
txn, _ := client.Compat.Transcriptions.Get("TR-sid")
client.Compat.Transcriptions.Delete("TR-sid")
```

## Tokens

```go
token, _ := client.Compat.Tokens.Create(map[string]any{
	"name":        "my-token",
	"permissions": []string{"calling", "messaging"},
})
client.Compat.Tokens.Update("token-id", map[string]any{"name": "renamed"})
client.Compat.Tokens.Delete("token-id")
```
