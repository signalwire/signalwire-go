//go:build ignore

// Example: advanced_datamap
//
// Advanced DataMap features: expressions with regex patterns, webhooks with
// body/headers/form_param, foreach array processing, and choosing a webhook
// with require_args, with a fallback output.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/datamap"
	"github.com/signalwire/signalwire-go/v3/pkg/swaig"
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
	// The platform requests one webhook per call: the first whose require_args
	// are met. Here the first webhook runs when the AI supplies data to send,
	// and the second when it supplies only the action. When the webhook that
	// ran fails, the fallback output runs; the platform doesn't try the other.
	advancedAPI := datamap.New("advanced_api_tool").
		Purpose("API tool with advanced webhook features").
		Parameter("action", "string", "Action to perform", true, nil).
		Parameter("data", "string", "Data to send", false, nil).
		// Requested when there's data to send
		Webhook("POST", "https://api.example.com/advanced",
			map[string]string{"Authorization": "Bearer YOUR_TOKEN"},
			"payload",        // form_param: send the body as one form field
			true,             // inputArgsAsParams: the arguments become the body
			[]string{"data"}, // requireArgs: only when data is provided
		).
		// Post-webhook expressions pick a reply from the response
		WebhookExpressions([]map[string]any{
			{
				"string":  "${status}",
				"pattern": "^success$",
				"output":  map[string]any{"response": "Operation completed successfully"},
			},
			{
				"string":  "${error_code}",
				"pattern": "^(404|500)$",
				"output":  map[string]any{"response": "API Error: ${error_message}"},
			},
		}).
		// Used when neither expression matches
		Output(swaig.NewFunctionResult("The API returned ${status} for ${input.args.action}")).
		ErrorKeys([]string{"fault", "exception"}).
		// Requested when there's no data: the action goes in the URL
		Webhook("GET", "https://api.example.com/simple?q=${enc:args.action}",
			map[string]string{"Accept": "application/json"},
			"", false, nil,
		).
		Output(swaig.NewFunctionResult("Result: ${data}")).
		ErrorKeys([]string{"error", "fault", "exception"}).
		// Used when the webhook that ran fails
		FallbackOutput(swaig.NewFunctionResult("The API is unavailable right now"))

	a.RegisterSwaigFunction(advancedAPI.ToSwaigFunction())

	// ---- DataMap 3: Form submission ----
	formDM := datamap.New("form_submission_tool").
		Purpose("Submit form data using form encoding").
		Parameter("name", "string", "User name", true, nil).
		Parameter("email", "string", "User email", true, nil).
		Parameter("message", "string", "Message content", true, nil).
		// With form_param, the platform sets the form Content-Type itself
		Webhook("POST", "https://forms.example.com/submit",
			map[string]string{"X-API-Key": "YOUR_API_KEY"},
			"form_data", false, nil,
		).
		Params(map[string]any{
			"name":    "${args.name}",
			"email":   "${args.email}",
			"message": "${args.message}",
		}).
		Output(swaig.NewFunctionResult("Form submitted successfully for ${input.args.name}")).
		ErrorKeys([]string{"error", "validation_errors"})

	a.RegisterSwaigFunction(formDM.ToSwaigFunction())

	// ---- DataMap 4: Foreach array processing ----
	searchDM := datamap.New("search_results_tool").
		Purpose("Search and format results from API").
		Parameter("query", "string", "Search query", true, nil).
		Parameter("limit", "string", "Maximum results", false, nil).
		// A GET's query goes in the URL: params would be a JSON body, and
		// make the request a POST
		Webhook("GET", "https://search-api.example.com/search?q=${enc:args.query}&max_results=${enc:args.limit}",
			map[string]string{"Authorization": "Bearer YOUR_SEARCH_TOKEN"},
			"", false, nil,
		).
		Foreach(map[string]any{
			"input_key":  "results",
			"output_key": "formatted_results",
			"max":        5,
			"append":     "Title: ${this.title}\n${this.summary}\nURL: ${this.url}\n\n",
		}).
		Output(swaig.NewFunctionResult("Search results for \"${input.args.query}\":\n\n${formatted_results}")).
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
