package httpapi

import (
	"net/http"

	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"go.uber.org/zap"
)

type createApplicationRequest struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	TeamID string `json:"teamId"`
}

type approveApplicationRequest struct {
	ID string `json:"id"`
}

type deprecateApplicationRequest struct {
	ID string `json:"id"`
}

type startApplicationOnboardingRequest struct {
	ID string `json:"id"`
}

type activateApplicationRequest struct {
	ID string `json:"id"`
}

//nolint:dupl
func (s *Server) createApplication(w http.ResponseWriter, r *http.Request) {
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req createApplicationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.Name == "" || req.TeamID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id, name and teamId are required")
		return
	}

	if err := s.services.CreateApplication(r.Context(), req.ID, req.Name, req.TeamID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("createApplication error", zap.Error(err))
		observability.ObserveDomainEvent("application_created", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_created", "success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) approveApplication(w http.ResponseWriter, r *http.Request) {
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req approveApplicationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.ApproveApplication(r.Context(), req.ID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("approveApplication error", zap.Error(err))
		observability.ObserveDomainEvent("application_approved", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_approved", "success")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) deprecateApplication(w http.ResponseWriter, r *http.Request) {
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req deprecateApplicationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.DeprecateApplication(r.Context(), req.ID, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("deprecateApplication error", zap.Error(err))
		observability.ObserveDomainEvent("application_deprecated", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_deprecated", "success")
	w.WriteHeader(http.StatusAccepted)
}

//nolint:dupl
func (s *Server) startApplicationOnboarding(w http.ResponseWriter, r *http.Request) {
	if !requireInternalAuth(w, r) {
		return
	}

	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req startApplicationOnboardingRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.StartApplicationOnboarding(r.Context(), req.ID, "workflow-engine"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("startApplicationOnboarding error", zap.Error(err))
		observability.ObserveDomainEvent("application_onboarding_started", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_onboarding_started", "success")
	w.WriteHeader(http.StatusAccepted)
}

//nolint:dupl
func (s *Server) activateApplication(w http.ResponseWriter, r *http.Request) {
	if !requireInternalAuth(w, r) {
		return
	}

	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req activateApplicationRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := s.services.ActivateApplication(r.Context(), req.ID, "workflow-engine"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("activateApplication error", zap.Error(err))
		observability.ObserveDomainEvent("application_activated", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("application_activated", "success")
	w.WriteHeader(http.StatusAccepted)
}
