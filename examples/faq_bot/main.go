//go:build ignore

// Example: faq_bot
//
// FAQBotAgent prefab. Creates an agent that answers frequently asked
// questions from a provided FAQ database with built-in search and
// category-based matching.
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/prefabs"
)

func main() {
	faqBot := prefabs.NewFAQBotAgent(prefabs.FAQBotOptions{
		Name:           "SignalWireFAQ",
		Route:          "/faq",
		SuggestRelated: true,
		Persona:        "You are a knowledgeable FAQ assistant for SignalWire.",
		FAQs: []prefabs.FAQ{
			{
				Question:   "What is SignalWire?",
				Answer:     "SignalWire is a communications platform that provides APIs for voice, video, and messaging.",
				Categories: []string{"general", "overview"},
			},
			{
				Question:   "How do I create an AI Agent?",
				Answer:     "You can create an AI Agent using the SignalWire AI Agent SDK, which provides a simple way to build and deploy conversational AI agents.",
				Categories: []string{"development", "agents"},
			},
			{
				Question:   "What is SWML?",
				Answer:     "SWML (SignalWire Markup Language) is a markup language for defining communications workflows, including AI interactions.",
				Categories: []string{"development", "swml"},
			},
			{
				Question:   "What are SWAIG functions?",
				Answer:     "SWAIG (SignalWire AI Gateway) functions are tool endpoints that AI agents can call during a conversation to perform actions or retrieve data.",
				Categories: []string{"development", "agents"},
			},
			{
				Question:   "How do I deploy an agent?",
				Answer:     "Agents can be deployed as standalone servers, in Kubernetes, on serverless platforms (Lambda, Cloud Functions), or behind reverse proxies.",
				Categories: []string{"deployment", "operations"},
			},
		},
	})

	fmt.Println("Starting SignalWire FAQ Bot on :3000/faq ...")
	fmt.Println("  FAQs: 5 questions loaded")
	fmt.Println("  Tools: search_faqs")

	if err := faqBot.Run(); err != nil {
		fmt.Printf("Agent error: %v\n", err)
	}
}
