// Package builtin provides the light built-in agent skills (datetime, math,
// joke, weather, web search, wikipedia, google maps, datasphere, etc.) — every
// builtin skill that carries no external dependencies beyond the standard
// library and the SDK itself. Each skill self-registers with the skills
// registry via its init(), so blank-importing this package makes the light set
// available to AgentBase.AddSkill by name.
//
// Dependency-carrying skills live in their own sub-packages (e.g.
// builtin/spider, which pulls goquery/htmlquery) so importing the light set
// never compiles those in. The pkg/skills/all umbrella blank-imports this
// package plus every such sub-package for the one-step "everything" path.
package builtin
