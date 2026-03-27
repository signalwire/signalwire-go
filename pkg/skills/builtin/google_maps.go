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

// GoogleMapsSkill validates addresses and computes routes using Google Maps.
type GoogleMapsSkill struct {
	skills.BaseSkill
	apiKey string
}

// NewGoogleMaps creates a new GoogleMapsSkill.
func NewGoogleMaps(params map[string]any) skills.SkillBase {
	return &GoogleMapsSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "google_maps",
			SkillDesc: "Validate addresses and compute driving routes using Google Maps",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *GoogleMapsSkill) RequiredEnvVars() []string {
	if s.Params != nil {
		if _, ok := s.Params["api_key"]; ok {
			return nil
		}
	}
	return []string{"GOOGLE_MAPS_API_KEY"}
}

func (s *GoogleMapsSkill) Setup() bool {
	s.apiKey = s.GetParamString("api_key", os.Getenv("GOOGLE_MAPS_API_KEY"))
	return s.apiKey != ""
}

func (s *GoogleMapsSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        "search_places",
			Description: "Search for places and validate addresses using Google Maps",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Address, business name, or place to search for",
					},
				},
				"required": []string{"query"},
			},
			Handler: s.handleSearchPlaces,
		},
	}
}

func (s *GoogleMapsSkill) handleSearchPlaces(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	query, _ := args["query"].(string)
	if query == "" {
		return swaig.NewFunctionResult("Please provide an address or place to search for.")
	}

	// Use Places Autocomplete API
	apiURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/autocomplete/json?input=%s&key=%s",
		url.QueryEscape(query),
		s.apiKey,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return swaig.NewFunctionResult("Error connecting to Google Maps.")
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing Google Maps response.")
	}

	status, _ := data["status"].(string)
	if status != "OK" {
		return swaig.NewFunctionResult("I couldn't find that address. Could you provide a more specific address?")
	}

	predictions, _ := data["predictions"].([]any)
	if len(predictions) == 0 {
		return swaig.NewFunctionResult("No results found for that search.")
	}

	// Get the top prediction's details
	pred, _ := predictions[0].(map[string]any)
	placeID, _ := pred["place_id"].(string)
	description, _ := pred["description"].(string)

	if placeID == "" {
		return swaig.NewFunctionResult("Address: " + description)
	}

	// Get place details for coordinates
	detailsURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/details/json?place_id=%s&fields=name,formatted_address,geometry&key=%s",
		placeID,
		s.apiKey,
	)

	detResp, err := client.Get(detailsURL)
	if err != nil {
		return swaig.NewFunctionResult("Address: " + description)
	}
	defer detResp.Body.Close()

	var detData map[string]any
	if err := json.NewDecoder(detResp.Body).Decode(&detData); err != nil {
		return swaig.NewFunctionResult("Address: " + description)
	}

	result, _ := detData["result"].(map[string]any)
	if result == nil {
		return swaig.NewFunctionResult("Address: " + description)
	}

	var parts []string
	if addr, ok := result["formatted_address"].(string); ok {
		parts = append(parts, "Address: "+addr)
	}
	if name, ok := result["name"].(string); ok {
		parts = append(parts, "Name: "+name)
	}
	if geo, ok := result["geometry"].(map[string]any); ok {
		if loc, ok := geo["location"].(map[string]any); ok {
			lat, _ := loc["lat"].(float64)
			lng, _ := loc["lng"].(float64)
			parts = append(parts, fmt.Sprintf("Coordinates: %.6f, %.6f", lat, lng))
		}
	}

	return swaig.NewFunctionResult(strings.Join(parts, "\n"))
}

func (s *GoogleMapsSkill) GetHints() []string {
	return []string{"address", "location", "route", "directions", "miles", "distance"}
}

func (s *GoogleMapsSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Google Maps",
			"body":  "You can search for places and validate addresses.",
			"bullets": []string{
				"Use search_places to validate and geocode addresses or business names",
				"Address lookup supports business names and street addresses",
			},
		},
	}
}

func (s *GoogleMapsSkill) GetParameterSchema() map[string]map[string]any {
	schema := s.BaseSkill.GetParameterSchema()
	schema["api_key"] = map[string]any{
		"type":        "string",
		"description": "Google Maps API key",
		"required":    true,
		"hidden":      true,
		"env_var":     "GOOGLE_MAPS_API_KEY",
	}
	return schema
}

func init() {
	skills.RegisterSkill("google_maps", NewGoogleMaps)
}
