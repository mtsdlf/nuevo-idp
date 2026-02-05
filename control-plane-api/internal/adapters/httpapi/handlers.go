package httpapi

import (
	"net/http"

	"github.com/nuevo-idp/control-plane-api/internal/application"
	"github.com/nuevo-idp/platform/config"
	"github.com/nuevo-idp/platform/httpx"
	perrors "github.com/nuevo-idp/platform/errors"
	"github.com/nuevo-idp/platform/observability"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

type Server struct {
	services *application.Services
	logger   *zap.Logger
}

func NewServer(services *application.Services, logger *zap.Logger) *Server {
	return &Server{services: services, logger: logger}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/commands/teams", s.createTeam)
	mux.HandleFunc("/commands/applications", s.createApplication)
	mux.HandleFunc("/commands/applications/approve", s.approveApplication)
	mux.HandleFunc("/commands/applications/start-onboarding", s.startApplicationOnboarding)
	mux.HandleFunc("/commands/applications/activate", s.activateApplication)
	mux.HandleFunc("/commands/applications/deprecate", s.deprecateApplication)
	mux.HandleFunc("/commands/environments", s.createEnvironment)
	mux.HandleFunc("/commands/application-environments", s.declareApplicationEnvironment)
	mux.HandleFunc("/commands/application-environments/complete-provisioning", s.completeApplicationEnvironmentProvisioning)
	mux.HandleFunc("/commands/secrets", s.createSecret)
	mux.HandleFunc("/commands/secrets/start-rotation", s.startSecretRotation)
	mux.HandleFunc("/commands/secrets/complete-rotation", s.completeSecretRotation)
	mux.HandleFunc("/commands/secret-bindings", s.declareSecretBinding)
	mux.HandleFunc("/commands/code-repositories", s.declareCodeRepository)
	mux.HandleFunc("/commands/deployment-repositories", s.declareDeploymentRepository)
	mux.HandleFunc("/commands/gitops-integrations", s.declareGitOpsIntegration)
	mux.HandleFunc("/queries/applications", s.getApplication)
	mux.HandleFunc("/queries/environments", s.getEnvironment)
	mux.HandleFunc("/queries/application-environments", s.getApplicationEnvironment)
	mux.Handle("/metrics", promhttp.Handler())

	instrumented := observability.InstrumentHTTP(mux)
	return otelhttp.NewHandler(instrumented, "control-plane-api")
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteText(w, http.StatusOK, "ok")
}

func (s *Server) getApplication(w http.ResponseWriter, r *http.Request) {
	if !httpx.RequireMethod(w, r, http.MethodGet) {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	app, err := s.services.GetApplication(r.Context(), id)
	if err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("getApplication error", zap.Error(err))
		writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, app)
}

func (s *Server) getEnvironment(w http.ResponseWriter, r *http.Request) {
	if !httpx.RequireMethod(w, r, http.MethodGet) {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	env, err := s.services.GetEnvironment(r.Context(), id)
	if err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("getEnvironment error", zap.Error(err))
		writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, env)
}

func (s *Server) getApplicationEnvironment(w http.ResponseWriter, r *http.Request) {
	if !httpx.RequireMethod(w, r, http.MethodGet) {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	ae, err := s.services.GetApplicationEnvironment(r.Context(), id)
	if err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("getApplicationEnvironment error", zap.Error(err))
		writeDomainError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ae)
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

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeDomainError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest

	switch {
	case perrors.IsKind(err, perrors.KindNotFound):
		status = http.StatusNotFound
	case perrors.IsKind(err, perrors.KindConflict):
		status = http.StatusConflict
	case perrors.IsKind(err, perrors.KindInternal):
		status = http.StatusInternalServerError
	case perrors.IsKind(err, perrors.KindDomain), perrors.IsKind(err, perrors.KindValidation):
		status = http.StatusBadRequest
	}

	code := perrors.Code(err)
	if code == "" {
		code = "unknown_error"
	}

	resp := errorResponse{
		Code:    code,
		Message: err.Error(),
	}

	httpx.WriteJSON(w, status, resp)
}
