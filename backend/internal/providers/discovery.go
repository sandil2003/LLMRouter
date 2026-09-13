package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DiscoveredModel represents a model discovered from provider API or internet catalog.
type DiscoveredModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	ContextLength int    `json:"context_length,omitempty"`
	IsActive      bool   `json:"is_active"`
}

// DiscoverModelsResult summarizes the result of model discovery.
type DiscoverModelsResult struct {
	ProviderID   string            `json:"provider_id"`
	ProviderName string            `json:"provider_name"`
	Source       string            `json:"source"` // "live_api", "internet_catalog", "curated_catalog"
	Models       []DiscoveredModel `json:"models"`
}

// Curated standard fallbacks by provider family
var curatedFallbacks = map[string][]DiscoveredModel{
	"gemini": {
		{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Description: "Ultra-fast multimodal model with high performance", ContextLength: 1048576},
		{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Description: "Advanced reasoning and complex multimodal understanding", ContextLength: 2097152},
		{ID: "gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash Lite", Description: "Lightweight and cost-optimized for high throughput", ContextLength: 1048576},
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", Description: "Fast next-gen multimodal speed and quality", ContextLength: 1048576},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Description: "High efficiency, 1M token context window", ContextLength: 1048576},
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Description: "Versatile large-context reasoning model", ContextLength: 2097152},
	},
	"openai": {
		{ID: "gpt-4o", Name: "GPT-4o", Description: "Flagship high-intelligence multimodal flagship model", ContextLength: 128000},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Fast, affordable, intelligent small model", ContextLength: 128000},
		{ID: "o1", Name: "o1", Description: "Advanced reasoning model for math, coding, and STEM", ContextLength: 200000},
		{ID: "o1-mini", Name: "o1 Mini", Description: "Fast reasoning model optimized for STEM and code", ContextLength: 128000},
		{ID: "o3-mini", Name: "o3 Mini", Description: "Next-gen compact reasoning model with high speed", ContextLength: 200000},
		{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Description: "Previous generation GPT-4 with vision capability", ContextLength: 128000},
	},
	"groq": {
		{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", Description: "State of the art 70B parameter open model on Groq LPU", ContextLength: 128000},
		{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", Description: "Ultra-low latency instant response model", ContextLength: 128000},
		{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Description: "High-speed Mixture-of-Experts model", ContextLength: 32768},
		{ID: "gemma2-9b-it", Name: "Gemma 2 9B", Description: "Google open model instruction tuned on Groq", ContextLength: 8192},
		{ID: "deepseek-r1-distill-llama-70b", Name: "DeepSeek R1 Distill Llama 70B", Description: "DeepSeek reasoning model distilled into Llama 70B", ContextLength: 128000},
	},
	"openrouter": {
		{ID: "meta-llama/llama-3.3-70b-instruct", Name: "Meta: Llama 3.3 70B Instruct", Description: "Top-tier open instruction model", ContextLength: 131072},
		{ID: "google/gemini-2.5-flash", Name: "Google: Gemini 2.5 Flash", Description: "Google flagship lightweight model via OpenRouter", ContextLength: 1048576},
		{ID: "deepseek/deepseek-r1", Name: "DeepSeek: R1", Description: "Full open weights reasoning model", ContextLength: 163840},
		{ID: "anthropic/claude-3.5-sonnet", Name: "Anthropic: Claude 3.5 Sonnet", Description: "Exceptional coding, reasoning, and visual capability", ContextLength: 200000},
		{ID: "openai/gpt-4o", Name: "OpenAI: GPT-4o", Description: "OpenAI flagship multi-modal model", ContextLength: 128000},
	},
}

// DiscoverModels searches for models for a given provider configuration.
// It prioritizes:
// 1. Live Provider API (authenticated or direct)
// 2. OpenRouter Public Internet Catalog (unauthenticated live internet catalog)
// 3. Curated built-in fallback catalog
func DiscoverModels(ctx context.Context, providerID, providerName, baseURL, apiKey string, activeModels []string) DiscoverModelsResult {
	activeMap := make(map[string]bool)
	for _, m := range activeModels {
		activeMap[strings.ToLower(strings.TrimSpace(m))] = true
	}

	result := DiscoverModelsResult{
		ProviderID:   providerID,
		ProviderName: providerName,
	}

	providerFamily := detectProviderFamily(providerName)
	fallbacks := curatedFallbacks[providerFamily]

	// 1. Attempt Live Provider API
	if modelsList, err := fetchLiveProviderModels(ctx, providerName, baseURL, apiKey); err == nil && len(modelsList) > 0 {
		result.Source = "live_api"
		result.Models = markAndSortModels(modelsList, activeMap)
		return result
	}

	// 2. Attempt Public Internet Catalog (OpenRouter live models API)
	if modelsList, err := fetchInternetCatalogModels(ctx, providerName); err == nil && len(modelsList) > 0 {
		result.Source = "internet_catalog"
		combined := append([]DiscoveredModel{}, modelsList...)
		combined = append(combined, fallbacks...)
		result.Models = markAndSortModels(combined, activeMap)
		return result
	}

	// 3. Fallback to Curated Catalog
	if len(fallbacks) > 0 {
		result.Source = "curated_catalog"
		result.Models = markAndSortModels(fallbacks, activeMap)
		return result
	}

	// Generic default if completely unknown provider
	result.Source = "curated_catalog"
	result.Models = markAndSortModels([]DiscoveredModel{
		{ID: "default", Name: "Default Provider Model", Description: "Standard model configured for this endpoint"},
	}, activeMap)
	return result
}

func detectProviderFamily(providerName string) string {
	lower := strings.ToLower(providerName)
	switch {
	case strings.Contains(lower, "gemini") || strings.Contains(lower, "google"):
		return "gemini"
	case strings.Contains(lower, "groq"):
		return "groq"
	case strings.Contains(lower, "openrouter"):
		return "openrouter"
	case strings.Contains(lower, "openai"):
		return "openai"
	default:
		return "openai"
	}
}

// fetchLiveProviderModels queries the provider's /models endpoint directly.
func fetchLiveProviderModels(ctx context.Context, providerName, baseURL, apiKey string) ([]DiscoveredModel, error) {
	family := detectProviderFamily(providerName)
	client := &http.Client{Timeout: 7 * time.Second}

	switch family {
	case "gemini":
		if apiKey == "" {
			return nil, fmt.Errorf("gemini requires api key for live discovery")
		}
		endpoint := baseURL
		if endpoint == "" {
			endpoint = "https://generativelanguage.googleapis.com/v1beta"
		}
		endpoint = strings.TrimRight(endpoint, "/") + "/models?key=" + apiKey

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("gemini live api returned %d", resp.StatusCode)
		}

		var geminiResp struct {
			Models []struct {
				Name                       string   `json:"name"`
				DisplayName                string   `json:"displayName"`
				Description                string   `json:"description"`
				InputTokenLimit            int      `json:"inputTokenLimit"`
				SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
			return nil, err
		}

		var results []DiscoveredModel
		for _, m := range geminiResp.Models {
			// Only include text/content generation models
			supportsGen := false
			for _, method := range m.SupportedGenerationMethods {
				if method == "generateContent" {
					supportsGen = true
					break
				}
			}
			if !supportsGen {
				continue
			}

			// Clean ID: strip "models/" prefix
			cleanID := strings.TrimPrefix(m.Name, "models/")
			displayName := m.DisplayName
			if displayName == "" {
				displayName = cleanID
			}

			results = append(results, DiscoveredModel{
				ID:            cleanID,
				Name:          displayName,
				Description:   m.Description,
				ContextLength: m.InputTokenLimit,
			})
		}
		return results, nil

	default: // OpenAI, Groq, OpenRouter, and standard compatible endpoints
		endpoint := baseURL
		if endpoint == "" {
			if family == "groq" {
				endpoint = "https://api.groq.com/openai/v1"
			} else if family == "openrouter" {
				endpoint = "https://openrouter.ai/api/v1"
			} else {
				endpoint = "https://api.openai.com/v1"
			}
		}
		endpoint = strings.TrimRight(endpoint, "/") + "/models"

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("provider models endpoint returned %d", resp.StatusCode)
		}

		var oaiResp struct {
			Data []struct {
				ID          string `json:"id"`
				Name        string `json:"name,omitempty"`
				Description string `json:"description,omitempty"`
				ContextLen  int    `json:"context_length,omitempty"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
			return nil, err
		}

		var results []DiscoveredModel
		for _, item := range oaiResp.Data {
			idLower := strings.ToLower(item.ID)
			// Filter out pure audio/whisper/embedding/moderation/tts models if OpenAI
			if family == "openai" {
				if strings.Contains(idLower, "whisper") ||
					strings.Contains(idLower, "tts") ||
					strings.Contains(idLower, "dall-e") ||
					strings.Contains(idLower, "embedding") ||
					strings.Contains(idLower, "moderation") {
					continue
				}
			}

			displayName := item.Name
			if displayName == "" {
				displayName = item.ID
			}

			results = append(results, DiscoveredModel{
				ID:            item.ID,
				Name:          displayName,
				Description:   item.Description,
				ContextLength: item.ContextLen,
			})
		}
		return results, nil
	}
}

// fetchInternetCatalogModels searches OpenRouter's free public model catalog over the internet.
func fetchInternetCatalogModels(ctx context.Context, providerName string) ([]DiscoveredModel, error) {
	client := &http.Client{Timeout: 7 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LLMRouter/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter catalog returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var catalogResp struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &catalogResp); err != nil {
		return nil, err
	}

	family := detectProviderFamily(providerName)
	var keywords []string
	switch family {
	case "gemini":
		keywords = []string{"google/", "gemini"}
	case "groq":
		keywords = []string{"groq", "llama", "mixtral"}
	case "openrouter":
		// OpenRouter supports all models
		keywords = []string{}
	case "openai":
		keywords = []string{"openai/", "gpt-"}
	default:
		keywords = []string{strings.ToLower(providerName)}
	}

	var matched []DiscoveredModel
	for _, m := range catalogResp.Data {
		idLower := strings.ToLower(m.ID)
		nameLower := strings.ToLower(m.Name)

		isMatch := len(keywords) == 0
		for _, kw := range keywords {
			if strings.Contains(idLower, kw) || strings.Contains(nameLower, kw) {
				isMatch = true
				break
			}
		}

		if !isMatch {
			continue
		}

		// For direct native providers (like Gemini or OpenAI), offer clean ID without provider prefix as well
		modelID := m.ID
		if family == "gemini" && strings.HasPrefix(modelID, "google/") {
			modelID = strings.TrimPrefix(modelID, "google/")
		} else if family == "openai" && strings.HasPrefix(modelID, "openai/") {
			modelID = strings.TrimPrefix(modelID, "openai/")
		}

		displayName := m.Name
		if displayName == "" {
			displayName = modelID
		}

		matched = append(matched, DiscoveredModel{
			ID:            modelID,
			Name:          displayName,
			Description:   m.Description,
			ContextLength: m.ContextLength,
		})

		if len(matched) >= 30 {
			break
		}
	}

	return matched, nil
}

func markAndSortModels(models []DiscoveredModel, activeMap map[string]bool) []DiscoveredModel {
	seen := make(map[string]bool)
	var deduped []DiscoveredModel

	for _, m := range models {
		cleanID := strings.TrimSpace(m.ID)
		if cleanID == "" || seen[cleanID] {
			continue
		}
		seen[cleanID] = true

		m.IsActive = activeMap[strings.ToLower(cleanID)]
		deduped = append(deduped, m)
	}

	sort.Slice(deduped, func(i, j int) bool {
		// Active models come first, then alphabetical by ID
		if deduped[i].IsActive != deduped[j].IsActive {
			return deduped[i].IsActive
		}
		return deduped[i].ID < deduped[j].ID
	})

	return deduped
}
