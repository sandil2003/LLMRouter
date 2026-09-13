package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CredentialProvider resolves API keys securely without storing plaintext in SQLite.
type CredentialProvider interface {
	GetAPIKey(ctx context.Context, providerID string) (string, error)
	SetAPIKey(ctx context.Context, providerID string, key string) error
}

// EnvCredentialProvider loads API keys from environment variables or local .credentials.json.
type EnvCredentialProvider struct {
	mu   sync.RWMutex
	keys map[string]string
}

func getCredentialPaths() []string {
	paths := []string{".credentials.json"}
	if _, err := os.Stat("backend"); err == nil {
		paths = append(paths, filepath.Join("backend", ".credentials.json"))
	}
	if _, err := os.Stat(".."); err == nil {
		paths = append(paths, filepath.Join("..", ".credentials.json"))
	}
	return paths
}

func NewEnvCredentialProvider() *EnvCredentialProvider {
	e := &EnvCredentialProvider{
		keys: make(map[string]string),
	}
	for _, p := range getCredentialPaths() {
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			if err := json.Unmarshal(data, &e.keys); err == nil && len(e.keys) > 0 {
				break
			}
		}
	}
	return e
}

func (e *EnvCredentialProvider) GetAPIKey(ctx context.Context, providerID string) (string, error) {
	e.mu.RLock()
	if val, ok := e.keys[providerID]; ok && val != "" {
		e.mu.RUnlock()
		return val, nil
	}
	e.mu.RUnlock()

	// Check environment variables: e.g. GEMINI_API_KEY, GROQ_API_KEY, OPENROUTER_API_KEY, OPENAI_API_KEY
	envVar := strings.ToUpper(providerID) + "_API_KEY"
	if val := os.Getenv(envVar); val != "" {
		return val, nil
	}
	envVar2 := "LLMROUTER_" + strings.ToUpper(providerID) + "_KEY"
	if val := os.Getenv(envVar2); val != "" {
		return val, nil
	}

	return "", nil
}

func (e *EnvCredentialProvider) SetAPIKey(ctx context.Context, providerID string, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if key == "" {
		delete(e.keys, providerID)
	} else {
		e.keys[providerID] = key
	}
	if data, err := json.MarshalIndent(e.keys, "", "  "); err == nil {
		for _, p := range getCredentialPaths() {
			_ = os.WriteFile(p, data, 0600)
		}
	}
	return nil
}

// Registry manages active in-memory provider instances.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	credStore CredentialProvider
}

func NewRegistry(credStore CredentialProvider) *Registry {
	if credStore == nil {
		credStore = NewEnvCredentialProvider()
	}
	return &Registry{
		providers: make(map[string]Provider),
		credStore: credStore,
	}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, id)
}

func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}

func (r *Registry) Credentials() CredentialProvider {
	return r.credStore
}

func (r *Registry) UpdateAPIKey(ctx context.Context, providerID, apiKey string) error {
	if err := r.credStore.SetAPIKey(ctx, providerID, apiKey); err != nil {
		return fmt.Errorf("set credential: %w", err)
	}
	return nil
}
