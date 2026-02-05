package httpapi

import (
	"net/http"

	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"go.uber.org/zap"
)

type declareCodeRepositoryRequest struct {
	ID            string `json:"id"`
	ApplicationID string `json:"applicationId"`
}

type declareDeploymentRepositoryRequest struct {
	ID             string `json:"id"`
	ApplicationID  string `json:"applicationId"`
	DeploymentModel string `json:"deploymentModel"`
}

type declareGitOpsIntegrationRequest struct {
	ID               string `json:"id"`
	ApplicationID    string `json:"applicationId"`
	DeploymentRepoID string `json:"deploymentRepositoryId"`
}

func (s *Server) declareCodeRepository(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req declareCodeRepositoryRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.ApplicationID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id and applicationId are required")
		return
	}

	if err := s.services.DeclareCodeRepository(r.Context(), req.ID, req.ApplicationID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("declareCodeRepository error", zap.Error(err))
		observability.ObserveDomainEvent("code_repository_declared", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("code_repository_declared", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) declareDeploymentRepository(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req declareDeploymentRepositoryRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.ApplicationID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id and applicationId are required")
		return
	}

	if err := s.services.DeclareDeploymentRepository(r.Context(), req.ID, req.ApplicationID, req.DeploymentModel, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("declareDeploymentRepository error", zap.Error(err))
		observability.ObserveDomainEvent("deployment_repository_declared", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("deployment_repository_declared", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) declareGitOpsIntegration(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req declareGitOpsIntegrationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.ApplicationID == "" || req.DeploymentRepoID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id, applicationId and deploymentRepositoryId are required")
		return
	}

	if err := s.services.DeclareGitOpsIntegration(r.Context(), req.ID, req.ApplicationID, req.DeploymentRepoID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("declareGitOpsIntegration error", zap.Error(err))
		observability.ObserveDomainEvent("gitops_integration_declared", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("gitops_integration_declared", "success")
	w.WriteHeader(http.StatusCreated)
}
