package skills

// SkillName is the closed set of built-in skill names as a defined string
// type with typed constants. It mirrors the PHP `SkillName` backed enum and
// gives Go callers editor autocompletion plus call-site typo checking: a bare
// string like "datetiem" only fails at runtime (on the server), whereas a
// mistyped constant fails to compile.
//
// AgentBase.AddSkill / RemoveSkill / HasSkill take SkillName. Because Go
// auto-converts untyped string-constant literals to a defined string type,
// every call site keeps working three ways:
//
//	agent.AddSkill(skills.SkillDatetime, nil)   // typed const — autocompleted
//	agent.AddSkill("datetime", nil)             // bare string literal still compiles
//	agent.AddSkill(skills.SkillName("custom"),  // open set: custom / 3rd-party skills
//	    map[string]any{...})
//
// SkillName is a string subtype, so its wire/JSON value is identical to the
// reference's bare `str` parameter — parity with the Python reference (which
// uses `str`) and with custom skills that aren't built in.
type SkillName string

// Built-in skill names. These are the canonical keys the builtin packages
// register via RegisterSkill(); the constant values must stay in lockstep
// with those registrations (see pkg/skills/builtin/*.go).
const (
	SkillAPINinjasTrivia    SkillName = "api_ninjas_trivia"
	SkillClaudeSkills       SkillName = "claude_skills"
	SkillCustomSkills       SkillName = "custom_skills"
	SkillDatasphere         SkillName = "datasphere"
	SkillDatasphereServerless SkillName = "datasphere_serverless"
	SkillDatetime           SkillName = "datetime"
	SkillGoogleMaps         SkillName = "google_maps"
	SkillInfoGatherer       SkillName = "info_gatherer"
	SkillJoke               SkillName = "joke"
	SkillMath               SkillName = "math"
	SkillMCPGateway         SkillName = "mcp_gateway"
	SkillNativeVectorSearch SkillName = "native_vector_search"
	SkillPlayBackgroundFile SkillName = "play_background_file"
	SkillSpider             SkillName = "spider"
	SkillSWMLTransfer       SkillName = "swml_transfer"
	SkillWeatherAPI         SkillName = "weather_api"
	SkillWebSearch          SkillName = "web_search"
	SkillWikipediaSearch    SkillName = "wikipedia_search"
)
