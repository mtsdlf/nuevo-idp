package workflow

import (
	"context"
	"time"

	"errors"

	"github.com/nuevo-idp/platform/observability"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/appenvprovhttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/controlplanehttp"
	"github.com/nuevo-idp/workflow-engine/internal/adapters/gitproviderhttp"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const ApplicationEnvironmentProvisioningTaskQueue = "appenv-provisioning-task-queue"

// ApplicationEnvironmentProvisioningInput is a minimal representation of the intent
// "ApplicationEnvironmentProvisioning" from ejemplo_estado_Deseado.json.
type ApplicationEnvironmentProvisioningInput struct {
	ApplicationEnvironmentID string
}

// ControlPlaneAPI is a narrow port used by activities to notify the
// control-plane-api about provisioning completion.
type ControlPlaneAPI interface {
	CompleteApplicationEnvironmentProvisioning(ctx context.Context, appEnvID string) error
}

var controlPlaneClient ControlPlaneAPI

// SetControlPlaneClient allows main and tests to inject a concrete implementation
// (HTTP client in production, fakes in tests). If not set, the finalize activity
// will behave as a no-op.
func SetControlPlaneClient(c ControlPlaneAPI) {
	controlPlaneClient = c
}

// GitProvider is a narrow port to execution-workers for repository operations.
type GitProvider interface {
	CreateRepository(ctx context.Context, owner, name string, private bool) error
}

var gitProvider GitProvider

// SetGitProvider allows main/tests to inject an implementation backed by execution-workers
// or fakes. If not set, MaterializeRepositories will be a no-op.
func SetGitProvider(p GitProvider) {
	gitProvider = p
}

// AppEnvProvisioningProvider is a narrow port to execution-workers for
// non-Git repository side effects during ApplicationEnvironment provisioning.
// It encapsulates branch protection, secrets provisioning, secret bindings
// and GitOps reconciliation verification.
type AppEnvProvisioningProvider interface {
	ApplyBranchProtection(ctx context.Context, appEnvID string) error
	ProvisionSecrets(ctx context.Context, appEnvID string) error
	CreateSecretBindings(ctx context.Context, appEnvID string) error
	VerifyGitOpsReconciliation(ctx context.Context, appEnvID string) error
}

var appEnvProvisioningProvider AppEnvProvisioningProvider

// SetAppEnvProvisioningProvider allows main/tests to inject an implementation
// backed by execution-workers or fakes. If not set, the corresponding
// activities will behave as no-ops.
func SetAppEnvProvisioningProvider(p AppEnvProvisioningProvider) {
	appEnvProvisioningProvider = p
}

func ApplicationEnvironmentProvisioning(ctx workflow.Context, input ApplicationEnvironmentProvisioningInput) (err error) {
	start := workflow.Now(ctx)
	result := "success"

	info := workflow.GetInfo(ctx)
	if info.Attempt > 1 {
		observability.ObserveWorkflowRetries("ApplicationEnvironmentProvisioning", 1)
	}

	defer func() {
		if err != nil {
			result = "error"
		}
		duration := workflow.Now(ctx).Sub(start).Seconds()
		observability.ObserveWorkflowDuration("ApplicationEnvironmentProvisioning", result, duration)
	}()

	if input.ApplicationEnvironmentID == "" {
		//nolint:wrapcheck // devolvemos directamente ApplicationError de Temporal para que el caller pueda inspeccionar Type
		return temporal.NewNonRetryableApplicationError("ApplicationEnvironmentID is required", "bad_input", nil)
	}

	// Each step is an activity so that later we can map them to
	// concrete calls to execution-workers (GitHub, secrets, etc.).
	opts := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Minute,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, opts)

	steps := []interface{}{
		MaterializeRepositories,
		ApplyBranchProtection,
		ProvisionSecrets,
		CreateSecretBindings,
		VerifyGitOpsReconciliation,
		FinalizeApplicationEnvironmentProvisioning,
	}

	for _, step := range steps {
		future := workflow.ExecuteActivity(ctx, step, input.ApplicationEnvironmentID)
		if err := future.Get(ctx, nil); err != nil {
			observability.ObserveDomainEvent("workflow_appenv_provisioning_failed", "error")
			//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
			return err
		}
	}

	observability.ObserveDomainEvent("workflow_appenv_provisioning_completed", "success")
	return nil
}

// Activities below are intentionally generic; real side-effects vivirán en execution-workers.

func MaterializeRepositories(ctx context.Context, appEnvID string) error {
	logger := activity.GetLogger(ctx)
	if gitProvider == nil {
		logger.Info("No GitProvider configured; skipping repository materialization", "appEnvID", appEnvID)
		return nil
	}

	// Por ahora usamos una convención simple de nombre; más adelante
	// podremos derivarlo de más contexto del ApplicationEnvironment.
	repoName := "appenv-" + appEnvID
	owner := "platform" // TODO: parametrizar por equipo/organización

	logger.Info("Creating Git repository via GitProvider", "owner", owner, "name", repoName, "appEnvID", appEnvID)
	err := gitProvider.CreateRepository(ctx, owner, repoName, true)
	logExecutionWorkersErrorIfAny(logger, err, "CreateRepository", appEnvID)
	return mapExecutionWorkersError(err)
}

func ApplyBranchProtection(ctx context.Context, appEnvID string) error {
	logger := activity.GetLogger(ctx)
	if appEnvProvisioningProvider == nil {
		logger.Info("No AppEnvProvisioningProvider configured; skipping branch protection", "appEnvID", appEnvID)
		return nil
	}

	logger.Info("Applying branch protection via provider", "appEnvID", appEnvID)
	err := appEnvProvisioningProvider.ApplyBranchProtection(ctx, appEnvID)
	logExecutionWorkersErrorIfAny(logger, err, "ApplyBranchProtection", appEnvID)
	return mapExecutionWorkersError(err)
}

func ProvisionSecrets(ctx context.Context, appEnvID string) error {
	logger := activity.GetLogger(ctx)
	if appEnvProvisioningProvider == nil {
		logger.Info("No AppEnvProvisioningProvider configured; skipping secrets provisioning", "appEnvID", appEnvID)
		return nil
	}

	logger.Info("Provisioning secrets via provider", "appEnvID", appEnvID)
	err := appEnvProvisioningProvider.ProvisionSecrets(ctx, appEnvID)
	logExecutionWorkersErrorIfAny(logger, err, "ProvisionSecrets", appEnvID)
	return mapExecutionWorkersError(err)
}

func CreateSecretBindings(ctx context.Context, appEnvID string) error {
	logger := activity.GetLogger(ctx)
	if appEnvProvisioningProvider == nil {
		logger.Info("No AppEnvProvisioningProvider configured; skipping secret bindings creation", "appEnvID", appEnvID)
		return nil
	}

	logger.Info("Creating secret bindings via provider", "appEnvID", appEnvID)
	err := appEnvProvisioningProvider.CreateSecretBindings(ctx, appEnvID)
	logExecutionWorkersErrorIfAny(logger, err, "CreateSecretBindings", appEnvID)
	return mapExecutionWorkersError(err)
}

func VerifyGitOpsReconciliation(ctx context.Context, appEnvID string) error {
	logger := activity.GetLogger(ctx)
	if appEnvProvisioningProvider == nil {
		logger.Info("No AppEnvProvisioningProvider configured; skipping GitOps reconciliation verification", "appEnvID", appEnvID)
		return nil
	}

	logger.Info("Verifying GitOps reconciliation via provider", "appEnvID", appEnvID)
	err := appEnvProvisioningProvider.VerifyGitOpsReconciliation(ctx, appEnvID)
	logExecutionWorkersErrorIfAny(logger, err, "VerifyGitOpsReconciliation", appEnvID)
	return mapExecutionWorkersError(err)
}

// FinalizeApplicationEnvironmentProvisioning is a placeholder for the final transition
// of the ApplicationEnvironment in control-plane-api. Later this will delegate to
// an adapter that calls the control-plane API.
func FinalizeApplicationEnvironmentProvisioning(ctx context.Context, appEnvID string) error {
	logger := activity.GetLogger(ctx)
	if controlPlaneClient == nil {
		logger.Info("No control-plane client configured; skipping state transition", "appEnvID", appEnvID)
		return nil
	}

	logger.Info("Calling control-plane-api to finalize ApplicationEnvironment provisioning", "appEnvID", appEnvID)
	err := controlPlaneClient.CompleteApplicationEnvironmentProvisioning(ctx, appEnvID)
	if err == nil {
		return nil
	}

	var apiErr *controlplanehttp.Error
	if errors.As(err, &apiErr) {
		logger.Error("control-plane-api error during appenv finalization",
			"appEnvID", appEnvID,
			"control_plane_status", apiErr.Status,
			"control_plane_code", apiErr.Code,
			"control_plane_message", apiErr.Message,
		)
	}

	// Si el control-plane-api devolvió un 4xx, consideramos el error como
	// no-retriable a nivel de Temporal para evitar reintentos inútiles.
	if apiErr != nil && apiErr.Status >= 400 && apiErr.Status < 500 {
		code := apiErr.Code
		if code == "" {
			code = "control_plane_client_error"
		}
		msg := apiErr.Message
		if msg == "" {
			msg = err.Error()
		}
		observability.ObserveDownstreamError("control-plane-api", code, apiErr.Status)
		//nolint:wrapcheck // devolvemos directamente ApplicationError de Temporal para que el workflow pueda inspeccionar Type
		return temporal.NewNonRetryableApplicationError(msg, code, err)
	}

	return err
}

// mapExecutionWorkersError convierte errores provenientes de los adapters HTTP
// de execution-workers en errores de Temporal no-retriables cuando corresponda
// (típicamente, status HTTP 4xx).
func mapExecutionWorkersError(err error) error {
	if err == nil {
		return nil
	}

	var gitErr *gitproviderhttp.Error
	if errors.As(err, &gitErr) && gitErr.Status >= 400 && gitErr.Status < 500 {
		msg := gitErr.Message
		if msg == "" {
			msg = err.Error()
		}
		observability.ObserveDownstreamError("execution-workers", "execution_workers_client_error", gitErr.Status)
		//nolint:wrapcheck // devolvemos directamente ApplicationError de Temporal para que el workflow pueda inspeccionar Type
		return temporal.NewNonRetryableApplicationError(msg, "execution_workers_client_error", err)
	}

	var appEnvErr *appenvprovhttp.Error
	if errors.As(err, &appEnvErr) && appEnvErr.Status >= 400 && appEnvErr.Status < 500 {
		msg := appEnvErr.Message
		if msg == "" {
			msg = err.Error()
		}
		observability.ObserveDownstreamError("execution-workers", "execution_workers_client_error", appEnvErr.Status)
		//nolint:wrapcheck // devolvemos directamente ApplicationError de Temporal para que el workflow pueda inspeccionar Type
		return temporal.NewNonRetryableApplicationError(msg, "execution_workers_client_error", err)
	}

	return err
}

// logExecutionWorkersErrorIfAny enriquece los logs de las actividades con el
// status y detalles del error cuando el fallo proviene de execution-workers.
func logExecutionWorkersErrorIfAny(logger log.Logger, err error, op string, appEnvID string) {
	if err == nil {
		return
	}

	var gitErr *gitproviderhttp.Error
	if errors.As(err, &gitErr) {
		logger.Error("execution-workers git provider error",
			"operation", op,
			"appEnvID", appEnvID,
			"execution_workers_status", gitErr.Status,
			"execution_workers_message", gitErr.Message,
		)
		return
	}

	var appEnvErr *appenvprovhttp.Error
	if errors.As(err, &appEnvErr) {
		logger.Error("execution-workers appenv provider error",
			"operation", op,
			"appEnvID", appEnvID,
			"execution_workers_status", appEnvErr.Status,
			"execution_workers_path", appEnvErr.Path,
			"execution_workers_message", appEnvErr.Message,
		)
		return
	}

	logger.Error("error calling execution-workers",
		"operation", op,
		"appEnvID", appEnvID,
		"error", err,
	)
}
