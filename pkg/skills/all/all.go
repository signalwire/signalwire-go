// Package all registers every built-in skill by blank-importing each skill
// package for its init()-time RegisterSkill side effect. This is the
// "everything works" convenience import: a consumer who wants the full skill
// set writes a single
//
//	import _ "github.com/signalwire/signalwire-go/pkg/skills/all"
//
// and every built-in skill — including the ones that carry heavier external
// dependencies (the spider skill pulls goquery/htmlquery/x/net) — is available
// to AgentBase.AddSkill by name.
//
// Why this exists: skills self-register in init(), and Go only runs a package's
// init() (and compiles in its dependencies) when that package is imported
// somewhere in the build. The light skills live in pkg/skills/builtin (zero
// external deps); the dependency-carrying skills live in their own
// sub-packages (e.g. pkg/skills/builtin/spider). Importing this umbrella pulls
// them all.
//
// To opt OUT of a heavy skill's dependencies, do NOT import this umbrella —
// instead blank-import only the skill sets you want (e.g.
// `import _ ".../pkg/skills/builtin"` for the light set alone). Then `go mod
// tidy` drops the unused skills' deps (goquery/htmlquery/x/net) from your build.
package all

import (
	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"        // 16 light skills (no external deps)
	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin/spider" // spider (goquery/htmlquery/x/net)
)
