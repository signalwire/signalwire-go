package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	if s.tempUnit != "fahrenheit" && s.tempUnit != "celsius" {
		s.tempUnit = "fahrenheit"
	}
	return true
}

func (s *WeatherAPISkill) RegisterTools() []skills.ToolRegistration {
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
			Handler: s.handleGetWeather,
		},
	}
}

func (s *WeatherAPISkill) handleGetWeather(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	location, _ := args["location"].(string)
	if location == "" {
		return swaig.NewFunctionResult("Please provide a location to get weather for.")
	}

	apiURL := fmt.Sprintf(
		"https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no",
		s.apiKey,
		url.QueryEscape(location),
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
