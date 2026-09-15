package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/llmrouter/backend/internal/api"
	"github.com/llmrouter/backend/internal/background"
	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/classifier"
	"github.com/llmrouter/backend/internal/config"
	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/langcache"
	"github.com/llmrouter/backend/internal/logging"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/providers/gemini"
	"github.com/llmrouter/backend/internal/providers/groq"
	"github.com/llmrouter/backend/internal/providers/openai"
	"github.com/llmrouter/backend/internal/providers/openrouter"
	"github.com/llmrouter/backend/internal/ratelimit"
	"github.com/llmrouter/backend/internal/registry"
	"github.com/llmrouter/backend/internal/router"
)

func main() {
	cfg := config.Load()
	flag.Parse()

	logger := logging.Setup(cfg.LogLevel)
	slog.SetDefault(logger)

	slog.Info("Starting LLMRouter Backend Gateway...",
		"host", cfg.Host,
		"port", cfg.Port,
		"db", cfg.DatabasePath,
	)

	// 1. Database Initialization
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 2. Repositories
	providerRepo := repository.NewProviderRepository(db)
	modelRepo := repository.NewModelRepository(db)
	routingRepo := repository.NewRoutingRepository(db)
	usageRepo := repository.NewUsageRepository(db)
	logRepo := repository.NewLogRepository(db)

	// 3. Provider Registry & Credential Store
	provRegistry := providers.NewRegistry(nil)

	// 4. Resilience Layer
	rateLimiter := ratelimit.NewTracker(60 * time.Second)
	circuitBreaker := circuitbreaker.NewManager(3, 60*time.Second)

	// 5. Intelligent Routing Pipeline Components
	fastClassifier := classifier.NewClassifier()
	cacheClient := langcache.NewLangCache(langcache.Config{
		Endpoint: os.Getenv("LANGCACHE_ENDPOINT"),
		CacheID:  os.Getenv("LANGCACHE_CACHE_ID"),
		APIKey:   os.Getenv("LANGCACHE_API_KEY"),
		RedisURL: os.Getenv("REDIS_URL"),
	})
	modelRegistry := registry.NewModelRegistry()
	policyEngine := registry.NewPolicyEngine(modelRegistry, registry.DefaultPolicyWeights())

	// 6. Routing Engine
	engine := router.NewEngine(
		providerRepo,
		modelRepo,
		routingRepo,
		usageRepo,
		logRepo,
		provRegistry,
		rateLimiter,
		circuitBreaker,
		fastClassifier,
		cacheClient,
		policyEngine,
		modelRegistry,
	)

	// 7. Decoupled Offline Background Sync Worker
	syncWorker := background.NewSyncWorker(modelRegistry, provRegistry, 12*time.Hour)
	syncWorker.Start()

	// 8. Bootstrap default providers if database is fresh
	ctx := context.Background()
	bootstrapDefaultProviders(ctx, providerRepo, modelRepo, provRegistry)

	// 9. HTTP Routes & Server
	deps := &api.ServerDeps{
		DB:           db,
		ProviderRepo: providerRepo,
		ModelRepo:    modelRepo,
		RoutingRepo:  routingRepo,
		UsageRepo:    usageRepo,
		LogRepo:      logRepo,
		Registry:     provRegistry,
		Engine:       engine,
		SyncWorker:   syncWorker,
	}

	handler := api.SetupRoutes(deps)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Graceful Shutdown on OS Signal
	shutdownDone := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigChan
		slog.Info("Shutdown signal received", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error during server shutdown", "error", err)
		}
		close(shutdownDone)
	}()

	slog.Info(fmt.Sprintf("LLMRouter gateway listening on http://%s", addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}

	<-shutdownDone
	slog.Info("LLMRouter Gateway terminated cleanly.")
}

func bootstrapDefaultProviders(
	ctx context.Context,
	pRepo *repository.ProviderRepository,
	mRepo *repository.ModelRepository,
	registry *providers.Registry,
) {
	existing, err := pRepo.GetAll(ctx)
	if err == nil && len(existing) > 0 {
		// Instantiate existing providers into memory registry
		for _, cfg := range existing {
			apiKey, _ := registry.Credentials().GetAPIKey(ctx, cfg.ID)
			instantiateProvider(cfg, apiKey, registry)
		}
		return
	}

	slog.Info("Seeding initial default provider configurations...")

	defaults := []struct {
		config models.ProviderConfig
		models []string
	}{
		{
			config: models.ProviderConfig{
				ID:       "gemini",
				Name:     "Gemini",
				Enabled:  true,
				Priority: 1,
				BaseURL:  "https://generativelanguage.googleapis.com/v1beta",
			},
			models: []string{"gemini-1.5-flash", "gemini-1.5-pro"},
		},
		{
			config: models.ProviderConfig{
				ID:       "groq",
				Name:     "Groq",
				Enabled:  true,
				Priority: 2,
				BaseURL:  "https://api.groq.com/openai/v1",
			},
			models: []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant"},
		},
		{
			config: models.ProviderConfig{
				ID:       "openrouter",
				Name:     "OpenRouter",
				Enabled:  true,
				Priority: 3,
				BaseURL:  "https://openrouter.ai/api/v1",
			},
			models: []string{"meta-llama/llama-3.3-70b-instruct", "google/gemini-flash-1.5"},
		},
		{
			config: models.ProviderConfig{
				ID:       "openai",
				Name:     "OpenAI",
				Enabled:  true,
				Priority: 4,
				BaseURL:  "https://api.openai.com/v1",
			},
			models: []string{"gpt-4o", "gpt-4o-mini"},
		},
	}

	for _, d := range defaults {
		if err := pRepo.Create(ctx, &d.config); err == nil {
			for _, m := range d.models {
				_ = mRepo.Upsert(ctx, &models.ModelConfig{
					ID:         d.config.ID + ":" + m,
					ProviderID: d.config.ID,
					Name:       m,
					Enabled:    true,
				})
			}
			apiKey, _ := registry.Credentials().GetAPIKey(ctx, d.config.ID)
			instantiateProvider(d.config, apiKey, registry)
		}
	}
}

func instantiateProvider(cfg models.ProviderConfig, apiKey string, registry *providers.Registry) {
	switch cfg.ID {
	case "gemini":
		registry.Register(gemini.New(cfg.ID, apiKey, cfg.BaseURL))
	case "groq":
		registry.Register(groq.New(cfg.ID, apiKey, cfg.BaseURL))
	case "openrouter":
		registry.Register(openrouter.New(cfg.ID, apiKey, cfg.BaseURL))
	default:
		registry.Register(openai.New(cfg.ID, cfg.Name, apiKey, cfg.BaseURL))
	}
}
