package controlplanehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nuevo-idp/platform/config"
	"github.com/nuevo-idp/platform/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type completeAppEnvProvisioningRequest struct {
	ID string `json:"id"`
}

type declareCodeRepositoryRequest struct {
	ID            string `json:"id"`
	ApplicationID string `json:"applicationId"`
}

type declareDeploymentRepositoryRequest struct {
	ID              string `json:"id"`
	ApplicationID   string `json:"applicationId"`
	DeploymentModel string `json:"deploymentModel"`
}

type declareGitOpsIntegrationRequest struct {
	ID               string `json:"id"`
	ApplicationID    string `json:"applicationId"`
	DeploymentRepoID string `json:"deploymentRepositoryId"`
}

type declareApplicationEnvironmentRequest struct {
	ID            string `json:"id"`
	ApplicationID string `json:"applicationId"`
	EnvironmentID string `json:"environmentId"`
}

type startApplicationOnboardingRequest struct {
	ID string `json:"id"`
}

type activateApplicationRequest struct {
	ID string `json:"id"`
}

type completeSecretRotationRequest struct {
	ID string `json:"id"`
}

// Error representa un error devuelto por control-plane-api. Captura el
// status HTTP junto con el código y mensaje estructurados del cuerpo JSON.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	code := e.Code
	if code == "" {
		code = "unknown_error"
	}

	msg := e.Message
	if strings.TrimSpace(msg) == "" {
		msg = fmt.Sprintf("control-plane-api returned status %d", e.Status)
	}

	return fmt.Sprintf("control-plane-api error: status=%d code=%s message=%s", e.Status, code, msg)
}

func newErrorFromResponse(resp *http.Response) error {
	if resp == nil {
		return &Error{Status: 0, Code: "unknown_error", Message: "nil response from control-plane-api"}
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(resp.Body)

	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &payload) // si falla, usamos el raw body abajo

	msg := payload.Message
	if strings.TrimSpace(msg) == "" {
		msg = strings.TrimSpace(string(body))
	}

	return &Error{
		Status:  resp.StatusCode,
		Code:    payload.Code,
		Message: msg,
	}
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) CompleteApplicationEnvironmentProvisioning(ctx context.Context, appEnvID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.CompleteApplicationEnvironmentProvisioning")
	span.SetAttributes(attribute.String("appenv.id", appEnvID))
	defer span.End()

	body, err := json.Marshal(completeAppEnvProvisioningRequest{ID: appEnvID})
	if err != nil {
		return fmt.Errorf("marshal complete appenv provisioning request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/application-environments/complete-provisioning", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create complete appenv provisioning request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call complete appenv provisioning endpoint: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

// DeclareCodeRepository implementa el puerto de onboarding para crear un CodeRepository
// asociado a una Application. Usa un ID derivado de la aplicación para mantener
// una convención simple.
func (c *Client) DeclareCodeRepository(ctx context.Context, applicationID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.DeclareCodeRepository")
	span.SetAttributes(attribute.String("application.id", applicationID))
	defer span.End()

	reqBody := declareCodeRepositoryRequest{
		ID:            "code-" + applicationID,
		ApplicationID: applicationID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal declare code repository request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/code-repositories", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create declare code repository request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call declare code repository endpoint: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

// DeclareDeploymentRepository crea un DeploymentRepository asociado a la Application.
// Para simplificar, siempre usa el modelo GitOpsPerApplication.
func (c *Client) DeclareDeploymentRepository(ctx context.Context, applicationID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.DeclareDeploymentRepository")
	defer span.End()

	reqBody := declareDeploymentRepositoryRequest{
		ID:              "dep-" + applicationID,
		ApplicationID:   applicationID,
		DeploymentModel: "GitOpsPerApplication",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal declare deployment repository request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/deployment-repositories", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create declare deployment repository request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call declare deployment repository endpoint: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

// DeclareGitOpsIntegration crea la integración GitOps usando el deployment repo
// derivado de la aplicación.
func (c *Client) DeclareGitOpsIntegration(ctx context.Context, applicationID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.DeclareGitOpsIntegration")
	defer span.End()

	depID := "dep-" + applicationID
	reqBody := declareGitOpsIntegrationRequest{
		ID:               "gi-" + applicationID,
		ApplicationID:    applicationID,
		DeploymentRepoID: depID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal declare gitops integration request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/gitops-integrations", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create declare gitops integration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call declare gitops integration endpoint: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

// DeclareApplicationEnvironments declara los ApplicationEnvironment para un conjunto
// fijo de environments esperados durante onboarding.
func (c *Client) DeclareApplicationEnvironments(ctx context.Context, applicationID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.DeclareApplicationEnvironments")
	defer span.End()

	envs := []string{"env-dev", "env-prod"}
	for _, envID := range envs {
		reqBody := declareApplicationEnvironmentRequest{
			ID:            applicationID + "-" + envID,
			ApplicationID: applicationID,
			EnvironmentID: envID,
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal declare application environment request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/application-environments", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create declare application environment request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		setInternalAuthHeader(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("call declare application environment endpoint: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Mantener el contexto del envID en el mensaje de error.
			if apiErr, ok := newErrorFromResponse(resp).(*Error); ok {
				apiErr.Message = fmt.Sprintf("%s (environment=%s)", apiErr.Message, envID)
				return apiErr
			}
			return fmt.Errorf("control-plane-api returned status %d for environment %s", resp.StatusCode, envID)
		}
	}

	return nil
}

//nolint:misspell
// MarkApplicationOnboarding llama al comando HTTP que mueve la Application a
// estado Onboarding.
func (c *Client) MarkApplicationOnboarding(ctx context.Context, applicationID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.MarkApplicationOnboarding")
	defer span.End()

	body, err := json.Marshal(startApplicationOnboardingRequest{ID: applicationID})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/applications/start-onboarding", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

//nolint:misspell
// ActivateApplication llama al comando HTTP que mueve la Application a
// estado Active.
func (c *Client) ActivateApplication(ctx context.Context, applicationID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.ActivateApplication")
	defer span.End()

	body, err := json.Marshal(activateApplicationRequest{ID: applicationID})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/applications/activate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

//nolint:misspell
// CompleteSecretRotation llama al comando HTTP que mueve un Secret de Rotating
// a Active.
func (c *Client) CompleteSecretRotation(ctx context.Context, secretID string) error {
	ctx, span := tracing.StartSpan(ctx, "controlplanehttp.CompleteSecretRotation")
	defer span.End()

	body, err := json.Marshal(completeSecretRotationRequest{ID: secretID})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/commands/secrets/complete-rotation", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

const internalAuthHeader = "X-Internal-Token"

func setInternalAuthHeader(req *http.Request) {
	if req == nil {
		return
	}
	if token := config.Get("INTERNAL_AUTH_TOKEN", ""); token != "" {
		req.Header.Set(internalAuthHeader, token)
	}
}
