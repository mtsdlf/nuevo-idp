package secretbindingshttp

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

type updateBindingsPayload struct {
	SecretID string `json:"secretId"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// UpdateSecretBindingsForSecret implementa el puerto SecretBindingsRotationPort
// del workflow SecretRotation. Envía una solicitud a execution-workers para que
// actualice todos los SecretBindings asociados a un Secret en sistemas externos.
func (c *Client) UpdateSecretBindingsForSecret(ctx context.Context, secretID string) error {
	ctx, span := tracing.StartSpan(ctx, "secretbindingshttp.UpdateSecretBindingsForSecret")
	span.SetAttributes(
		attribute.String("secret.id", secretID),
	)
	defer span.End()

	body, err := json.Marshal(updateBindingsPayload{SecretID: secretID})
	if err != nil {
		return fmt.Errorf("marshal secret bindings payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/secrets/bindings/update", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create secret bindings request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setInternalAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call secret bindings endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newErrorFromResponse(resp)
	}

	return nil
}

// Error representa un error devuelto por execution-workers al intentar
// actualizar SecretBindings para un Secret concreto.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = fmt.Sprintf("execution-workers returned status %d", e.Status)
	}

	return fmt.Sprintf("execution-workers secret bindings provider error: status=%d message=%s", e.Status, msg)
}

func newErrorFromResponse(resp *http.Response) error {
	if resp == nil {
		return &Error{Status: 0, Message: "nil response from execution-workers"}
	}

	body, _ := io.ReadAll(resp.Body)
	return &Error{
		Status:  resp.StatusCode,
		Message: string(body),
	}
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
