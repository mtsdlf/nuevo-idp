package main

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"

	"github.com/nuevo-idp/execution-workers/internal/harbor"
	"github.com/nuevo-idp/platform/config"
	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"github.com/nuevo-idp/platform/tracing"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type createRepoRequest struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

const internalAuthHeader = "X-Internal-Token"

// requireInternalAuth aplica autenticación interna para llamadas servicio-a-servicio.
// Si INTERNAL_AUTH_TOKEN no está configurado, no se aplica enforcement (modo dev).
func requireInternalAuth(w http.ResponseWriter, r *http.Request) bool {
	token, ok := config.Require("INTERNAL_AUTH_TOKEN")
	if !ok || token == "" {
		return true
	}
	if r.Header.Get(internalAuthHeader) != token {
		httpx.WriteText(w, http.StatusUnauthorized, "missing or invalid internal auth token")
		return false
	}
	return true
}

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
	shutdownTracing, err := tracing.InitTracing(ctx, "execution-workers")
	if err != nil {
		logger.Warn("failed to initialize tracing", zap.Error(err))
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTracing(shutdownCtx)
		}()
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteText(w, http.StatusOK, "ok")
	})
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/github/repos", func(w http.ResponseWriter, r *http.Request) {
		handleCreateGitHubRepo(logger, w, r)
	})
	mux.HandleFunc("/appenv/branch-protection", func(w http.ResponseWriter, r *http.Request) {
		handleAppEnvBranchProtection(logger, w, r)
	})
	mux.HandleFunc("/appenv/secrets", func(w http.ResponseWriter, r *http.Request) {
		handleAppEnvSecrets(logger, w, r)
	})
	mux.HandleFunc("/appenv/secret-bindings", func(w http.ResponseWriter, r *http.Request) {
		handleAppEnvSecretBindings(logger, w, r)
	})
	mux.HandleFunc("/appenv/gitops-verify", func(w http.ResponseWriter, r *http.Request) {
		handleAppEnvGitOpsVerify(logger, w, r)
	})

	mux.HandleFunc("/secrets/bindings/update", func(w http.ResponseWriter, r *http.Request) {
		handleSecretBindingsUpdate(logger, w, r)
	})

	server := &http.Server{
		Addr:         ":8082",
		Handler:      otelhttp.NewHandler(observability.InstrumentHTTP(mux), "execution-workers"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("execution-workers listening", zap.String("addr", server.Addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("http server failed", zap.Error(err))
	}
}

func handleCreateGitHubRepo(baseLogger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	if !requireInternalAuth(w, r) {
		return
	}

	ctx, span := tracing.StartSpan(r.Context(), "execution-workers.github.CreateRepository")
	defer span.End()

	logger := observability.LoggerWithTrace(ctx, baseLogger)
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req createRepoRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json body") {
		return
	}
	span.SetAttributes(
		attribute.String("git.owner", req.Owner),
		attribute.String("git.repo", req.Name),
	)
	if req.Name == "" {
		httpx.WriteText(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Owner == "" {
		httpx.WriteText(w, http.StatusBadRequest, "owner is required")
		return
	}

	token, ok := config.Require("GITHUB_TOKEN")
	if !ok {
		logger.Error("GITHUB_TOKEN not configured")
		observability.ObserveDomainEvent("github_repo_created", "error")
		httpx.WriteText(w, http.StatusInternalServerError, "GITHUB_TOKEN not configured")
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	if apiURL := config.Get("GITHUB_API_URL", ""); apiURL != "" {
		if u, err := url.Parse(apiURL); err == nil {
			client.BaseURL = u
		} else {
			logger.Warn("invalid GITHUB_API_URL, falling back to default", zap.Error(err))
		}
	}

	repo := &github.Repository{
		Name:    github.String(req.Name),
		Private: github.Bool(req.Private),
	}

	created, _, err := client.Repositories.Create(ctx, "", repo)
	if err != nil {
		logger.Error("error creating repo in GitHub", zap.Error(err))
		observability.ObserveDomainEvent("github_repo_created", "error")
		httpx.WriteText(w, http.StatusBadGateway, "failed to create repository in GitHub")
		return
	}

	observability.ObserveDomainEvent("github_repo_created", "success")
	httpx.WriteJSON(w, http.StatusCreated, created)
}

type appEnvRequest struct {
	ApplicationEnvironmentID string `json:"applicationEnvironmentId"`
}

type secretBindingsUpdateRequest struct {
	SecretID string `json:"secretId"`
}

func handleAppEnvBranchProtection(baseLogger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	if !requireInternalAuth(w, r) {
		return
	}

	ctx, span := tracing.StartSpan(r.Context(), "execution-workers.appenv.ApplyBranchProtection")
	defer span.End()

	logger := observability.LoggerWithTrace(ctx, baseLogger)
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req appEnvRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json body") {
		return
	}
	if req.ApplicationEnvironmentID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "applicationEnvironmentId is required")
		return
	}

	appEnvID := req.ApplicationEnvironmentID
	span.SetAttributes(attribute.String("appenv.id", appEnvID))

	token, ok := config.Require("GITHUB_TOKEN")
	if !ok {
		logger.Error("GITHUB_TOKEN not configured for branch protection")
		observability.ObserveDomainEvent("appenv_branch_protection_applied", "error")
		httpx.WriteText(w, http.StatusInternalServerError, "GITHUB_TOKEN not configured")
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	if apiURL := config.Get("GITHUB_API_URL", ""); apiURL != "" {
		if u, err := url.Parse(apiURL); err == nil {
			client.BaseURL = u
		} else {
			logger.Warn("invalid GITHUB_API_URL, falling back to default", zap.Error(err))
		}
	}

	owner := "platform"
	repo := "appenv-" + appEnvID
	branch := "main"

	span.SetAttributes(
		attribute.String("git.owner", owner),
		attribute.String("git.repo", repo),
		attribute.String("git.branch", branch),
	)

	protReq := &github.ProtectionRequest{
		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			RequiredApprovingReviewCount: 1,
		},
	}

	_, _, err := client.Repositories.UpdateBranchProtection(ctx, owner, repo, branch, protReq)
	if err != nil {
		logger.Error("error applying branch protection in GitHub", zap.Error(err),
			zap.String("owner", owner), zap.String("repo", repo), zap.String("branch", branch))
		observability.ObserveDomainEvent("appenv_branch_protection_applied", "error")
		httpx.WriteText(w, http.StatusBadGateway, "failed to apply branch protection in GitHub")
		return
	}

	observability.ObserveDomainEvent("appenv_branch_protection_applied", "success")
	w.WriteHeader(http.StatusAccepted)
}

// handleAppEnvAction encapsula el patrón común de handlers appenv que delegan
// en endpoints externos configurados por variable de entorno.
func handleAppEnvAction(
	baseLogger *zap.Logger,
	w http.ResponseWriter,
	r *http.Request,
	spanName string,
	endpointEnvVar string,
	domainEvent string,
	endpointNotConfiguredLog string,
	endpointNotConfiguredText string,
	requestErrorLog string,
	requestErrorText string,
	callErrorLog string,
	callErrorText string,
	non2xxLog string,
	non2xxText string,
) {
	if !requireInternalAuth(w, r) {
		return
	}

	ctx, span := tracing.StartSpan(r.Context(), spanName)
	defer span.End()

	logger := observability.LoggerWithTrace(ctx, baseLogger)
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req appEnvRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json body") {
		return
	}
	if req.ApplicationEnvironmentID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "applicationEnvironmentId is required")
		return
	}

	appEnvID := req.ApplicationEnvironmentID
	span.SetAttributes(attribute.String("appenv.id", appEnvID))

	endpoint := config.Get(endpointEnvVar, "")
	if endpoint == "" {
		logger.Error(endpointNotConfiguredLog)
		observability.ObserveDomainEvent(domainEvent, "error")
		httpx.WriteText(w, http.StatusInternalServerError, endpointNotConfiguredText)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	body := bytes.NewBufferString("{\"applicationEnvironmentId\":\"" + appEnvID + "\"}")
	reqOut, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		logger.Error(requestErrorLog, zap.Error(err))
		observability.ObserveDomainEvent(domainEvent, "error")
		httpx.WriteText(w, http.StatusBadGateway, requestErrorText)
		return
	}
	reqOut.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(reqOut)
	if err != nil {
		logger.Error(callErrorLog, zap.Error(err))
		observability.ObserveDomainEvent(domainEvent, "error")
		httpx.WriteText(w, http.StatusBadGateway, callErrorText)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error(non2xxLog, zap.Int("status", resp.StatusCode))
		observability.ObserveDomainEvent(domainEvent, "error")
		httpx.WriteText(w, http.StatusBadGateway, non2xxText)
		return
	}

	observability.ObserveDomainEvent(domainEvent, "success")
	w.WriteHeader(http.StatusAccepted)
}

func handleAppEnvSecrets(baseLogger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	handleAppEnvAction(
		baseLogger,
		w,
		r,
		"execution-workers.appenv.ProvisionSecrets",
		"APPENV_SECRETS_ENDPOINT",
		"appenv_secrets_provisioned",
		"APPENV_SECRETS_ENDPOINT not configured",
		"APPENV_SECRETS_ENDPOINT not configured",
		"error creating appenv secrets request",
		"failed to provision appenv secrets",
		"error calling appenv secrets endpoint",
		"failed to provision appenv secrets",
		"appenv secrets endpoint returned non-2xx",
		"failed to provision appenv secrets",
	)
}

func handleAppEnvSecretBindings(baseLogger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	handleAppEnvAction(
		baseLogger,
		w,
		r,
		"execution-workers.appenv.CreateSecretBindings",
		"APPENV_SECRET_BINDINGS_ENDPOINT",
		"appenv_secret_bindings_created",
		"APPENV_SECRET_BINDINGS_ENDPOINT not configured",
		"APPENV_SECRET_BINDINGS_ENDPOINT not configured",
		"error creating appenv secret bindings request",
		"failed to create appenv secret bindings",
		"error calling appenv secret bindings endpoint",
		"failed to create appenv secret bindings",
		"appenv secret bindings endpoint returned non-2xx",
		"failed to create appenv secret bindings",
	)
}

func handleAppEnvGitOpsVerify(baseLogger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	handleAppEnvAction(
		baseLogger,
		w,
		r,
		"execution-workers.appenv.VerifyGitOpsReconciliation",
		"APPENV_GITOPS_VERIFY_ENDPOINT",
		"appenv_gitops_verified",
		"APPENV_GITOPS_VERIFY_ENDPOINT not configured",
		"APPENV_GITOPS_VERIFY_ENDPOINT not configured",
		"error creating appenv gitops verify request",
		"failed to verify appenv gitops reconciliation",
		"error calling appenv gitops verify endpoint",
		"failed to verify appenv gitops reconciliation",
		"appenv gitops verify endpoint returned non-2xx",
		"failed to verify appenv gitops reconciliation",
	)
}

// handleSecretBindingsUpdate actúa como stub para el proveedor de actualización
// de SecretBindings. A futuro debería integrar con Harbor (u otros vendors)
// para propagar tokens rotados a todos los targets. Por ahora sólo valida
// input, emite un evento de dominio y responde 202.
func handleSecretBindingsUpdate(baseLogger *zap.Logger, w http.ResponseWriter, r *http.Request) {
	if !requireInternalAuth(w, r) {
		return
	}

	ctx, span := tracing.StartSpan(r.Context(), "execution-workers.secrets.UpdateBindingsForSecret")
	defer span.End()

	logger := observability.LoggerWithTrace(ctx, baseLogger)
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req secretBindingsUpdateRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json body") {
		return
	}
	if req.SecretID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "secretId is required")
		return
	}

	span.SetAttributes(attribute.String("secret.id", req.SecretID))

	// Intento best-effort de obtener un cliente Harbor desde configuración.
	// Si no está configurado, dejamos registro y respondemos 202 igualmente
	// para no bloquear el flujo de desarrollo/local.
	var token string
	hClient, err := harbor.NewClientFromEnv()
	if err != nil {
		logger.Warn("Harbor client not configured; skipping robot token rotation", zap.Error(err))
	} else {
		rotatedToken, err := hClient.RotateRobotToken(ctx)
		if err != nil {
			logger.Error("error rotating Harbor robot token", zap.Error(err))
		} else {
			logger.Info("Harbor robot token rotated successfully")
			token = rotatedToken
		}
	}

	// Si tenemos un token nuevo y un endpoint configurado, delegamos en un
	// sistema externo la propagación del token a todos los SecretBindings
	// asociados al Secret.
	bindingsEndpoint := config.Get("SECRET_BINDINGS_UPDATE_ENDPOINT", "")
	if bindingsEndpoint == "" {
		logger.Info("SECRET_BINDINGS_UPDATE_ENDPOINT not configured; skipping bindings propagation")
	} else if token == "" {
		logger.Warn("no rotated token available; skipping bindings propagation")
	} else {
		propCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		body := bytes.NewBufferString("{\"secretId\":\"" + req.SecretID + "\",\"token\":\"" + token + "\"}")
		outReq, err := http.NewRequestWithContext(propCtx, http.MethodPost, bindingsEndpoint, body)
		if err != nil {
			logger.Error("error creating secret bindings propagation request", zap.Error(err))
		} else {
			outReq.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(outReq)
			if err != nil {
				logger.Error("error calling secret bindings propagation endpoint", zap.Error(err))
			} else {
				defer func() {
					_ = resp.Body.Close()
				}()
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					logger.Error("secret bindings propagation endpoint returned non-2xx", zap.Int("status", resp.StatusCode))
				} else {
					logger.Info("secret bindings propagation completed successfully", zap.Int("status", resp.StatusCode))
				}
			}
		}
	}

	logger.Info("accepted secret bindings update request", zap.String("secretId", req.SecretID))
	w.WriteHeader(http.StatusAccepted)
	observability.ObserveDomainEvent("secret_bindings_update_accepted", "success")
}
