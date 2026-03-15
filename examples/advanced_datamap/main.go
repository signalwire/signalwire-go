//go:build ignore

// Example: advanced_datamap
//
// Advanced DataMap features: expressions with regex patterns, webhooks with
// body/headers/form_param, foreach array processing, multi-webhook fallback
// chains, and global error keys.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-agents-go/pkg/agent"
	"github.com/signalwire/signalwire-agents-go/pkg/datamap"
	"github.com/signalwire/signalwire-agents-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("AdvancedDataMapDemo"),
		agent.WithRoute("/advanced-datamap"),
		agent.WithPort(3009),
	)

	a.SetPromptText(
		"You are a helpful assistant that can process commands, submit forms, " +
			"and search for information. Use the available tools to help the user.",
	)

	// ---- DataMap 1: Expression-based command processor ----
	// Uses regex patterns to match user commands and return different responses.
	commandDM := datamap.New("command_processor").
		Purpose("Process user commands with pattern matching").
		Parameter("command", "string", "User command to process", true, nil).
		Parameter("target", "string", "Optional target for the command", false, nil)

	commandDM.Expression(
		"${args.command}", "^start",
		swaig.NewFunctionResult("Starting process: ${args.target}").
			AddAction("start_process", map[string]any{"target": "${args.target}"}),
		nil,
	)
	commandDM.Expression(
		"${args.command}", "^stop",
		swaig.NewFunctionResult("Stopping process: ${args.target}").
			AddAction("stop_process", map[string]any{"target": "${args.target}"}),
		nil,
	)
	commandDM.Expression(
		"${args.command}", "^status",
		swaig.NewFunctionResult("Checking status of: ${args.target}"),
		swaig.NewFunctionResult("Unknown command: ${args.command}. Try start, stop, or status."),
	)

	a.RegisterSwaigFunction(commandDM.ToSwaigFunction())

	// ---- DataMap 2: Webhook with headers, form_param, and require_args ----
	advancedAPI := datamap.New("advanced_api_tool").
		Purpose("API tool with advanced webhook features").
		Parameter("action", "string", "Action to perform", true, nil).
		Parameter("data", "string", "Data to send", false, nil).
		Webhook("POST", "https://api.example.com/advanced",
			map[string]string{
				"Authorization": "Bearer ${token}",
				"User-Agent":    "SignalWire-Agent/1.0",
			},
			"payload", // form_param: sends body as a form parameter
			true,      // inputArgsAsParams: merge function args into params
			[]string{"action"}, // requireArgs: only execute if action is present
		).
		WebhookExpressions([]map[string]any{
			{
				"string":  "${response.status}",
				"pattern": "^success$",
				"output":  map[string]any{"response": "Operation completed successfully"},
			},
			{
				"string":  "${response.error_code}",
				"pattern": "^(404|500)$",
				"output":  map[string]any{"response": "API Error: ${response.error_message}"},
			},
		}).
		Output(swaig.NewFunctionResult("Result: ${response.data}")).
		// Second webhook as fallback
		Webhook("GET", "https://backup-api.example.com/simple",
			map[string]string{"Accept": "application/json"},
			"", false, nil,
		).
		Params(map[string]any{"q": "${args.action}"}).
		Output(swaig.NewFunctionResult("Backup result: ${response.data}")).
		FallbackOutput(swaig.NewFunctionResult("All APIs are currently unavailable")).
		GlobalErrorKeys([]string{"error", "fault", "exception"})

	a.RegisterSwaigFunction(advancedAPI.ToSwaigFunction())

	// ---- DataMap 3: Form submission ----
	formDM := datamap.New("form_submission_tool").
		Purpose("Submit form data using form encoding").
		Parameter("name", "string", "User name", true, nil).
		Parameter("email", "string", "User email", true, nil).
		Parameter("message", "string", "Message content", true, nil).
		Webhook("POST", "https://forms.example.com/submit",
			map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"X-API-Key":    "${api_key}",
			},
			"form_data", false, nil,
		).
		Params(map[string]any{
			"name":    "${args.name}",
			"email":   "${args.email}",
			"message": "${args.message}",
		}).
		Output(swaig.NewFunctionResult("Form submitted successfully for ${args.name}")).
		ErrorKeys([]string{"error", "validation_errors"})

	a.RegisterSwaigFunction(formDM.ToSwaigFunction())

	// ---- DataMap 4: Foreach array processing ----
	searchDM := datamap.New("search_results_tool").
		Purpose("Search and format results from API").
		Parameter("query", "string", "Search query", true, nil).
		Parameter("limit", "string", "Maximum results", false, nil).
		Webhook("GET", "https://search-api.example.com/search",
			map[string]string{"Authorization": "Bearer ${search_token}"},
			"", false, nil,
		).
		Params(map[string]any{
			"q":           "${args.query}",
			"max_results": "${args.limit}",
		}).
		Foreach(map[string]any{
			"input_key":  "results",
			"output_key": "formatted_results",
			"max":        5,
			"append":     "Title: ${this.title}\n${this.summary}\nURL: ${this.url}\n\n",
		}).
		Output(swaig.NewFunctionResult("Search results for \"${args.query}\":\n\n${formatted_results}")).
		ErrorKeys([]string{"error"})

	a.RegisterSwaigFunction(searchDM.ToSwaigFunction())

	// ---- Print all tool definitions for inspection ----
	fmt.Println("Advanced DataMap Demo - Tool Definitions:")
	for _, tool := range a.DefineTools() {
		if tool.SwaigFields != nil {
			data, _ := json.MarshalIndent(tool.SwaigFields, "", "  ")
			fmt.Printf("\n%s:\n%s\n", tool.Name, string(data))
		}
	}

	fmt.Println("\nStarting AdvancedDataMapDemo on :3009/advanced-datamap ...")
	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
