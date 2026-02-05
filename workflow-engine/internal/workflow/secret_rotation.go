package workflow

import (
	"context"
	"time"

	"github.com/nuevo-idp/platform/observability"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const SecretRotationTaskQueue = "secret-rotation-task-queue"

// SecretRotationInput modela la intención "SecretRotation" del estado deseado.
// La precondición es que el Secret ya esté en estado Rotating.
type SecretRotationInput struct {
	SecretID string
}

// SecretRotationPort es un puerto estrecho hacia control-plane-api para
// completar la rotación de un Secret.
type SecretRotationPort interface {
	CompleteSecretRotation(ctx context.Context, secretID string) error
}

var secretRotationPort SecretRotationPort

// SetSecretRotationPort permite inyectar la implementación (HTTP client en prod,
// fakes en tests).
func SetSecretRotationPort(p SecretRotationPort) {
	secretRotationPort = p
}

// SecretBindingsRotationPort es un puerto estrecho hacia execution-workers (u otro
// proveedor externo) para propagar la rotación de un Secret a todos sus
// SecretBindings asociados. La implementación concreta decidirá cómo resolver
// los targets (CodeRepository, DeploymentRepository, ApplicationEnvironment, etc.).
type SecretBindingsRotationPort interface {
	UpdateSecretBindingsForSecret(ctx context.Context, secretID string) error
}

var secretBindingsRotationPort SecretBindingsRotationPort

// SetSecretBindingsRotationPort permite inyectar la implementación concreta
// desde main o tests. Si no se configura, el paso de actualización de
// SecretBindings será un no-op con logs para debugging.
func SetSecretBindingsRotationPort(p SecretBindingsRotationPort) {
	secretBindingsRotationPort = p
}

const rotationValidatedSignalName = "RotationValidatedExternally"

// SecretRotation orquesta la rotación de un Secret ya marcado como Rotating.
// Sigue el modelo del JSON:
// - rotate Secret (aquí modelado como actividad sin efectos reales todavía)
// - waitForEvent RotationValidatedExternally con timeout
// - update SecretBindings (por ahora asumimos que ocurre fuera del dominio)
// - transition Secret to Active
func SecretRotation(ctx workflow.Context, input SecretRotationInput) (err error) {
	start := workflow.Now(ctx)
	result := "success"

	info := workflow.GetInfo(ctx)
	if info.Attempt > 1 {
		observability.ObserveWorkflowRetries("SecretRotation", 1)
	}

	defer func() {
		if err != nil {
			result = "error"
		}
		duration := workflow.Now(ctx).Sub(start).Seconds()
		observability.ObserveWorkflowDuration("SecretRotation", result, duration)
	}()

	if input.SecretID == "" {
		return temporal.NewNonRetryableApplicationError("SecretID is required", "bad_input", nil)
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

	// 1. Rotar el secreto (aún sin efectos en execution-workers).
	if err := workflow.ExecuteActivity(ctx, PerformSecretRotation, input.SecretID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_secret_rotation_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 2. Esperar la validación externa de la rotación.
	if err := waitForRotationValidatedExternally(ctx); err != nil {
		observability.ObserveDomainEvent("workflow_secret_rotation_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 3. Actualizar los SecretBindings asociados al Secret en sistemas externos.
	if err := workflow.ExecuteActivity(ctx, UpdateSecretBindingsForSecret, input.SecretID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_secret_rotation_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	// 4. Completar la rotación en el control-plane.
	if err := workflow.ExecuteActivity(ctx, CompleteSecretRotationActivity, input.SecretID).Get(ctx, nil); err != nil {
		observability.ObserveDomainEvent("workflow_secret_rotation_failed", "error")
		//nolint:wrapcheck // propagamos el error tal cual para preservar el tipo de ApplicationError
		return err
	}

	observability.ObserveDomainEvent("workflow_secret_rotation_completed", "success")
	return nil
}

func waitForRotationValidatedExternally(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Waiting for RotationValidatedExternally event")

	signalCh := workflow.GetSignalChannel(ctx, rotationValidatedSignalName)
	timer := workflow.NewTimer(ctx, 60*time.Minute)

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
		logger.Info("Received RotationValidatedExternally signal")
		return nil
	}

	logger.Info("Secret rotation validation timeout reached")
	return temporal.NewNonRetryableApplicationError("secret rotation timeout", "secret_rotation_timeout", nil)
}

// PerformSecretRotation es una actividad placeholder; la rotación real del
// secreto puede vivir más adelante en execution-workers.
func PerformSecretRotation(ctx context.Context, secretID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing secret rotation (placeholder)", "secretId", secretID)
	return nil
}

// UpdateSecretBindingsForSecret es una actividad que delega en
// SecretBindingsRotationPort para propagar la rotación del Secret a todos
// los SecretBindings asociados en sistemas externos (repos, runtimes, etc.).
//
// Si no hay puerto configurado, se comporta como no-op pero deja trazas de
// log para facilitar el debugging temprano.
func UpdateSecretBindingsForSecret(ctx context.Context, secretID string) error {
	logger := activity.GetLogger(ctx)
	if secretBindingsRotationPort == nil {
		logger.Info("No SecretBindingsRotationPort configured; skipping SecretBindings update", "secretId", secretID)
		return nil
	}

	logger.Info("Updating SecretBindings for Secret via external provider", "secretId", secretID)
	if err := secretBindingsRotationPort.UpdateSecretBindingsForSecret(ctx, secretID); err != nil {
		logger.Error("error updating SecretBindings for Secret", "secretId", secretID, "error", err)
		//nolint:wrapcheck // propagamos el error del puerto tal cual para preservar su tipo
		return err
	}

	return nil
}

// CompleteSecretRotationActivity delega en SecretRotationPort para mover el
// Secret de Rotating a Active en el control-plane.
func CompleteSecretRotationActivity(ctx context.Context, secretID string) error {
	logger := activity.GetLogger(ctx)
	if secretRotationPort == nil {
		logger.Info("No SecretRotationPort configured; skipping CompleteSecretRotation", "secretId", secretID)
		return nil
	}

	logger.Info("Completing secret rotation via control-plane-api", "secretId", secretID)
	err := secretRotationPort.CompleteSecretRotation(ctx, secretID)
	logControlPlaneErrorIfAny(logger, err, "CompleteSecretRotation")
	return mapControlPlaneError(err)
}
