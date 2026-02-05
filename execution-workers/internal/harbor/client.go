package harbor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nuevo-idp/platform/config"
	"github.com/nuevo-idp/platform/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Client modela un cliente muy simple para Harbor (o para un endpoint
// compatible). Para el MVP sólo encapsula la rotación de un token de
// robot account global a través de una llamada HTTP autenticada.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClientFromEnv construye un cliente Harbor leyendo configuración
// desde variables de entorno. Si faltan valores, devuelve un error para
// que el caller pueda decidir si falla o degrada a no-op.
//
// Nota: HARBOR_URL se interpreta como el endpoint base al que se hará
// una petición HTTP para rotar el token del robot (por ejemplo, un
// pequeño servicio interno que a su vez habla con Harbor).
func NewClientFromEnv() (*Client, error) {
	baseURL, ok := config.Require("HARBOR_URL")
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("HARBOR_URL not configured")
	}

	username, ok := config.Require("HARBOR_ROBOT_USERNAME")
	if !ok || username == "" {
		return nil, fmt.Errorf("HARBOR_ROBOT_USERNAME not configured")
	}

	password, ok := config.Require("HARBOR_ROBOT_PASSWORD")
	if !ok || password == "" {
		return nil, fmt.Errorf("HARBOR_ROBOT_PASSWORD not configured")
	}

	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// RotateRobotToken es una operación de alto nivel que, en una integración
// real, debería llamar a la API de Harbor para generar un nuevo token para
// el robot account configurado y devolverlo para su uso inmediato.
//
// Para este MVP dejamos el cuerpo como un seam explícito: el caller sabe
// si la operación fue "exitosa" o no, pero los detalles de la API concreta
// se implementarán más adelante.
func (c *Client) RotateRobotToken(ctx context.Context) (string, error) {
	ctx, span := tracing.StartSpan(ctx, "execution-workers.harbor.RotateRobotToken")
	span.SetAttributes(
		attribute.String("harbor.base_url", c.baseURL),
		attribute.String("harbor.robot_user", c.username),
	)
	defer span.End()

	// Para el MVP, interpretamos baseURL como el endpoint HTTP que sabe
	// cómo rotar el token del robot account configurado. No asumimos la
	// forma exacta de la API de Harbor; puede ser un pequeño façade
	// interno que hable con Harbor.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("creating harbor request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling harbor endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("harbor endpoint returned status %d", resp.StatusCode)
	}

	// Esperamos un cuerpo JSON mínimo con el nuevo token, por ejemplo:
	// { "token": "<nuevo-token>" }
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decoding harbor response: %w", err)
	}
	if body.Token == "" {
		return "", fmt.Errorf("harbor response missing token field")
	}

	return body.Token, nil
}
