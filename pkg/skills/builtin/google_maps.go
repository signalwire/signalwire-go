package builtin

import (
	"bytes"
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
	apiKey         string
	lookupToolName string
	routeToolName  string
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
	s.lookupToolName = s.GetParamString("lookup_tool_name", "lookup_address")
	s.routeToolName = s.GetParamString("route_tool_name", "compute_route")
	return s.apiKey != ""
}

func (s *GoogleMapsSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name: s.lookupToolName,
			Description: "Validate and geocode a street address or business name using Google Maps. " +
				"Optionally bias results toward a known location (e.g. find the nearest Walmart).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"address": map[string]any{
						"type":        "string",
						"description": "The address or business name to look up",
					},
					"bias_lat": map[string]any{
						"type":        "number",
						"description": "Latitude to bias results toward (optional)",
					},
					"bias_lng": map[string]any{
						"type":        "number",
						"description": "Longitude to bias results toward (optional)",
					},
				},
				"required": []string{"address"},
			},
			Handler: s.handleLookupAddress,
		},
		{
			Name: s.routeToolName,
			Description: "Compute a driving route between two points using Google Maps Routes API. " +
				"Returns distance and estimated travel time.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"origin_lat": map[string]any{
						"type":        "number",
						"description": "Origin latitude",
					},
					"origin_lng": map[string]any{
						"type":        "number",
						"description": "Origin longitude",
					},
					"dest_lat": map[string]any{
						"type":        "number",
						"description": "Destination latitude",
					},
					"dest_lng": map[string]any{
						"type":        "number",
						"description": "Destination longitude",
					},
				},
				"required": []string{"origin_lat", "origin_lng", "dest_lat", "dest_lng"},
			},
			Handler: s.handleComputeRoute,
		},
	}
}

func (s *GoogleMapsSkill) handleLookupAddress(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	address, _ := args["address"].(string)
	address = strings.TrimSpace(address)
	if address == "" {
		return swaig.NewFunctionResult("Please provide an address or business name to look up.")
	}

	var biasLat, biasLng *float64
	if v, ok := args["bias_lat"].(float64); ok {
		biasLat = &v
	}
	if v, ok := args["bias_lng"].(float64); ok {
		biasLng = &v
	}

	// Use Places Autocomplete API (with optional location bias)
	autocompleteParams := url.Values{}
	autocompleteParams.Set("input", address)
	autocompleteParams.Set("key", s.apiKey)
	if biasLat != nil && biasLng != nil {
		autocompleteParams.Set("location", fmt.Sprintf("%f,%f", *biasLat, *biasLng))
		autocompleteParams.Set("radius", "50000")
	}

	apiURL := "https://maps.googleapis.com/maps/api/place/autocomplete/json?" + autocompleteParams.Encode()

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

func (s *GoogleMapsSkill) handleComputeRoute(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	originLat, okOLat := args["origin_lat"].(float64)
	originLng, okOLng := args["origin_lng"].(float64)
	destLat, okDLat := args["dest_lat"].(float64)
	destLng, okDLng := args["dest_lng"].(float64)

	if !okOLat || !okOLng || !okDLat || !okDLng {
		return swaig.NewFunctionResult("All four coordinates are required: origin_lat, origin_lng, dest_lat, dest_lng.")
	}

	routesURL := "https://routes.googleapis.com/directions/v2:computeRoutes"
	headers := map[string]string{
		"Content-Type":     "application/json",
		"X-Goog-Api-Key":   s.apiKey,
		"X-Goog-FieldMask": "routes.distanceMeters,routes.duration",
	}
	body := map[string]any{
		"origin": map[string]any{
			"location": map[string]any{
				"latLng": map[string]any{
					"latitude":  originLat,
					"longitude": originLng,
				},
			},
		},
		"destination": map[string]any{
			"location": map[string]any{
				"latLng": map[string]any{
					"latitude":  destLat,
					"longitude": destLng,
				},
			},
		},
		"travelMode":        "DRIVE",
		"routingPreference": "TRAFFIC_AWARE",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return swaig.NewFunctionResult("Error preparing route request.")
	}

	req, err := http.NewRequest(http.MethodPost, routesURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return swaig.NewFunctionResult("Error preparing route request.")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return swaig.NewFunctionResult("Error connecting to Google Maps Routes API.")
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return swaig.NewFunctionResult("Error processing route response.")
	}

	routes, _ := data["routes"].([]any)
	if len(routes) == 0 {
		return swaig.NewFunctionResult("I couldn't compute a route between those locations. Please verify the coordinates.")
	}

	route, _ := routes[0].(map[string]any)
	distanceMeters, _ := route["distanceMeters"].(float64)
	durationStr, _ := route["duration"].(string)
	durationStr = strings.TrimSuffix(durationStr, "s")

	var durationSeconds float64
	fmt.Sscanf(durationStr, "%f", &durationSeconds)

	distanceMiles := distanceMeters / 1609.344
	durationMin := durationSeconds / 60.0

	return swaig.NewFunctionResult(fmt.Sprintf(
		"Distance: %.1f miles\nEstimated travel time: %d minutes",
		distanceMiles,
		int(durationMin),
	))
}

func (s *GoogleMapsSkill) GetHints() []string {
	return []string{"address", "location", "route", "directions", "miles", "distance"}
}

func (s *GoogleMapsSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Google Maps",
			"body":  "You can validate addresses and compute driving routes.",
			"bullets": []string{
				fmt.Sprintf("Use %s to validate and geocode addresses or business names", s.lookupToolName),
				fmt.Sprintf("Use %s to get driving distance and time between two points", s.routeToolName),
				"Address lookup supports spoken numbers (e.g. 'seven one four' becomes '714')",
				"You can bias address results toward a known location to find the nearest match",
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
	schema["lookup_tool_name"] = map[string]any{
		"type":        "string",
		"description": "Name for the address lookup tool",
		"default":     "lookup_address",
		"required":    false,
	}
	schema["route_tool_name"] = map[string]any{
		"type":        "string",
		"description": "Name for the route computation tool",
		"default":     "compute_route",
		"required":    false,
	}
	return schema
}

func init() {
	skills.RegisterSkill("google_maps", NewGoogleMaps)
}
