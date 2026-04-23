// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package datamap

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// Tests for the public symbols added in this PR that the alignment
// review flagged as untested: FunctionName getter and ExpressionRegexp
// compiled-regexp convenience wrapper. Mirrors Python data_map.DataMap
// surface in tests/unit/core/test_data_map.py.

func TestFunctionNameReturnsConstructorValue(t *testing.T) {
	dm := New("lookup_weather")
	if got := dm.FunctionName(); got != "lookup_weather" {
		t.Fatalf("FunctionName = %q, want %q", got, "lookup_weather")
	}
}

func TestFunctionNameEmpty(t *testing.T) {
	dm := New("")
	if got := dm.FunctionName(); got != "" {
		t.Fatalf("FunctionName = %q for empty name, want empty string", got)
	}
}

func TestExpressionRegexpStoresPatternString(t *testing.T) {
	// ExpressionRegexp should mirror Expression() but accept a compiled
	// *regexp.Regexp and store its .String() as the pattern — matches
	// Python's expression() handling of a compiled re.Pattern.
	dm := New("cmd")
	pat := regexp.MustCompile(`^ping\s*$`)
	out := swaig.NewFunctionResult("pong")
	dm.ExpressionRegexp("${args.command}", pat, out, nil)

	body, err := json.Marshal(dm.ToSwaigFunction())
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	s := string(body)
	if !strings.Contains(s, `^ping`) {
		t.Fatalf("expected compiled regexp's string form in data_map output, got: %s", s)
	}
	if !strings.Contains(s, "${args.command}") {
		t.Fatalf("expected test value to be preserved, got: %s", s)
	}
}

func TestExpressionRegexpEquivalentToExpression(t *testing.T) {
	// ExpressionRegexp("${x}", regexp.MustCompile("p"), out, nil)
	// should produce identical output to Expression("${x}", "p", out, nil).
	out := swaig.NewFunctionResult("match")

	dmA := New("a")
	dmA.Expression("${args.x}", "^foo$", out, nil)

	dmB := New("a")
	dmB.ExpressionRegexp("${args.x}", regexp.MustCompile("^foo$"), out, nil)

	aBytes, _ := json.Marshal(dmA.ToSwaigFunction())
	bBytes, _ := json.Marshal(dmB.ToSwaigFunction())
	if string(aBytes) != string(bBytes) {
		t.Fatalf("ExpressionRegexp and Expression should produce equivalent output\nExpression:\n%s\nExpressionRegexp:\n%s",
			aBytes, bBytes)
	}
}
