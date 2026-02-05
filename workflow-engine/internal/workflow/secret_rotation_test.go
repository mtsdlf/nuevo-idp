package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nuevo-idp/workflow-engine/internal/adapters/controlplanehttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/secretbindingshttp"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type fakeSecretRotationPort struct {
	completeCalls int
}

func (f *fakeSecretRotationPort) CompleteSecretRotation(_ context.Context, _ string) error {
	f.completeCalls++
	return nil
}

func TestSecretRotation_HappyPath(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeSecretRotationPort{}
	SetSecretRotationPort(fake)

	env.RegisterWorkflow(SecretRotation)
	env.RegisterActivity(PerformSecretRotation)
	env.RegisterActivity(UpdateSecretBindingsForSecret)
	env.RegisterActivity(CompleteSecretRotationActivity)

	// Simulamos evento externo RotationValidatedExternally antes del timeout.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(rotationValidatedSignalName, nil)
	}, time.Minute)

	input := SecretRotationInput{SecretID: "sec-1"}
	env.ExecuteWorkflow(SecretRotation, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if fake.completeCalls != 1 {
		t.Fatalf("expected 1 CompleteSecretRotation call, got %d", fake.completeCalls)
	}
}

type failingSecretRotationPort struct{}

func (f *failingSecretRotationPort) CompleteSecretRotation(_ context.Context, _ string) error {
	return &controlplanehttp.Error{Status: 400, Code: "secret_invalid_state_for_complete_rotation", Message: "secret can only complete rotation from Rotating state"}
}

func TestSecretRotation_RequiresID(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(SecretRotation)

	input := SecretRotationInput{}
	env.ExecuteWorkflow(SecretRotation, input)

	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected error for missing SecretID, got nil")
	}
}

func TestSecretRotation_FailsOnTimeout(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeSecretRotationPort{}
	SetSecretRotationPort(fake)

	env.RegisterWorkflow(SecretRotation)
	env.RegisterActivity(PerformSecretRotation)
	env.RegisterActivity(UpdateSecretBindingsForSecret)
	env.RegisterActivity(CompleteSecretRotationActivity)

	input := SecretRotationInput{SecretID: "sec-timeout"}
	env.ExecuteWorkflow(SecretRotation, input)

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected error due to secret rotation timeout, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected underlying ApplicationError, got %T", err)
	}
	if appErr.Type() != "secret_rotation_timeout" {
		t.Fatalf("expected error type 'secret_rotation_timeout', got %q", appErr.Type())
	}
}

func TestSecretRotation_FailsOnControlPlaneDomainError(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	failing := &failingSecretRotationPort{}
	SetSecretRotationPort(failing)

	env.RegisterWorkflow(SecretRotation)
	env.RegisterActivity(PerformSecretRotation)
	env.RegisterActivity(UpdateSecretBindingsForSecret)
	env.RegisterActivity(CompleteSecretRotationActivity)

	// Enviamos la señal de validación para que el workflow llegue a la
	// actividad CompleteSecretRotationActivity y reciba el error del puerto.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(rotationValidatedSignalName, nil)
	}, time.Minute)

	input := SecretRotationInput{SecretID: "sec-invalid"}
	env.ExecuteWorkflow(SecretRotation, input)

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected error due to control-plane domain error, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected underlying ApplicationError, got %T", err)
	}
	if appErr.Type() != "secret_invalid_state_for_complete_rotation" {
		t.Fatalf("expected error type 'secret_invalid_state_for_complete_rotation', got %q", appErr.Type())
	}
	if !appErr.NonRetryable() {
		t.Fatalf("expected error to be non-retriable")
	}
}

// Test de integración ligero que usa los clients HTTP reales de control-plane
// y secretbindings contra httptest.Server para verificar el contrato HTTP del
// workflow SecretRotation.
func TestSecretRotation_HTTPIntegration(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	controlPlaneRequests := make(map[string]int)
	controlPlaneServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		controlPlaneRequests[r.URL.Path]++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer controlPlaneServer.Close()

	secretBindingsRequests := make(map[string]int)
	secretBindingsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secretBindingsRequests[r.URL.Path]++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer secretBindingsServer.Close()

	secretRotationClient := controlplanehttp.NewClient(controlPlaneServer.URL)
	bindingsClient := secretbindingshttp.NewClient(secretBindingsServer.URL)

	SetSecretRotationPort(secretRotationClient)
	SetSecretBindingsRotationPort(bindingsClient)

	env.RegisterWorkflow(SecretRotation)
	env.RegisterActivity(PerformSecretRotation)
	env.RegisterActivity(UpdateSecretBindingsForSecret)
	env.RegisterActivity(CompleteSecretRotationActivity)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(rotationValidatedSignalName, nil)
	}, time.Minute)

	input := SecretRotationInput{SecretID: "sec-int"}
	env.ExecuteWorkflow(SecretRotation, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := secretBindingsRequests["/secrets/bindings/update"]; got != 1 {
		t.Fatalf("expected 1 call to /secrets/bindings/update, got %d", got)
	}
	if got := controlPlaneRequests["/commands/secrets/complete-rotation"]; got != 1 {
		t.Fatalf("expected 1 call to /commands/secrets/complete-rotation, got %d", got)
	}
}
