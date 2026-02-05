package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/httpapi"
	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/adapters/pgrepo"
	"github.com/nuevo-idp/control-plane-api/internal/application"
	"github.com/nuevo-idp/platform/config"
	"github.com/nuevo-idp/platform/observability"
	"github.com/nuevo-idp/platform/tracing"
)

func main() {
	logger, err := observability.NewLogger()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Inicializar tracing global para el servicio.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownTracing, err := tracing.InitTracing(ctx, "control-plane-api")
	if err != nil {
		log.Printf("failed to initialize tracing: %v", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTracing(shutdownCtx)
		}()
	}

	// Registrar métricas globales HTTP
	observability.InitMetrics()

	var teamRepo application.TeamRepository = memoryrepo.NewTeamRepository()

	dsn := config.Get("DATABASE_URL", "")
	if dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			log.Printf("failed to create pgx pool, using in-memory TeamRepository: %v", err)
		} else {
			if err := pool.Ping(ctx); err != nil {
				log.Printf("failed to ping Postgres, using in-memory TeamRepository: %v", err)
				pool.Close()
			} else {
				log.Printf("using Postgres-backed TeamRepository")
				teamRepo = pgrepo.NewTeamRepository(pool)
			}
		}
	}
	appRepo := memoryrepo.NewApplicationRepository()
	codeRepo := memoryrepo.NewCodeRepositoryRepository()
	envRepo := memoryrepo.NewEnvironmentRepository()
	appEnvRepo := memoryrepo.NewApplicationEnvironmentRepository()
	secretRepo := memoryrepo.NewSecretRepository()
	secretBindingRepo := memoryrepo.NewSecretBindingRepository()
	depRepo := memoryrepo.NewDeploymentRepositoryRepository()
	gitopsRepo := memoryrepo.NewGitOpsIntegrationRepository()

	services := &application.Services{
		Teams:                   teamRepo,
		Applications:            appRepo,
		CodeRepositories:        codeRepo,
		Environments:            envRepo,
		ApplicationEnvironments: appEnvRepo,
		Secrets:                 secretRepo,
		SecretBindings:          secretBindingRepo,
		DeploymentRepositories:  depRepo,
		GitOpsIntegrations:      gitopsRepo,
	}

	server := httpapi.NewServer(services, logger)
	handler := server.Routes()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("control-plane-api listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server error: %v", err)
	}
}
