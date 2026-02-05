package smoke

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func waitForHealthy(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		resp, err := http.Get(url) // #nosec G107 - URL controlado en tests
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("service at %s not healthy in time: %v", url, err)
			}
			_ = resp.Body.Close()
			t.Fatalf("service at %s returned status %d", url, resp.StatusCode)
		}
		time.Sleep(2 * time.Second)
	}
}

func TestStackSmoke_ApplicationLifecycle(t *testing.T) {
	// URLs dentro de la red de docker-compose
	controlPlaneURL := "http://control-plane-api:8080"
	workflowURL := "http://workflow-engine:8081"
	executionWorkersURL := "http://execution-workers:8082"

	waitForHealthy(t, controlPlaneURL+"/healthz", 60*time.Second)
	waitForHealthy(t, workflowURL+"/healthz", 60*time.Second)
	waitForHealthy(t, executionWorkersURL+"/healthz", 60*time.Second)

	client := &http.Client{Timeout: 15 * time.Second}

	// 1) Crear team
	teamBody, _ := json.Marshal(map[string]string{
		"id":   "team-smoke",
		"name": "Smoke Team",
	})
	resp, err := client.Post(controlPlaneURL+"/commands/teams", "application/json", bytes.NewReader(teamBody))
	if err != nil {
		t.Fatalf("error creating team: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 201 or 409 for create team, got %d", resp.StatusCode)
	}

	// 2) Crear aplicación ligada al team
	appBody, _ := json.Marshal(map[string]string{
		"id":     "app-smoke",
		"name":   "Smoke App",
		"teamId": "team-smoke",
	})
	resp, err = client.Post(controlPlaneURL+"/commands/applications", "application/json", bytes.NewReader(appBody))
	if err != nil {
		t.Fatalf("error creating application: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 201 or 409 for create application, got %d", resp.StatusCode)
	}

	// 3) Aprobar aplicación (idempotente para el smoke)
	approveBody, _ := json.Marshal(map[string]string{
		"id": "app-smoke",
	})
	resp, err = client.Post(controlPlaneURL+"/commands/applications/approve", "application/json", bytes.NewReader(approveBody))
	if err != nil {
		t.Fatalf("error approving application: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 202 or 400 for approve application, got %d", resp.StatusCode)
	}

	// 4) Verificar que el query endpoint responde para la app
	resp, err = client.Get(controlPlaneURL + "/queries/applications?id=app-smoke")
	if err != nil {
		t.Fatalf("error querying application: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 for get application, got %d", resp.StatusCode)
	}

	// Simple sanity check de que INTERNAL_AUTH_TOKEN está configurado en este entorno
	if os.Getenv("INTERNAL_AUTH_TOKEN") == "" {
		t.Fatalf("INTERNAL_AUTH_TOKEN should be set in smoke-tests environment")
	}
}
