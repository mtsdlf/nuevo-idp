package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nuevo-idp/workflow-engine/internal/adapters/controlplanehttp"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestApplicationOnboarding_HappyPath(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeApplicationOnboardingPort{}
	SetApplicationOnboardingPort(fake)

	env.RegisterWorkflow(ApplicationOnboarding)
	env.RegisterActivity(CreateCodeRepositoryForApplication)
	env.RegisterActivity(CreateDeploymentRepositoryForApplication)
	env.RegisterActivity(CreateGitOpsIntegrationForApplication)
	env.RegisterActivity(DeclareApplicationEnvironmentsForApplication)
	env.RegisterActivity(TransitionApplicationToOnboarding)

	// Simulamos que el sistema externo envía el evento SecurityScanPassed
	// antes de que venza el timeout de 15 minutos.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(securityScanPassedSignalName, nil)
	}, time.Minute)

	input := ApplicationOnboardingInput{ApplicationID: "app-1"}
	env.ExecuteWorkflow(ApplicationOnboarding, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if fake.codeRepoCalls != 1 {
		t.Fatalf("expected 1 CodeRepository declaration, got %d", fake.codeRepoCalls)
	}
	if fake.deploymentRepoCalls != 1 {
		t.Fatalf("expected 1 DeploymentRepository declaration, got %d", fake.deploymentRepoCalls)
	}
	if fake.gitOpsIntegrationCalls != 1 {
		t.Fatalf("expected 1 GitOpsIntegration declaration, got %d", fake.gitOpsIntegrationCalls)
	}
	if fake.appEnvDeclarationCalls != 1 {
		t.Fatalf("expected 1 ApplicationEnvironments declaration, got %d", fake.appEnvDeclarationCalls)
	}
	if fake.applicationOnboardingCalls != 1 {
		t.Fatalf("expected 1 Application onboarding state transition, got %d", fake.applicationOnboardingCalls)
	}
}

// Test de integración ligero: usa el client HTTP real de control-planehttp
// contra un httptest.Server que simula el control-plane-api. Verifica que el
// workflow invoque los endpoints HTTP esperados.
func TestApplicationOnboarding_HTTPIntegration(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	requests := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := controlplanehttp.NewClient(server.URL)
	SetApplicationOnboardingPort(client)

	env.RegisterWorkflow(ApplicationOnboarding)
	env.RegisterActivity(CreateCodeRepositoryForApplication)
	env.RegisterActivity(CreateDeploymentRepositoryForApplication)
	env.RegisterActivity(CreateGitOpsIntegrationForApplication)
	env.RegisterActivity(DeclareApplicationEnvironmentsForApplication)
	env.RegisterActivity(TransitionApplicationToOnboarding)

	// Enviamos la señal SecurityScanPassed para que el workflow avance.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(securityScanPassedSignalName, nil)
	}, time.Minute)

	input := ApplicationOnboardingInput{ApplicationID: "app-int"}
	env.ExecuteWorkflow(ApplicationOnboarding, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := requests["/commands/code-repositories"]; got != 1 {
		t.Fatalf("expected 1 call to /commands/code-repositories, got %d", got)
	}
	if got := requests["/commands/deployment-repositories"]; got != 1 {
		t.Fatalf("expected 1 call to /commands/deployment-repositories, got %d", got)
	}
	if got := requests["/commands/gitops-integrations"]; got != 1 {
		t.Fatalf("expected 1 call to /commands/gitops-integrations, got %d", got)
	}
	if got := requests["/commands/application-environments"]; got != 2 { // env-dev y env-prod
		t.Fatalf("expected 2 calls to /commands/application-environments, got %d", got)
	}
	if got := requests["/commands/applications/start-onboarding"]; got != 1 {
		t.Fatalf("expected 1 call to /commands/applications/start-onboarding, got %d", got)
	}
}

func TestApplicationOnboarding_RequiresID(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ApplicationOnboarding)

	input := ApplicationOnboardingInput{}
	env.ExecuteWorkflow(ApplicationOnboarding, input)

	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected error for missing ApplicationID, got nil")
	}
}

func TestApplicationOnboarding_FailsOnSecurityScanTimeout(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeApplicationOnboardingPort{}
	SetApplicationOnboardingPort(fake)

	env.RegisterWorkflow(ApplicationOnboarding)
	env.RegisterActivity(CreateCodeRepositoryForApplication)
	env.RegisterActivity(CreateDeploymentRepositoryForApplication)
	env.RegisterActivity(CreateGitOpsIntegrationForApplication)
	env.RegisterActivity(DeclareApplicationEnvironmentsForApplication)
	env.RegisterActivity(TransitionApplicationToOnboarding)

	input := ApplicationOnboardingInput{ApplicationID: "app-timeout"}
	env.ExecuteWorkflow(ApplicationOnboarding, input)

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected error due to security scan timeout, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected underlying ApplicationError, got %T", err)
	}
	if appErr.Type() != "security_scan_timeout" {
		t.Fatalf("expected error type 'security_scan_timeout', got %q", appErr.Type())
	}
}

// Simula un error de dominio del control-plane-api al marcar la Application
// como Onboarding. El puerto devuelve un *controlplanehttp.Error con status 400
// y un código estable, que debe mapearse a un ApplicationError no-retriable
// con ese mismo Type.
type failingOnboardingPort struct{}

func (f *failingOnboardingPort) DeclareCodeRepository(_ context.Context, _ string) error {
	return nil
}

func (f *failingOnboardingPort) DeclareDeploymentRepository(_ context.Context, _ string) error {
	return nil
}

func (f *failingOnboardingPort) DeclareGitOpsIntegration(_ context.Context, _ string) error {
	return nil
}

func (f *failingOnboardingPort) DeclareApplicationEnvironments(_ context.Context, _ string) error {
	return nil
}

func (f *failingOnboardingPort) MarkApplicationOnboarding(_ context.Context, _ string) error {
	return &controlplanehttp.Error{Status: 400, Code: "application_invalid_state_for_onboarding", Message: "application can only start onboarding from Approved state"}
}

func (f *failingOnboardingPort) ActivateApplication(_ context.Context, _ string) error {
	return nil
}

func TestApplicationOnboarding_FailsOnControlPlaneDomainError(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	SetApplicationOnboardingPort(&failingOnboardingPort{})

	env.RegisterWorkflow(ApplicationOnboarding)
	env.RegisterActivity(CreateCodeRepositoryForApplication)
	env.RegisterActivity(CreateDeploymentRepositoryForApplication)
	env.RegisterActivity(CreateGitOpsIntegrationForApplication)
	env.RegisterActivity(DeclareApplicationEnvironmentsForApplication)
	env.RegisterActivity(TransitionApplicationToOnboarding)

	// Hacemos que el workflow avance hasta la actividad
	// TransitionApplicationToOnboarding enviando la señal de security scan.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(securityScanPassedSignalName, nil)
	}, time.Minute)

	input := ApplicationOnboardingInput{ApplicationID: "app-invalid"}
	env.ExecuteWorkflow(ApplicationOnboarding, input)

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected error due to control-plane domain error, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected underlying ApplicationError, got %T", err)
	}
	if appErr.Type() != "application_invalid_state_for_onboarding" {
		t.Fatalf("expected error type 'application_invalid_state_for_onboarding', got %q", appErr.Type())
	}
	if !appErr.NonRetryable() {
		t.Fatalf("expected error to be non-retriable")
	}
}

type fakeApplicationOnboardingPort struct {
	codeRepoCalls             int
	deploymentRepoCalls       int
	gitOpsIntegrationCalls    int
	appEnvDeclarationCalls    int
	applicationOnboardingCalls int
	applicationActivationCalls int
}

func (f *fakeApplicationOnboardingPort) DeclareCodeRepository(_ context.Context, _ string) error {
	f.codeRepoCalls++
	return nil
}

func (f *fakeApplicationOnboardingPort) DeclareDeploymentRepository(_ context.Context, _ string) error {
	f.deploymentRepoCalls++
	return nil
}

func (f *fakeApplicationOnboardingPort) DeclareGitOpsIntegration(_ context.Context, _ string) error {
	f.gitOpsIntegrationCalls++
	return nil
}

func (f *fakeApplicationOnboardingPort) DeclareApplicationEnvironments(_ context.Context, _ string) error {
	f.appEnvDeclarationCalls++
	return nil
}

func (f *fakeApplicationOnboardingPort) MarkApplicationOnboarding(_ context.Context, _ string) error {
	f.applicationOnboardingCalls++
	return nil
}

func (f *fakeApplicationOnboardingPort) ActivateApplication(_ context.Context, _ string) error {
	f.applicationActivationCalls++
	return nil
}
// ApplicationActivation workflows

