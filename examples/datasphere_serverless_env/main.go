//go:build ignore

// Example: datasphere_serverless_env
//
// DataSphere Serverless skill configured from environment variables.
// Shows best practices for production deployment.
//
// Required: SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE,
//           DATASPHERE_DOCUMENT_ID
// Optional: DATASPHERE_COUNT, DATASPHERE_DISTANCE, DATASPHERE_TAGS
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/signalwire/signalwire-go/pkg/agent"

	_ "github.com/signalwire/signalwire-go/pkg/skills/builtin"
)

func requireEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		fmt.Printf("Error: Required environment variable %s is not set.\n", name)
		os.Exit(1)
	}
	return val
}

func main() {
	documentID := requireEnv("DATASPHERE_DOCUMENT_ID")

	count := 3
	if v := os.Getenv("DATASPHERE_COUNT"); v != "" {
		count, _ = strconv.Atoi(v)
	}

	distance := 4.0
	if v := os.Getenv("DATASPHERE_DISTANCE"); v != "" {
		distance, _ = strconv.ParseFloat(v, 64)
	}

	a := agent.NewAgentBase(
		agent.WithName("DatasphereServerlessEnv"),
		agent.WithRoute("/datasphere-env"),
		agent.WithPort(3031),
	)

	a.AddLanguage(map[string]any{
		"name":  "English",
		"code":  "en-US",
		"voice": "rime.spore",
	})

	a.PromptAddSection("Role",
		"You are a knowledge assistant with access to a document library via "+
			"serverless DataSphere. Search for relevant information to answer questions.",
		nil,
	)

	a.AddSkill("datetime", nil)
	a.AddSkill("math", nil)

	config := map[string]any{
		"document_id": documentID,
		"count":       count,
		"distance":    distance,
	}

	if tags := os.Getenv("DATASPHERE_TAGS"); tags != "" {
		config["tags"] = strings.Split(tags, ",")
	}

	a.AddSkill("datasphere", config)

	fmt.Println("Starting DatasphereServerlessEnv on :3031/datasphere-env ...")
	fmt.Printf("  Document: %s\n", documentID)
	fmt.Printf("  Count: %d, Distance: %.1f\n", count, distance)

	if err := a.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
