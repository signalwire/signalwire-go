package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// WeatherAPISkill gets current weather from WeatherAPI.com.
type WeatherAPISkill struct {
	skills.BaseSkill
	apiKey      string
	toolName    string
	tempUnit    string
}

// NewWeatherAPI creates a new WeatherAPISkill.
func NewWeatherAPI(params map[string]any) skills.SkillBase {
	return &WeatherAPISkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "weather_api",
			SkillDesc: "Get current weather information from WeatherAPI.com",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *WeatherAPISkill) RequiredEnvVars() []string {
	// Only require env var if not provided in params
	if s.Params != nil {
		if _, ok := s.Params["api_key"]; ok {
			return nil
		}
	}
	return []string{"WEATHER_API_KEY"}
}

func (s *WeatherAPISkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", os.Getenv("WEATHER_API_KEY"))
	if s.apiKey == "" {
		return false
	}
	s.toolName = s.GetParamString("tool_name", "get_weather")
	s.tempUnit = s.GetParamString("temperature_unit", "fahrenheit")
	// Match Python _validate_config(): reject invalid temperature_unit values rather than silently
	// resetting (skill.py:103-104 raises ValueError; Go equivalent returns false from Setup).
	if s.tempUnit != "fahrenheit" && s.tempUnit != "celsius" {
		return false
	}
	return true
}

func (s *WeatherAPISkill) RegisterTools() []skills.ToolRegistration {
	// Determine temperature fields based on unit — mirrors Python get_tools() (skill.py:133-140).
	tempField := "temp_f"
	feelsLikeField := "feelslike_f"
	unitName := "Fahrenheit"
	if s.tempUnit == "celsius" {
		tempField = "temp_c"
		feelsLikeField = "feelslike_c"
		unitName = "Celsius"
	}

	// Build TTS-friendly response template matching Python (skill.py:143-160).
	responseInstruction := fmt.Sprintf(
		"Tell the user the current weather conditions. "+
			"Express all temperatures in %s using natural language numbers "+
			"without abbreviations or symbols for clear text-to-speech pronunciation. "+
			"For example, say 'seventy two degrees %s' instead of '72F' or '72°F'. "+
			"Include the condition, current temperature, wind direction and speed, "+
			"cloud coverage percentage, and what the temperature feels like.",
		unitName, unitName,
	)
	weatherTemplate := fmt.Sprintf(
		"%s Current conditions: ${current.condition.text}. "+
			"Temperature: ${current.%s} degrees %s. "+
			"Wind: ${current.wind_dir} at ${current.wind_mph} miles per hour. "+
			"Cloud coverage: ${current.cloud} percent. "+
			"Feels like: ${current.%s} degrees %s.",
		responseInstruction, tempField, unitName, feelsLikeField, unitName,
	)

	// DataMap webhook URL uses the platform-side variable expansion (Python skill.py:179).
	// Base URL is normally api.weatherapi.com; the porting-sdk's
	// audit_skills_dispatch.py overrides via WEATHER_API_BASE_URL so a
	// loopback fixture can stand in for the real WeatherAPI.com.
	base := os.Getenv("WEATHER_API_BASE_URL")
	if base == "" {
		base = "https://api.weatherapi.com"
	}
	base = strings.TrimRight(base, "/")
	webhookURL := fmt.Sprintf(
		"%s/v1/current.json?key=%s&q=${lc:enc:args.location}&aqi=no",
		base, s.apiKey,
	)

	return []skills.ToolRegistration{
		{
			Name:        s.toolName,
			Description: "Get current weather information for any location",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "The city, state, country, or location to get weather for",
					},
				},
				"required": []string{"location"},
			},
			// DataMap-based execution matches Python get_tools() data_map.webhooks pattern
			// (skill.py:176-188). The Handler below provides a local fallback for environments
			// where the platform-side DataMap is not available.
			SwaigFields: map[string]any{
				"data_map": map[string]any{
					"webhooks": []map[string]any{
						{
							"url":    webhookURL,
							"method": "GET",
							"output": map[string]any{
								"response": weatherTemplate,
							},
						},
					},
					"error_keys": []string{"error"},
					"output": map[string]any{
						"response": "Sorry, I cannot get weather information right now. Please try again later or check if the location name is correct.",
					},
				},
			},
			Handler: s.handleGetWeather,
		},
	}
}

func (s *WeatherAPISkill) handleGetWeather(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	location, _ := args["location"].(string)
	if location == "" {
		return swaig.NewFunctionResult("Please provide a location to get weather for.")
	}

	base := os.Getenv("WEATHER_API_BASE_URL")
	if base == "" {
		base = "https://api.weatherapi.com"
	}
	base = strings.TrimRight(base, "/")
	apiURL := fmt.Sprintf(
		"%s/v1/current.json?key=%s&q=%s&aqi=no",
		base, s.apiKey, url.QueryEscape(location),
	)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return swaig.NewFunctionResult("Sorry, I cannot get weather information right now. Please try again later.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swaig.NewFunctionResult("Sorry, I cannot get weather information right now. Please try again later or check if the location name is correct.")
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing weather data.")
	}

	current, _ := data["current"].(map[string]any)
	if current == nil {
		return swaig.NewFunctionResult("No weather data available for that location.")
	}

	condition, _ := current["condition"].(map[string]any)
	condText := "Unknown"
	if condition != nil {
		if ct, ok := condition["text"].(string); ok {
			condText = ct
		}
	}

	var temp, feelsLike float64
	unitName := "Fahrenheit"
	if s.tempUnit == "celsius" {
		temp, _ = current["temp_c"].(float64)
		feelsLike, _ = current["feelslike_c"].(float64)
		unitName = "Celsius"
	} else {
		temp, _ = current["temp_f"].(float64)
		feelsLike, _ = current["feelslike_f"].(float64)
	}
	windDir, _ := current["wind_dir"].(string)
	windMph, _ := current["wind_mph"].(float64)
	cloud, _ := current["cloud"].(float64)

	response := fmt.Sprintf(
		"Current conditions: %s. Temperature: %.0f degrees %s. "+
			"Wind: %s at %.0f miles per hour. "+
			"Cloud coverage: %.0f percent. "+
			"Feels like: %.0f degrees %s.",
		condText, temp, unitName,
		windDir, windMph,
		cloud,
		feelsLike, unitName,
	)

	return swaig.NewFunctionResult(response)
}

func (s *WeatherAPISkill) GetHints() []string {
	return []string{"weather", "temperature", "forecast", "wind", "clouds"}
}

func (s *WeatherAPISkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Weather Information",
			"body":  "You can get current weather information for any location.",
			"bullets": []string{
				"Use " + s.toolName + " to get current weather conditions",
				"Provides temperature, wind, cloud coverage, and feels-like temperature",
			},
		},
	}
}

func (s *WeatherAPISkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["api_key"] = map[string]any{
		"type":        "string",
		"description": "WeatherAPI.com API key",
		"required":    true,
		"hidden":      true,
		"env_var":     "WEATHER_API_KEY",
	}
	schema["tool_name"] = map[string]any{
		"type":        "string",
		"description": "Custom name for the weather tool",
		"default":     "get_weather",
		"required":    false,
	}
	schema["temperature_unit"] = map[string]any{
		"type":        "string",
		"description": "Temperature unit to display",
		"default":     "fahrenheit",
		"required":    false,
		"enum":        []string{"fahrenheit", "celsius"},
	}
	return schema
}

func init() {
	skills.RegisterSkill("weather_api", NewWeatherAPI)
}
