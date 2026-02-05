package httpapi

import (
	"net/http"

	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"go.uber.org/zap"
)

type createEnvironmentRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type declareApplicationEnvironmentRequest struct {
	ID            string `json:"id"`
	ApplicationID string `json:"applicationId"`
	EnvironmentID string `json:"environmentId"`
}

type completeApplicationEnvironmentProvisioningRequest struct {
	ID string `json:"id"`
}

func (s *Server) createEnvironment(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req createEnvironmentRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.Name == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id and name are required")
		return
	}

	if err := s.services.CreateEnvironment(r.Context(), req.ID, req.Name, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("createEnvironment error", zap.Error(err))
		observability.ObserveDomainEvent("environment_created", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("environment_created", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) declareApplicationEnvironment(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req declareApplicationEnvironmentRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.ApplicationID == "" || req.EnvironmentID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id, applicationId and environmentId are required")
		return
	}

	if err := s.services.DeclareApplicationEnvironment(r.Context(), req.ID, req.ApplicationID, req.EnvironmentID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("declareApplicationEnvironment error", zap.Error(err))
		observability.ObserveDomainEvent("application_environment_declared", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_environment_declared", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) completeApplicationEnvironmentProvisioning(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !requireInternalAuth(w, r) {
		return
	}

	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req completeApplicationEnvironmentProvisioningRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.CompleteApplicationEnvironmentProvisioning(r.Context(), req.ID, "workflow-engine"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("completeApplicationEnvironmentProvisioning error", zap.Error(err))
		observability.ObserveDomainEvent("application_environment_provisioning_completed", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_environment_provisioning_completed", "success")
	// 202 para reflejar que viene de un workflow ya corrido.
	w.WriteHeader(http.StatusAccepted)
}
