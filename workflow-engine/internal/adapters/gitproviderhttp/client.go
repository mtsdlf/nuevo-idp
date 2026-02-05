package gitproviderhttp

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

type createRepoPayload struct {
    Owner   string `json:"owner"`
    Name    string `json:"name"`
    Private bool   `json:"private"`
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{Timeout: 15 * time.Second},
    }
}

func (c *Client) CreateRepository(ctx context.Context, owner, name string, private bool) error {
    ctx, span := tracing.StartSpan(ctx, "gitproviderhttp.CreateRepository")
    span.SetAttributes(
        attribute.String("git.owner", owner),
        attribute.String("git.repo", name),
        attribute.Bool("git.private", private),
    )
    defer span.End()

    body, err := json.Marshal(createRepoPayload{
        Owner:   owner,
        Name:    name,
        Private: private,
    })
    if err != nil {
        return fmt.Errorf("marshal create repository payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/github/repos", bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("create github repos request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    setInternalAuthHeader(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("call github repos endpoint: %w", err)
    }
    defer func() {
        _ = resp.Body.Close()
    }()

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

// Error representa un error devuelto por execution-workers para operaciones
// del proveedor Git. Captura el status HTTP y el cuerpo en texto plano.
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

    return fmt.Sprintf("execution-workers git provider error: status=%d message=%s", e.Status, msg)
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
