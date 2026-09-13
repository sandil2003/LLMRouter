package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/llmrouter/backend/internal/api/handlers"
	"github.com/llmrouter/backend/internal/api/middleware"
	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/router"
)

type ServerDeps struct {
	DB           *database.DB
	ProviderRepo *repository.ProviderRepository
	ModelRepo    *repository.ModelRepository
	RoutingRepo  *repository.RoutingRepository
	UsageRepo    *repository.UsageRepository
	LogRepo      *repository.LogRepository
	Registry     *providers.Registry
	Engine       *router.Engine
}

func SetupRoutes(deps *ServerDeps) http.Handler {
	r := chi.NewRouter()

	// Global Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recover)
	r.Use(middleware.CORS())

	// Handlers
	healthH := handlers.NewHealthHandler(deps.DB)
	providersH := handlers.NewProvidersHandler(
		deps.ProviderRepo,
		deps.ModelRepo,
		deps.Registry,
		deps.Engine.RateLimiter(),
		deps.Engine.CircuitBreaker(),
	)
	modelsH := handlers.NewModelsHandler(deps.ModelRepo)
	routingH := handlers.NewRoutingHandler(deps.RoutingRepo, deps.ProviderRepo)
	usageH := handlers.NewUsageHandler(deps.UsageRepo)
	logsH := handlers.NewLogsHandler(deps.LogRepo)
	chatH := handlers.NewChatHandler(deps.Engine)

	// API Routes
	r.Route("/api", func(api chi.Router) {
		api.Get("/health", healthH.Health)

		api.Route("/providers", func(p chi.Router) {
			p.Get("/", providersH.List)
			p.Post("/", providersH.Create)
			p.Get("/discover-models", providersH.DiscoverModels)

			p.Route("/{id}", func(sub chi.Router) {
				sub.Put("/", providersH.Update)
				sub.Delete("/", providersH.Delete)
				sub.Post("/test", providersH.TestConnection)
				sub.Get("/available-models", providersH.GetAvailableModels)
				sub.Post("/models", providersH.AddModel)
				sub.Delete("/models/{model}", providersH.RemoveModel)
			})
		})

		api.Get("/models", modelsH.List)

		api.Get("/routing", routingH.Get)
		api.Put("/routing", routingH.Update)

		api.Get("/usage", usageH.Get)
		api.Get("/logs", logsH.List)
	})

	// OpenAI Gateway Compatibility
	r.Route("/v1", func(v1 chi.Router) {
		v1.Post("/chat/completions", chatH.Completions)
	})

	return r
}
