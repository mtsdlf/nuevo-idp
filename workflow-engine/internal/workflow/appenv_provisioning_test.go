package workflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nuevo-idp/workflow-engine/internal/adapters/appenvprovhttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/controlplanehttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/gitproviderhttp"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestApplicationEnvironmentProvisioning_WorkflowHappyPath(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ApplicationEnvironmentProvisioning)
	env.RegisterActivity(MaterializeRepositories)
	env.RegisterActivity(ApplyBranchProtection)
	env.RegisterActivity(ProvisionSecrets)
	env.RegisterActivity(CreateSecretBindings)
	env.RegisterActivity(VerifyGitOpsReconciliation)
	env.RegisterActivity(FinalizeApplicationEnvironmentProvisioning)

	input := ApplicationEnvironmentProvisioningInput{ApplicationEnvironmentID: "ae-1"}
	env.ExecuteWorkflow(ApplicationEnvironmentProvisioning, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

type fakeGitProvider struct {
	called bool
	owner  string
	name   string
}

func (f *fakeGitProvider) CreateRepository(_ context.Context, owner, name string, _ bool) error {
	f.called = true
	f.owner = owner
	f.name = name
	return nil
}

func TestMaterializeRepositories_UsesGitProviderWhenConfigured(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeGitProvider{}
	SetGitProvider(fake)

	env.RegisterWorkflow(ApplicationEnvironmentProvisioning)
	env.RegisterActivity(MaterializeRepositories)
	env.RegisterActivity(ApplyBranchProtection)
	env.RegisterActivity(ProvisionSecrets)
	env.RegisterActivity(CreateSecretBindings)
	env.RegisterActivity(VerifyGitOpsReconciliation)
	env.RegisterActivity(FinalizeApplicationEnvironmentProvisioning)

	input := ApplicationEnvironmentProvisioningInput{ApplicationEnvironmentID: "ae-test"}
	env.ExecuteWorkflow(ApplicationEnvironmentProvisioning, input)

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !fake.called {
		t.Fatalf("expected GitProvider to be called")
	}
	if fake.owner != "platform" {
		t.Fatalf("expected owner 'platform', got %q", fake.owner)
	}
	if fake.name != "appenv-ae-test" {
		t.Fatalf("expected repo name 'appenv-ae-test', got %q", fake.name)
	}
}

func TestApplicationEnvironmentProvisioning_RequiresID(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ApplicationEnvironmentProvisioning)

	input := ApplicationEnvironmentProvisioningInput{}
	env.ExecuteWorkflow(ApplicationEnvironmentProvisioning, input)

	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected error for missing ApplicationEnvironmentID, got nil")
	}
}

type fakeAppEnvProvisioningProvider struct {
	branchProtectionCalls int
	secretsCalls          int
	bindingsCalls         int
	gitOpsCalls           int
}

func (f *fakeAppEnvProvisioningProvider) ApplyBranchProtection(_ context.Context, _ string) error {
	f.branchProtectionCalls++
	return nil
}

func (f *fakeAppEnvProvisioningProvider) ProvisionSecrets(_ context.Context, _ string) error {
	f.secretsCalls++
	return nil
}

func (f *fakeAppEnvProvisioningProvider) CreateSecretBindings(_ context.Context, _ string) error {
	f.bindingsCalls++
	return nil
}

func (f *fakeAppEnvProvisioningProvider) VerifyGitOpsReconciliation(_ context.Context, _ string) error {
	f.gitOpsCalls++
	return nil
}

func TestApplicationEnvironmentProvisioning_UsesAppEnvProvisioningProviderWhenConfigured(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	fake := &fakeAppEnvProvisioningProvider{}
	SetAppEnvProvisioningProvider(fake)

	env.RegisterWorkflow(ApplicationEnvironmentProvisioning)
	env.RegisterActivity(MaterializeRepositories)
	env.RegisterActivity(ApplyBranchProtection)
	env.RegisterActivity(ProvisionSecrets)
	env.RegisterActivity(CreateSecretBindings)
	env.RegisterActivity(VerifyGitOpsReconciliation)
	env.RegisterActivity(FinalizeApplicationEnvironmentProvisioning)

	input := ApplicationEnvironmentProvisioningInput{ApplicationEnvironmentID: "ae-prov"}
	env.ExecuteWorkflow(ApplicationEnvironmentProvisioning, input)

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fake.branchProtectionCalls != 1 {
		t.Fatalf("expected 1 branch protection call, got %d", fake.branchProtectionCalls)
	}
	if fake.secretsCalls != 1 {
		t.Fatalf("expected 1 secrets provisioning call, got %d", fake.secretsCalls)
	}
	if fake.bindingsCalls != 1 {
		t.Fatalf("expected 1 secret bindings call, got %d", fake.bindingsCalls)
	}
	if fake.gitOpsCalls != 1 {
		t.Fatalf("expected 1 gitops verification call, got %d", fake.gitOpsCalls)
	}
}

// Test de integración ligero que usa los adapters HTTP reales contra
// servidores httptest para verificar que el workflow invoque los
// endpoints esperados en execution-workers y control-plane-api.
func TestApplicationEnvironmentProvisioning_HTTPIntegration(t *testing.T) {
	execRequests := make(map[string]int)
	execServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		execRequests[r.URL.Path]++
		// Simulamos códigos típicos: 201 para repos, 202 para side-effects.
		status := http.StatusAccepted
		if r.URL.Path == "/github/repos" {
			status = http.StatusCreated
		}
		w.WriteHeader(status)
	}))
	defer execServer.Close()

	cpRequests := make(map[string]int)
	cpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cpRequests[r.URL.Path]++
		w.WriteHeader(http.StatusOK)
	}))
	defer cpServer.Close()

	SetGitProvider(gitproviderhttp.NewClient(execServer.URL))
	SetAppEnvProvisioningProvider(appenvprovhttp.NewClient(execServer.URL))
	SetControlPlaneClient(controlplanehttp.NewClient(cpServer.URL))

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(ApplicationEnvironmentProvisioning)
	env.RegisterActivity(MaterializeRepositories)
	env.RegisterActivity(ApplyBranchProtection)
	env.RegisterActivity(ProvisionSecrets)
	env.RegisterActivity(CreateSecretBindings)
	env.RegisterActivity(VerifyGitOpsReconciliation)
	env.RegisterActivity(FinalizeApplicationEnvironmentProvisioning)

	input := ApplicationEnvironmentProvisioningInput{ApplicationEnvironmentID: "ae-int"}
	env.ExecuteWorkflow(ApplicationEnvironmentProvisioning, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow not completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verificamos llamadas a execution-workers
	if got := execRequests["/github/repos"]; got != 1 {
		t.Fatalf("expected 1 call to /github/repos, got %d", got)
	}
	if got := execRequests["/appenv/branch-protection"]; got != 1 {
		t.Fatalf("expected 1 call to /appenv/branch-protection, got %d", got)
	}
	if got := execRequests["/appenv/secrets"]; got != 1 {
		t.Fatalf("expected 1 call to /appenv/secrets, got %d", got)
	}
	if got := execRequests["/appenv/secret-bindings"]; got != 1 {
		t.Fatalf("expected 1 call to /appenv/secret-bindings, got %d", got)
	}
	if got := execRequests["/appenv/gitops-verify"]; got != 1 {
		t.Fatalf("expected 1 call to /appenv/gitops-verify, got %d", got)
	}

	// Verificamos llamada a control-plane-api
	if got := cpRequests["/commands/application-environments/complete-provisioning"]; got != 1 {
		t.Fatalf("expected 1 call to /commands/application-environments/complete-provisioning, got %d", got)
	}
}

// failingControlPlaneClient simula un error de dominio del control-plane-api
// al completar el aprovisionamiento del ApplicationEnvironment. Devuelve un
// *controlplanehttp.Error con status 400 y un código estable que debe
// mapearse a un ApplicationError no-retriable con ese mismo Type.
type failingControlPlaneClient struct{}

func (f *failingControlPlaneClient) CompleteApplicationEnvironmentProvisioning(_ context.Context, _ string) error {
	return &controlplanehttp.Error{
		Status:  400,
		Code:    "application_environment_invalid_state_for_complete_provisioning",
		Message: "application environment can only complete provisioning from Provisioning state",
	}
}

func TestApplicationEnvironmentProvisioning_FailsOnControlPlaneDomainError(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	// Aseguramos un estado conocido de los puertos antes de ejecutar el workflow
	// para que este test no dependa del orden de ejecución de otros tests.
	SetGitProvider(nil)
	SetAppEnvProvisioningProvider(nil)
	SetControlPlaneClient(&failingControlPlaneClient{})

	env.RegisterWorkflow(ApplicationEnvironmentProvisioning)
	env.RegisterActivity(MaterializeRepositories)
	env.RegisterActivity(ApplyBranchProtection)
	env.RegisterActivity(ProvisionSecrets)
	env.RegisterActivity(CreateSecretBindings)
	env.RegisterActivity(VerifyGitOpsReconciliation)
	env.RegisterActivity(FinalizeApplicationEnvironmentProvisioning)

	input := ApplicationEnvironmentProvisioningInput{ApplicationEnvironmentID: "ae-invalid"}
	env.ExecuteWorkflow(ApplicationEnvironmentProvisioning, input)

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected error due to control-plane domain error, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected underlying ApplicationError, got %T", err)
	}
	if appErr.Type() != "application_environment_invalid_state_for_complete_provisioning" {
		t.Fatalf("expected error type 'application_environment_invalid_state_for_complete_provisioning', got %q", appErr.Type())
	}
	if !appErr.NonRetryable() {
		t.Fatalf("expected error to be non-retriable")
	}
}

func TestMapExecutionWorkersError_GitProvider4xx(t *testing.T) {
	err := mapExecutionWorkersError(&gitproviderhttp.Error{Status: 400, Message: "bad request"})
	if err == nil {
		t.Fatalf("expected mapped error, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected ApplicationError, got %T", err)
	}
	if appErr.Type() != "execution_workers_client_error" {
		t.Fatalf("expected type 'execution_workers_client_error', got %q", appErr.Type())
	}
	if !appErr.NonRetryable() {
		t.Fatalf("expected error to be non-retriable")
	}
}

func TestMapExecutionWorkersError_AppEnvProvider4xx(t *testing.T) {
	err := mapExecutionWorkersError(&appenvprovhttp.Error{Status: 409, Path: "/appenv/secrets", Message: "conflict"})
	if err == nil {
		t.Fatalf("expected mapped error, got nil")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected ApplicationError, got %T", err)
	}
	if appErr.Type() != "execution_workers_client_error" {
		t.Fatalf("expected type 'execution_workers_client_error', got %q", appErr.Type())
	}
	if !appErr.NonRetryable() {
		t.Fatalf("expected error to be non-retriable")
	}
}
