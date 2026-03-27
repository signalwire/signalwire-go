// Example: datamap_demo
//
// Server-side tools using DataMap that execute on SignalWire servers
// without requiring webhook endpoints. Demonstrates both webhook-based
// API calls (with variable expansion) and expression-based pattern matching.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/datamap"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("DataMapDemo"),
		agent.WithRoute("/datamap"),
		agent.WithPort(3005),
	)

	a.SetPromptText(
		"You are a helpful assistant that can look up weather information " +
			"and control music playback. Use the available tools to help the user.",
	)

	// ---- DataMap 1: Weather API webhook ----
	// This tool runs entirely on SignalWire servers. When invoked, it makes
	// an HTTP GET to the weather API with variable expansion for the location.
	weatherDM := datamap.New("get_weather").
		Purpose("Get current weather information for a location").
		Parameter("location", "string", "City name or zip code", true, nil).
		Webhook("GET",
			"https://api.weatherapi.com/v1/current.json?key=YOUR_API_KEY&q=${args.location}",
			nil, "", false, nil,
		).
		Output(swaig.NewFunctionResult(
			"Weather in ${args.location}: ${response.current.condition.text}, "+
				"Temperature: ${response.current.temp_f}°F (${response.current.temp_c}°C), "+
				"Humidity: ${response.current.humidity}%, "+
				"Wind: ${response.current.wind_mph} mph",
		)).
		FallbackOutput(swaig.NewFunctionResult(
			"Sorry, I could not retrieve the weather for that location.",
		))

	// Register the weather DataMap as a SWAIG function
	a.RegisterSwaigFunction(weatherDM.ToSwaigFunction())

	// ---- DataMap 2: Expression-based pattern matching ----
	// This tool uses expressions to match user commands and return actions.
	// No external API call is needed.
	controlDM := datamap.New("music_control").
		Purpose("Control music playback with play, pause, skip, or stop commands").
		Parameter("command", "string", "The music command (play, pause, skip, stop)", true,
			[]string{"play", "pause", "skip", "stop"})

	// Add pattern matching expressions
	controlDM.Expression(
		"${args.command}", "play",
		swaig.NewFunctionResult("Now playing music.").
			AddAction("playback_bg", "https://cdn.example.com/music/playlist.mp3"),
		nil,
	)
	controlDM.Expression(
		"${args.command}", "pause",
		swaig.NewFunctionResult("Music paused.").
			AddAction("stop_playback_bg", true),
		nil,
	)
	controlDM.Expression(
		"${args.command}", "skip",
		swaig.NewFunctionResult("Skipping to the next track.").
			AddAction("stop_playback_bg", true).
			AddAction("playback_bg", "https://cdn.example.com/music/next_track.mp3"),
		nil,
	)
	controlDM.Expression(
		"${args.command}", "stop",
		swaig.NewFunctionResult("Music stopped.").
			AddAction("stop_playback_bg", true),
		swaig.NewFunctionResult("Unknown command. Please use play, pause, skip, or stop."),
	)

	// Register the expression-based DataMap
	a.RegisterSwaigFunction(controlDM.ToSwaigFunction())

	fmt.Println("Starting DataMapDemo on :3005/datamap ...")
	fmt.Println("  get_weather:   webhook-based API call with variable expansion")
	fmt.Println("  music_control: expression-based pattern matching")

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
