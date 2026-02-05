package httpapi

import (
	"net/http"

	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"go.uber.org/zap"
)

type createSecretRequest struct {
	ID          string `json:"id"`
	OwnerTeamID string `json:"ownerTeamId"`
	Purpose     string `json:"purpose"`
	Sensitivity string `json:"sensitivity"`
}

type declareSecretBindingRequest struct {
	ID         string `json:"id"`
	SecretID   string `json:"secretId"`
	TargetID   string `json:"targetId"`
	TargetType string `json:"targetType"`
}

type startSecretRotationRequest struct {
	ID string `json:"id"`
}

type completeSecretRotationRequest struct {
	ID string `json:"id"`
}

func (s *Server) createSecret(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req createSecretRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.OwnerTeamID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id and ownerTeamId are required")
		return
	}

	if err := s.services.CreateSecret(r.Context(), req.ID, req.OwnerTeamID, req.Purpose, req.Sensitivity, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("createSecret error", zap.Error(err))
		observability.ObserveDomainEvent("secret_created", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("secret_created", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) declareSecretBinding(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req declareSecretBindingRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.SecretID == "" || req.TargetID == "" || req.TargetType == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id, secretId, targetId and targetType are required")
		return
	}

	if err := s.services.DeclareSecretBinding(r.Context(), req.ID, req.SecretID, req.TargetID, req.TargetType, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("declareSecretBinding error", zap.Error(err))
		observability.ObserveDomainEvent("secret_binding_declared", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("secret_binding_declared", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) startSecretRotation(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req startSecretRotationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.StartSecretRotation(r.Context(), req.ID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("startSecretRotation error", zap.Error(err))
		observability.ObserveDomainEvent("secret_rotation_started", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("secret_rotation_started", "success")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) completeSecretRotation(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !requireInternalAuth(w, r) {
		return
	}

	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req completeSecretRotationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.CompleteSecretRotation(r.Context(), req.ID, "workflow-engine"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("completeSecretRotation error", zap.Error(err))
		observability.ObserveDomainEvent("secret_rotation_completed", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("secret_rotation_completed", "success")
	w.WriteHeader(http.StatusAccepted)
}
