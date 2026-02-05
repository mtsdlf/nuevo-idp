package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/nuevo-idp/platform/config"
	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"github.com/nuevo-idp/platform/tracing"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/appenvprovhttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/controlplanehttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/gitproviderhttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/secretbindingshttp"
	internalworkflow "github.com/nuevo-idp/workflow-engine/internal/workflow"
)

func main() {
	logger, err := observability.NewLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	observability.InitMetrics()

	// Inicializar tracing global para el servicio.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownTracing, err := tracing.InitTracing(ctx, "workflow-engine")
	if err != nil {
		logger.Warn("failed to initialize tracing", zap.Error(err))
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTracing(shutdownCtx)
		}()
	}

	// HTTP health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteText(w, http.StatusOK, "ok")
	})
	mux.Handle("/metrics", promhttp.Handler())

	// Start Temporal worker in background
	go func() {
		if err := runTemporalWorker(logger); err != nil {
			// En entornos donde Temporal aún no está listo (p.ej., smoke-tests
			// levantando toda la stack), no derribamos el proceso HTTP completo;
			// registramos el error y dejamos vivo el health endpoint.
			logger.Error("temporal worker failed", zap.Error(err))
		}
	}()

	logger.Info("workflow-engine listening and connected to Temporal", zap.String("addr", ":8081"))

	// Handle graceful shutdown for HTTP server
	server := &http.Server{
		Addr:              ":8081",
		Handler:           otelhttp.NewHandler(observability.InstrumentHTTP(mux), "workflow-engine"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server failed", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutting down workflow-engine")
}

func runTemporalWorker(logger *zap.Logger) error {
	host := config.Get("TEMPORAL_HOST", "temporal:7233")

	c, err := client.Dial(client.Options{HostPort: host})
	if err != nil {
		return fmt.Errorf("dial temporal client: %w", err)
	}

	// Configure control-plane-api client for activities
	cpBaseURL := config.Get("CONTROL_PLANE_API_URL", "http://control-plane-api:8080")
	cpClient := controlplanehttp.NewClient(cpBaseURL)
	internalworkflow.SetControlPlaneClient(cpClient)
	internalworkflow.SetApplicationOnboardingPort(cpClient)
	internalworkflow.SetSecretRotationPort(cpClient)

	// Configure Git provider client (execution-workers)
	ewBaseURL := config.Get("EXECUTION_WORKERS_URL", "http://execution-workers:8082")
	internalworkflow.SetGitProvider(gitproviderhttp.NewClient(ewBaseURL))
	internalworkflow.SetAppEnvProvisioningProvider(appenvprovhttp.NewClient(ewBaseURL))
	internalworkflow.SetSecretBindingsRotationPort(secretbindingshttp.NewClient(ewBaseURL))

	w := worker.New(c, internalworkflow.ApplicationEnvironmentProvisioningTaskQueue, worker.Options{})
	w.RegisterWorkflow(internalworkflow.ApplicationEnvironmentProvisioning)
	w.RegisterWorkflow(internalworkflow.ApplicationOnboarding)
	w.RegisterWorkflow(internalworkflow.ApplicationActivation)
	w.RegisterWorkflow(internalworkflow.SecretRotation)
	w.RegisterActivity(internalworkflow.MaterializeRepositories)
	w.RegisterActivity(internalworkflow.ApplyBranchProtection)
	w.RegisterActivity(internalworkflow.ProvisionSecrets)
	w.RegisterActivity(internalworkflow.CreateSecretBindings)
	w.RegisterActivity(internalworkflow.VerifyGitOpsReconciliation)
	w.RegisterActivity(internalworkflow.FinalizeApplicationEnvironmentProvisioning)
	w.RegisterActivity(internalworkflow.CreateCodeRepositoryForApplication)
	w.RegisterActivity(internalworkflow.CreateDeploymentRepositoryForApplication)
	w.RegisterActivity(internalworkflow.CreateGitOpsIntegrationForApplication)
	w.RegisterActivity(internalworkflow.DeclareApplicationEnvironmentsForApplication)
	w.RegisterActivity(internalworkflow.TransitionApplicationToOnboarding)
	w.RegisterActivity(internalworkflow.TransitionApplicationToActive)
	w.RegisterActivity(internalworkflow.PerformSecretRotation)
	w.RegisterActivity(internalworkflow.CompleteSecretRotationActivity)
	w.RegisterActivity(internalworkflow.UpdateSecretBindingsForSecret)

	logger.Info("starting Temporal worker", zap.String("taskQueue", internalworkflow.ApplicationEnvironmentProvisioningTaskQueue))
	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("run temporal worker: %w", err)
	}

	return nil
}
