package services

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestGroqProviderRequestTranslation verifies that a Gemini-shaped
// generateContent request is correctly translated to Groq's
// OpenAI-compatible chat-completions API and wrapped back into the
// Gemini response shape. Requires GROQ_API_KEY; skipped otherwise.
func TestGroqProviderRequestTranslation(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		t.Skip("GROQ_API_KEY not set")
	}

	p := NewOpenAICompatibleProvider("groq", apiKey, "openai/gpt-oss-120b", 4000, 0.7, 30)

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{
				{"text": `Reply with JSON only: {"ok": true}`},
			}},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  100,
			"temperature":      0,
			"responseMimeType": "application/json",
		},
	}
	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body, status, err := p.doRequest(ctx, jsonData)
	if err != nil {
		t.Fatalf("doRequest failed (status %d): %v", status, err)
	}

	// Response must be Gemini-shaped so existing parsers work
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		t.Fatalf("response not Gemini-shaped: %v", err)
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		t.Fatal("empty candidates in translated response")
	}
	t.Logf("response text: %s", geminiResp.Candidates[0].Content.Parts[0].Text)
}

// TestGetAIProviderGroq ensures the factory wires Groq config correctly.
func TestGetAIProviderGroq(t *testing.T) {
	p, err := GetAIProvider("groq", "key", "openai/gpt-oss-120b", 4000, 0.7, 10)
	if err != nil {
		t.Fatalf("GetAIProvider: %v", err)
	}
	if p.format != "openai" {
		t.Fatalf("expected format=openai, got %q", p.format)
	}
	if p.apiURL != "https://api.groq.com/openai/v1/chat/completions" {
		t.Fatalf("unexpected apiURL: %s", p.apiURL)
	}
}
