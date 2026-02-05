package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/nuevo-idp/workflow-engine/internal/adapters/controlplanehttp"
	"github.com/nuevo-idp/platform/observability"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const ApplicationOnboardingTaskQueue = "application-onboarding-task-queue"

// ApplicationOnboardingInput modela la intención "ApplicationOnboarding" del
// ejemplo de estado deseado. Es deliberadamente mínima: el resto del contexto
// vive en el control-plane-api.
type ApplicationOnboardingInput struct {
	ApplicationID string
}

// ApplicationActivationInput modela la intención "ApplicationActivation" del
// estado deseado. También es mínima: el workflow asume que las //nolint:misspell // comentarios en español, "asume" es correcto
// precondiciones (todos los ApplicationEnvironment activos) ya se cumplieron.
type ApplicationActivationInput struct {
	ApplicationID string
}

// ApplicationOnboardingPort es un puerto estrecho hacia control-plane-api para
// realizar las operaciones de dominio necesarias durante el onboarding.
type ApplicationOnboardingPort interface {
	DeclareCodeRepository(ctx context.Context, applicationID string) error
	DeclareDeploymentRepository(ctx context.Context, applicationID string) error
	DeclareGitOpsIntegration(ctx context.Context, applicationID string) error
	DeclareApplicationEnvironments(ctx context.Context, applicationID string) error
	MarkApplicationOnboarding(ctx context.Context, applicationID string) error
	ActivateApplication(ctx context.Context, applicationID string) error
}

var applicationOnboardingPort ApplicationOnboardingPort

// SetApplicationOnboardingPort permite a main y a los tests inyectar una
// implementación concreta (adapter HTTP en producción, fakes en tests).
func SetApplicationOnboardingPort(p ApplicationOnboardingPort) {
	applicationOnboardingPort = p
}

const securityScanPassedSignalName = "SecurityScanPassed"

// ApplicationOnboarding orquesta el onboarding de una Application ya aprobada.
// Los steps se inspiran en ejemplo_estado_Deseado.json:
// - crear CodeRepository
// - crear DeploymentRepository (si aplica)
// - crear GitOpsIntegration
// - declarar ApplicationEnvironments
// - esperar evento SecurityScanPassed con timeout
// - transicionar Application a Onboarding
func ApplicationOnboarding(ctx workflow.Context, input ApplicationOnboardingInput) (err error) {
	start := workflow.Now(ctx)
	result := "success"

	info := workflow.GetInfo(ctx)
	if info.Attempt > 1 {
		observability.ObserveWorkflowRetries("ApplicationOnboarding", 1)
	}

	defer func() {
		if err != nil {
			result = "error"
		}
		duration := workflow.Now(ctx).Sub(start).Seconds()
		observability.ObserveWorkflowDuration("ApplicationOnboarding", result, duration)
	}()

	if input.ApplicationID == "" {
		return temporal.NewNonRetryableApplicationError("ApplicationID is required", "bad_input", nil)
	}

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

	// 1. Crear CodeRepository para la aplicación.
	if err := workflow.ExecuteActivity(ctx, CreateCodeRepositoryForApplication, input.ApplicationID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_application_onboarding_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 2. Crear DeploymentRepository si aplica. El adapter decidirá si realmente
	// crea algo o si es un no-op según el modelo de despliegue.
	if err := workflow.ExecuteActivity(ctx, CreateDeploymentRepositoryForApplication, input.ApplicationID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_application_onboarding_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 3. Crear GitOpsIntegration para la aplicación.
	if err := workflow.ExecuteActivity(ctx, CreateGitOpsIntegrationForApplication, input.ApplicationID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_application_onboarding_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 4. Declarar los ApplicationEnvironments necesarios para la aplicación.
	if err := workflow.ExecuteActivity(ctx, DeclareApplicationEnvironmentsForApplication, input.ApplicationID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_application_onboarding_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 5. Esperar evento externo SecurityScanPassed con timeout.
	if err := waitForSecurityScanPassed(ctx); err != nil {
		observability.ObserveDomainEvent("workflow_application_onboarding_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 6. Transicionar la Application a estado Onboarding.
	if err := workflow.ExecuteActivity(ctx, TransitionApplicationToOnboarding, input.ApplicationID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_application_onboarding_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	observability.ObserveDomainEvent("workflow_application_onboarding_completed", "success")
	return nil
}

func waitForSecurityScanPassed(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Waiting for SecurityScanPassed event")

	signalCh := workflow.GetSignalChannel(ctx, securityScanPassedSignalName)
	timer := workflow.NewTimer(ctx, 15*time.Minute) // 900s como en el JSON de estado deseado

	selector := workflow.NewSelector(ctx)
	var received bool

	selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
		var payload interface{}
		c.Receive(ctx, &payload)
		received = true
	})
	selector.AddFuture(timer, func(f workflow.Future) {})

	selector.Select(ctx)

	if received {
		logger.Info("Received SecurityScanPassed signal")
		return nil
	}

	logger.Info("Security scan timeout reached")
	return temporal.NewNonRetryableApplicationError("security scan timeout", "security_scan_timeout", nil)
}

// Las actividades siguientes delegan en el puerto ApplicationOnboardingPort.
// Si no hay puerto configurado, se comportan como no-ops pero dejan trazas de log
// para facilitar el debugging temprano del modelo.

func CreateCodeRepositoryForApplication(ctx context.Context, applicationID string) error {
	logger := activity.GetLogger(ctx)
	if applicationOnboardingPort == nil {
		logger.Info("No ApplicationOnboardingPort configured; skipping CodeRepository creation", "applicationId", applicationID)
		return nil
	}

	logger.Info("Creating CodeRepository for Application", "applicationId", applicationID)
	err := applicationOnboardingPort.DeclareCodeRepository(ctx, applicationID)
	logControlPlaneErrorIfAny(logger, err, "DeclareCodeRepository")
	return mapControlPlaneError(err)
}

func CreateDeploymentRepositoryForApplication(ctx context.Context, applicationID string) error {
	logger := activity.GetLogger(ctx)
	if applicationOnboardingPort == nil {
		logger.Info("No ApplicationOnboardingPort configured; skipping DeploymentRepository creation", "applicationId", applicationID)
		return nil
	}

	logger.Info("Creating DeploymentRepository for Application", "applicationId", applicationID)
	err := applicationOnboardingPort.DeclareDeploymentRepository(ctx, applicationID)
	logControlPlaneErrorIfAny(logger, err, "DeclareDeploymentRepository")
	return mapControlPlaneError(err)
}

func CreateGitOpsIntegrationForApplication(ctx context.Context, applicationID string) error {
	logger := activity.GetLogger(ctx)
	if applicationOnboardingPort == nil {
		logger.Info("No ApplicationOnboardingPort configured; skipping GitOpsIntegration creation", "applicationId", applicationID)
		return nil
	}

	logger.Info("Creating GitOpsIntegration for Application", "applicationId", applicationID)
	err := applicationOnboardingPort.DeclareGitOpsIntegration(ctx, applicationID)
	logControlPlaneErrorIfAny(logger, err, "DeclareGitOpsIntegration")
	return mapControlPlaneError(err)
}

func DeclareApplicationEnvironmentsForApplication(ctx context.Context, applicationID string) error {
	logger := activity.GetLogger(ctx)
	if applicationOnboardingPort == nil {
		logger.Info("No ApplicationOnboardingPort configured; skipping ApplicationEnvironments declaration", "applicationId", applicationID)
		return nil
	}

	logger.Info("Declaring ApplicationEnvironments for Application", "applicationId", applicationID)
	err := applicationOnboardingPort.DeclareApplicationEnvironments(ctx, applicationID)
	logControlPlaneErrorIfAny(logger, err, "DeclareApplicationEnvironments")
	return mapControlPlaneError(err)
}

func TransitionApplicationToOnboarding(ctx context.Context, applicationID string) error {
	logger := activity.GetLogger(ctx)
	if applicationOnboardingPort == nil {
		logger.Info("No ApplicationOnboardingPort configured; skipping Application state transition to Onboarding", "applicationId", applicationID)
		return nil
	}

	logger.Info("Marking Application as Onboarding", "applicationId", applicationID)
	err := applicationOnboardingPort.MarkApplicationOnboarding(ctx, applicationID)
	logControlPlaneErrorIfAny(logger, err, "MarkApplicationOnboarding")
	return mapControlPlaneError(err)
}

// ApplicationActivation es un workflow simple que realiza la transición final
// de la Application a Active. Asume que todas las precondiciones ya fueron //nolint:misspell // comentarios en español, "Asume" es correcto
// validadas antes de dispararlo (por ejemplo, todos los ApplicationEnvironment
// están en estado Active).
func ApplicationActivation(ctx workflow.Context, input ApplicationActivationInput) (err error) {
	start := workflow.Now(ctx)
	result := "success"

	info := workflow.GetInfo(ctx)
	if info.Attempt > 1 {
		observability.ObserveWorkflowRetries("ApplicationActivation", 1)
	}

	defer func() {
		if err != nil {
			result = "error"
		}
		duration := workflow.Now(ctx).Sub(start).Seconds()
		observability.ObserveWorkflowDuration("ApplicationActivation", result, duration)
	}()

	if input.ApplicationID == "" {
		return temporal.NewNonRetryableApplicationError("ApplicationID is required", "bad_input", nil)
	}

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

	if err := workflow.ExecuteActivity(ctx, TransitionApplicationToActive, input.ApplicationID).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

// TransitionApplicationToActive delega en el puerto para mover la Application
// a estado Active.
func TransitionApplicationToActive(ctx context.Context, applicationID string) error {
	logger := activity.GetLogger(ctx)
	if applicationOnboardingPort == nil {
		logger.Info("No ApplicationOnboardingPort configured; skipping Application state transition to Active", "applicationId", applicationID)
		return nil
	}

	logger.Info("Marking Application as Active", "applicationId", applicationID)
	err := applicationOnboardingPort.ActivateApplication(ctx, applicationID)
	logControlPlaneErrorIfAny(logger, err, "ActivateApplication")
	return mapControlPlaneError(err)
}

// logControlPlaneErrorIfAny enriquece los logs de las actividades con status y
// código de error cuando el fallo proviene del control-plane-api.
//
// Recibe una interfaz mínima compatible con el logger de Temporal
// (que expone Error(msg string, keysAndValues ...interface{})).
func logControlPlaneErrorIfAny(logger interface{ Error(msg string, keysAndValues ...interface{}) }, err error, op string) {
	if err == nil {
		return
	}

	var apiErr *controlplanehttp.Error
	if errors.As(err, &apiErr) {
		logger.Error("control-plane-api error during onboarding operation",
			"operation", op,
			"control_plane_status", apiErr.Status,
			"control_plane_code", apiErr.Code,
			"control_plane_message", apiErr.Message,
		)
		return
	}

	logger.Error("error calling control-plane-api during onboarding operation",
		"operation", op,
		"error", err,
	)
}

// mapControlPlaneError convierte errores provenientes del adapter HTTP de
// control-plane-api en errores de Temporal no-retriables cuando corresponda
// (típicamente, status HTTP 4xx). Esto evita reintentos inútiles cuando el
// fallo es de dominio/validación en lugar de un problema transitorio.
func mapControlPlaneError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *controlplanehttp.Error
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		code := apiErr.Code
		if code == "" {
			code = "control_plane_client_error"
		}
		msg := apiErr.Message
		if msg == "" {
			msg = err.Error()
		}

		observability.ObserveDownstreamError("control-plane-api", code, apiErr.Status)
		return temporal.NewNonRetryableApplicationError(msg, code, err)
	}

	return err
}
