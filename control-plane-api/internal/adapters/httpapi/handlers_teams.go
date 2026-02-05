package httpapi

import (
	"net/http"

	"github.com/nuevo-idp/platform/httpx"
	"github.com/nuevo-idp/platform/observability"
	"go.uber.org/zap"
)

type createTeamRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Server) createTeam(w http.ResponseWriter, r *http.Request) { //nolint:dupl // handler HTTP pequeño y simétrico con otros; duplicación es intencional por claridad
	if !httpx.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req createTeamRequest
	if !httpx.DecodeJSON(w, r, &req, "invalid json") {
		return
	}

	if req.ID == "" || req.Name == "" {
		httpx.WriteText(w, http.StatusBadRequest, "id and name are required")
		return
	}

	if err := s.services.CreateTeam(r.Context(), req.ID, req.Name, "api"); err != nil {
		logger := observability.LoggerWithTrace(r.Context(), s.logger)
		logger.Error("createTeam error", zap.Error(err))
		observability.ObserveDomainEvent("team_created", "error")
		writeDomainError(w, err)
		return
	}

	observability.ObserveDomainEvent("team_created", "success")
	w.WriteHeader(http.StatusCreated)
}
