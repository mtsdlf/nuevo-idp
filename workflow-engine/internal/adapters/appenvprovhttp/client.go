package appenvprovhttp

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

type appEnvPayload struct {
	ApplicationEnvironmentID string `json:"applicationEnvironmentId"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) ApplyBranchProtection(ctx context.Context, appEnvID string) error {
	return c.post(ctx, "/appenv/branch-protection", appEnvID)
}

func (c *Client) ProvisionSecrets(ctx context.Context, appEnvID string) error {
	return c.post(ctx, "/appenv/secrets", appEnvID)
}

func (c *Client) CreateSecretBindings(ctx context.Context, appEnvID string) error {
	return c.post(ctx, "/appenv/secret-bindings", appEnvID)
}

func (c *Client) VerifyGitOpsReconciliation(ctx context.Context, appEnvID string) error {
	return c.post(ctx, "/appenv/gitops-verify", appEnvID)
}

func (c *Client) post(ctx context.Context, path string, appEnvID string) error {
	ctx, span := tracing.StartSpan(ctx, "appenvprovhttp.post "+path)
	span.SetAttributes(
		attribute.String("appenv.id", appEnvID),
		attribute.String("http.path", path),
	)
	defer span.End()

	body, err := json.Marshal(appEnvPayload{ApplicationEnvironmentID: appEnvID})
	if err != nil {
		return fmt.Errorf("marshal appenv payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create appenv request for %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call appenv provider %s: %w", path, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp, path)
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

// Error representa un error devuelto por execution-workers para operaciones
// de aprovisionamiento de ApplicationEnvironment. Incluye el path para poder
// distinguir qué operación falló.
type Error struct {
	Status  int
	Path    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = fmt.Sprintf("execution-workers returned status %d for %s", e.Status, e.Path)
	}

	return fmt.Sprintf("execution-workers appenv provider error: status=%d path=%s message=%s", e.Status, e.Path, msg)
}

func newErrorFromResponse(resp *http.Response, path string) error {
	if resp == nil {
		return &Error{Status: 0, Path: path, Message: "nil response from execution-workers"}
	}

	body, _ := io.ReadAll(resp.Body)
	return &Error{
		Status:  resp.StatusCode,
		Path:    path,
		Message: string(body),
	}
}
