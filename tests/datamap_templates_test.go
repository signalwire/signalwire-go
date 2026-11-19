package tests

// TestDataMapTemplates guards the DataMap template corrections of
// cloud-product issue 21197. It scans docs/*.md and the Go sources in
// examples/ and pkg/ (not _test.go files) for templates that the platform does
// not expand as written. A docs line that warns about one of these forms on
// purpose carries the marker below, which Markdown doesn't render.

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// warningMarker exempts a docs line from textRules.
const warningMarker = "<!-- datamap-audit: warning -->"

// textRules apply to every docs line and to every Go string literal.
var textRules = []struct {
	re  *regexp.Regexp
	why string
}{
	{regexp.MustCompile(`[$%]\{response\.`), "a webhook's output reads the response's fields from the root; there is no response. prefix"},
	{regexp.MustCompile(`\$\{enc:url:`), "there is no enc:url helper; write ${enc:args.x}"},
	{regexp.MustCompile(`@\{strftime `), "there is no strftime function; write @{strftime_tz <zone> <format>}"},
	{regexp.MustCompile(`[?&][A-Za-z0-9_]+=[$%]\{args\.`), "an argument in a URL query string must be encoded: ${enc:args.x}"},
}

// webhookArgsRe finds ${args.x} (with or without helpers such as enc:), which
// expands to nothing in a webhook's output, foreach and expressions.
var webhookArgsRe = regexp.MustCompile(`[$%]\{(?:[a-z_]+:)*args\.`)

const webhookArgsWhy = "${args.x} is empty in a webhook's output, foreach and expressions; write ${input.args.x}"

func TestDataMapTemplates(t *testing.T) {
	root := ".."
	var findings []string
	report := func(where, why, text string) {
		findings = append(findings, fmt.Sprintf("%s: %s: %q", where, why, strings.TrimSpace(text)))
	}

	docs, err := filepath.Glob(filepath.Join(root, "docs", "*.md"))
	if err != nil || len(docs) == 0 {
		t.Fatalf("no docs found under %s/docs: %v", root, err)
	}
	for _, path := range docs {
		scanMarkdown(t, path, report)
	}

	for _, dir := range []string{"examples", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			scanGoFile(t, path, report)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, f := range findings {
		t.Error(f)
	}
}

// TestDataMapTemplatesFindsEachRule checks that each rule reports its sample,
// and that a marked warning line passes.
func TestDataMapTemplatesFindsEachRule(t *testing.T) {
	doc := "There's no `${enc:url:x}`. " + warningMarker + "\n" +
		"- Built-in functions: `@{strftime %Y-%m-%d}`\n" +
		"```json\n" +
		`{"function": "f",
		  "parameters": {"properties": {"query": {"type": "string"}}},
		  "data_map": {
		    "webhooks": [{
		      "url": "https://example.com/s?q=${args.query}&p=${query}",
		      "output": {"response": "${response.total} for ${args.query}"}
		    }],
		    "output": {"response": "No ${enc:url:args.query}"}
		  }}` + "\n" +
		"```\n" +
		"```go\n" +
		`dm := datamap.New("f").
	Parameter("city", "string", "City", true, nil).
	Webhook("GET", "https://example.com/w/${city}", nil, "", false, nil).
	Output(swaig.NewFunctionResult("Weather in ${args.city}"))
tool := datamap.CreateSimpleAPITool("g", "https://example.com/g/${zip}", "Zip ${args.zip}",
	map[string]map[string]any{"zip": {"type": "string"}}, "GET", nil, nil, nil)` + "\n" +
		"```\n"
	path := filepath.Join(t.TempDir(), "sample.md")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}

	got := map[string]int{}
	scanMarkdown(t, path, func(_, why, _ string) { got[why]++ })

	for why, n := range map[string]int{
		textRules[0].why: 1, // ${response.total}
		textRules[1].why: 1, // ${enc:url:args.query} in the block; the marked line passes
		textRules[2].why: 1, // @{strftime
		textRules[3].why: 1, // ?q=${args.query}
		webhookArgsWhy:   3, // ${args.query} in the JSON output, ${args.city} and ${args.zip} in the Go outputs
		`the argument "query" is under args here; write ${args.query}`: 1,
		`the argument "city" is under args here; write ${args.city}`:   1,
		`the argument "zip" is under args here; write ${args.zip}`:     1,
	} {
		if got[why] != n {
			t.Errorf("%q reported %d times, want %d", why, got[why], n)
		}
	}
}

// scanMarkdown checks every line against textRules, and each fenced code
// block: a JSON block that parses against the data_map rules, and a Go block
// that parses against the builder rules.
func scanMarkdown(t *testing.T, path string, report func(where, why, text string)) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var block []string
	inFence, lang, start := false, "", 0
	for i, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, warningMarker) {
			for _, r := range textRules {
				if r.re.MatchString(line) {
					report(fmt.Sprintf("%s:%d", path, i+1), r.why, line)
				}
			}
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				inFence, lang, start, block = true, strings.TrimPrefix(trimmed, "```"), i+2, nil
				continue
			}
			inFence = false
			body := strings.Join(block, "\n")
			where := fmt.Sprintf("%s:%d", path, start)
			var v any
			if json.Unmarshal([]byte(body), &v) == nil {
				walkJSON(v, nil, func(why, text string) { report(where, why, text) })
			}
			if lang == "go" {
				scanGoSnippet(path, start, body, report)
			}
			continue
		}
		if inFence {
			block = append(block, line)
		}
	}
}

// walkJSON finds each object with "webhooks" and checks it as a data_map. The
// parameters are those of the nearest enclosing function definition.
func walkJSON(v any, params []string, report func(why, text string)) {
	switch x := v.(type) {
	case map[string]any:
		if p, ok := x["parameters"].(map[string]any); ok {
			if props, ok := p["properties"].(map[string]any); ok {
				params = nil
				for name := range props {
					params = append(params, name)
				}
			}
		}
		if _, ok := x["webhooks"]; ok {
			checkJSONDataMap(x, params, report)
		}
		for _, child := range x {
			walkJSON(child, params, report)
		}
	case []any:
		for _, child := range x {
			walkJSON(child, params, report)
		}
	}
}

func checkJSONDataMap(dm map[string]any, params []string, report func(why, text string)) {
	webhooks, _ := dm["webhooks"].([]any)
	for _, w := range webhooks {
		wh, _ := w.(map[string]any)
		for _, key := range []string{"output", "foreach", "expressions"} {
			eachJSONString(wh[key], func(s string) {
				if webhookArgsRe.MatchString(s) {
					report(webhookArgsWhy, s)
				}
			})
		}
		for _, key := range []string{"url", "params"} {
			eachJSONString(wh[key], func(s string) { checkBare(s, params, report) })
		}
	}
	for _, key := range []string{"output", "expressions"} {
		eachJSONString(dm[key], func(s string) { checkBare(s, params, report) })
	}
}

func eachJSONString(v any, fn func(string)) {
	switch x := v.(type) {
	case string:
		fn(x)
	case map[string]any:
		for _, child := range x {
			eachJSONString(child, fn)
		}
	case []any:
		for _, child := range x {
			eachJSONString(child, fn)
		}
	}
}

// checkBare reports ${name} for a declared parameter where the template reads
// the call data, in which the arguments are under args.
func checkBare(s string, params []string, report func(why, text string)) {
	for _, name := range params {
		re := regexp.MustCompile(`[$%]\{(?:[a-z_]+:)*` + regexp.QuoteMeta(name) + `[}.\[]`)
		if re.MatchString(s) {
			report(fmt.Sprintf("the argument %q is under args here; write ${args.%s}", name, name), s)
		}
	}
}

func scanGoFile(t *testing.T, path string, report func(where, why, text string)) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			for _, r := range textRules {
				if r.re.MatchString(s) {
					report(fset.Position(lit.Pos()).String(), r.why, s)
				}
			}
		}
		return true
	})
	checkGoAST(fset, f, func(pos token.Pos, why, text string) {
		report(fset.Position(pos).String(), why, text)
	})
}

// scanGoSnippet checks a Go code block from the docs with the builder rules.
// A snippet is parsed as a file, as a function body, or as imports followed
// by a function body. One that parses as none of these is left to the text
// rules.
func scanGoSnippet(path string, start int, body string, report func(where, why, text string)) {
	imports, rest := splitImports(body)
	for _, wrap := range []struct {
		head, body, tail string
		skip             int // snippet lines before wrap.body
	}{
		{"package p\n", body, "", 0},
		{"package p\nfunc _() {\n", body, "\n}", 0},
		{"package p\n" + imports + "\nfunc _() {\n", rest, "\n}", strings.Count(imports, "\n") + 1},
	} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", wrap.head+wrap.body+wrap.tail, 0)
		if err != nil {
			continue
		}
		offset := strings.Count(wrap.head, "\n")
		checkGoAST(fset, f, func(pos token.Pos, why, text string) {
			report(fmt.Sprintf("%s:%d", path, start+fset.Position(pos).Line-1-offset+wrap.skip), why, text)
		})
		return
	}
}

// splitImports separates the import declarations at the start of a snippet
// from the rest.
func splitImports(body string) (imports, rest string) {
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); {
		l := strings.TrimSpace(lines[i])
		switch {
		case l == "" || strings.HasPrefix(l, "//"):
			i++
		case strings.HasPrefix(l, "import ("):
			for i < len(lines) && strings.TrimSpace(lines[i]) != ")" {
				i++
			}
			i++
		case strings.HasPrefix(l, "import "):
			i++
		default:
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i:], "\n")
		}
	}
	return body, ""
}

// checkGoAST applies the data_map rules to Go code, in two shapes: a map
// literal under a "webhooks" key (as the built-in skills write it), and a
// chain of datamap builder calls.
func checkGoAST(fset *token.FileSet, f *ast.File, report func(pos token.Pos, why, text string)) {
	inChain := map[*ast.CallExpr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if goStringValue(x.Key) == "webhooks" {
				checkWebhooksLiteral(x.Value, report)
			}
		case *ast.CallExpr:
			if !inChain[x] {
				checkBuilderChain(flattenChain(x, inChain), report)
			}
		}
		return true
	})
}

// checkWebhooksLiteral reports ${args.x} in the output, foreach and
// expressions of a webhooks map literal. It doesn't look inside function
// calls, such as a strings.ReplaceAll that rewrites ${args. on purpose.
func checkWebhooksLiteral(v ast.Expr, report func(pos token.Pos, why, text string)) {
	ast.Inspect(v, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		switch goStringValue(kv.Key) {
		case "output", "foreach", "expressions":
			ast.Inspect(kv.Value, func(n ast.Node) bool {
				if _, ok := n.(*ast.CallExpr); ok {
					return false
				}
				if s := goStringValue(n); s != "" && webhookArgsRe.MatchString(s) {
					report(n.Pos(), webhookArgsWhy, s)
				}
				return true
			})
			return false
		}
		return true
	})
}

// flattenChain returns the calls of a method chain such as
// datamap.New(...).Webhook(...).Output(...), first call first.
func flattenChain(call *ast.CallExpr, inChain map[*ast.CallExpr]bool) []*ast.CallExpr {
	var calls []*ast.CallExpr
	for c := call; c != nil; {
		calls = append([]*ast.CallExpr{c}, calls...)
		inChain[c] = true
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}
		c, _ = sel.X.(*ast.CallExpr)
	}
	return calls
}

func checkBuilderChain(calls []*ast.CallExpr, report func(pos token.Pos, why, text string)) {
	var params []string
	inWebhook := false
	bare := func(args []ast.Expr) {
		for _, a := range args {
			eachGoString(a, func(pos token.Pos, s string) {
				checkBare(s, params, func(why, text string) { report(pos, why, text) })
			})
		}
	}
	for _, c := range calls {
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		switch sel.Sel.Name {
		case "CreateSimpleAPITool": // (name, url, responseTemplate, parameters, ...)
			if len(c.Args) < 4 {
				continue
			}
			params = append(params, goMapKeys(c.Args[3])...)
			inWebhook = true
			bare(c.Args[1:2])
			eachGoString(c.Args[2], func(pos token.Pos, s string) {
				if webhookArgsRe.MatchString(s) {
					report(pos, webhookArgsWhy, s)
				}
			})
		case "CreateExpressionTool": // (name, patterns, parameters)
			if len(c.Args) < 3 {
				continue
			}
			params = append(params, goMapKeys(c.Args[2])...)
			bare(c.Args[1:2])
		case "Parameter":
			if len(c.Args) > 0 {
				if s := goStringValue(c.Args[0]); s != "" {
					params = append(params, s)
				}
			}
		case "Webhook":
			inWebhook = true
			if len(c.Args) > 1 {
				bare(c.Args[1:2])
			}
		case "Params", "Body", "Expression", "FallbackOutput":
			bare(c.Args)
		case "Output", "Foreach", "WebhookExpressions":
			if !inWebhook {
				continue
			}
			for _, a := range c.Args {
				eachGoString(a, func(pos token.Pos, s string) {
					if webhookArgsRe.MatchString(s) {
						report(pos, webhookArgsWhy, s)
					}
				})
			}
		}
	}
}

func eachGoString(n ast.Node, fn func(token.Pos, string)) {
	ast.Inspect(n, func(n ast.Node) bool {
		if s := goStringValue(n); s != "" {
			fn(n.Pos(), s)
		}
		return true
	})
}

// goMapKeys returns the string keys of a map composite literal.
func goMapKeys(e ast.Expr) []string {
	lit, ok := e.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var keys []string
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if k := goStringValue(kv.Key); k != "" {
				keys = append(keys, k)
			}
		}
	}
	return keys
}

func goStringValue(n ast.Node) string {
	lit, ok := n.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return s
}
