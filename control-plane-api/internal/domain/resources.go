package domain

import "time"

// Core resource aggregates inspired by ejemplo_estado_Deseado.json.

type TeamState string

type ApplicationState string

type CodeRepositoryState string

type DeploymentRepositoryState string

type EnvironmentState string

type ApplicationEnvironmentState string

type SecretState string

type SecretBindingState string

const (
	TeamStateDraft     TeamState = "Draft"
	TeamStateActive    TeamState = "Active"
	TeamStateSuspended TeamState = "Suspended"
	TeamStateArchived  TeamState = "Archived"

	ApplicationStateProposed        ApplicationState = "Proposed"
	ApplicationStateApproved        ApplicationState = "Approved"
	ApplicationStateOnboarding      ApplicationState = "Onboarding"
	ApplicationStateActive          ApplicationState = "Active"
	ApplicationStateDeprecated      ApplicationState = "Deprecated"
	ApplicationStateDecommissioning ApplicationState = "Decommissioning"
	ApplicationStateArchived        ApplicationState = "Archived"

	CodeRepositoryStateDeclared     CodeRepositoryState = "Declared"
	CodeRepositoryStateProvisioning CodeRepositoryState = "Provisioning"
	CodeRepositoryStateActive       CodeRepositoryState = "Active"
	CodeRepositoryStateArchived     CodeRepositoryState = "Archived"

	DeploymentRepositoryStateDeclared     DeploymentRepositoryState = "Declared"
	DeploymentRepositoryStateProvisioning DeploymentRepositoryState = "Provisioning"
	DeploymentRepositoryStateActive       DeploymentRepositoryState = "Active"
	DeploymentRepositoryStateArchived     DeploymentRepositoryState = "Archived"

	EnvironmentStatePlanned  EnvironmentState = "Planned"
	EnvironmentStateActive   EnvironmentState = "Active"
	EnvironmentStateFrozen   EnvironmentState = "Frozen"
	EnvironmentStateRetired  EnvironmentState = "Retired"

	ApplicationEnvironmentStateDeclared       ApplicationEnvironmentState = "Declared"
	ApplicationEnvironmentStateProvisioning   ApplicationEnvironmentState = "Provisioning"
	ApplicationEnvironmentStateActive         ApplicationEnvironmentState = "Active"
	ApplicationEnvironmentStateFrozen         ApplicationEnvironmentState = "Frozen"
	ApplicationEnvironmentStateDecommissioning ApplicationEnvironmentState = "Decommissioning"
	ApplicationEnvironmentStateRetired        ApplicationEnvironmentState = "Retired"

	SecretStateDeclared     SecretState = "Declared"
	SecretStateProvisioning SecretState = "Provisioning"
	SecretStateActive       SecretState = "Active"
	SecretStateRotating     SecretState = "Rotating"
	SecretStateSuspended    SecretState = "Suspended"
	SecretStateRevoked      SecretState = "Revoked"
	SecretStateArchived     SecretState = "Archived"

	SecretBindingStateDeclared     SecretBindingState = "Declared"
	SecretBindingStateProvisioning SecretBindingState = "Provisioning"
	SecretBindingStateActive       SecretBindingState = "Active"
	SecretBindingStateSuspended    SecretBindingState = "Suspended"
	SecretBindingStateRevoked      SecretBindingState = "Revoked"
)

type Metadata struct {
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	Tags      []string  `json:"tags,omitempty"`
}

type Team struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	State    TeamState `json:"state"`
	Metadata Metadata  `json:"metadata"`
}

type Application struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	TeamID   string           `json:"teamId"`
	State    ApplicationState `json:"state"`
	Metadata Metadata         `json:"metadata"`
}

type CodeRepository struct {
	ID            string              `json:"id"`
	ApplicationID string              `json:"applicationId"`
	State         CodeRepositoryState `json:"state"`
	Metadata      Metadata            `json:"metadata"`
}

// DeploymentRepository representa el repositorio de despliegue (GitOps) de una Application.
// El modelo deseado incluye el campo deploymentModel, que dejamos como string
// para no acoplar aún a un enum específico.
type DeploymentRepository struct {
	ID            string                     `json:"id"`
	ApplicationID string                     `json:"applicationId"`
	DeploymentModel string                  `json:"deploymentModel"`
	State         DeploymentRepositoryState `json:"state"`
	Metadata      Metadata                  `json:"metadata"`
}

// GitOpsIntegration vincula una Application con un DeploymentRepository específico.
// En el ejemplo deseado es principalmente una relación, por lo que no modelamos
// un estado explícito por ahora.
type GitOpsIntegration struct {
	ID                   string   `json:"id"`
	ApplicationID        string   `json:"applicationId"`
	DeploymentRepositoryID string `json:"deploymentRepositoryId"`
	Metadata             Metadata `json:"metadata"`
}

// Environment is a global environment (dev, staging, prod...).
type Environment struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	State    EnvironmentState `json:"state"`
	Metadata Metadata         `json:"metadata"`
}

// ApplicationEnvironment links an Application to an Environment.
// Invariants a nivel de dominio (a reforzar en application layer):
// - unique_application_environment_pair
// - runtime_requires_active_application_environment
type ApplicationEnvironment struct {
	ID            string                     `json:"id"`
	ApplicationID string                     `json:"applicationId"`
	EnvironmentID string                     `json:"environmentId"`
	State         ApplicationEnvironmentState `json:"state"`
	Metadata      Metadata                   `json:"metadata"`
}

// Secret representa un secreto propiedad de un Team.
// Invariants a nivel de dominio:
// - secret_must_have_owner_team
// - revoked_secret_cannot_be_reactivated
type Secret struct {
	ID        string      `json:"id"`
	OwnerTeam string      `json:"ownerTeamId"`
	Purpose   string      `json:"purpose"`
	Sensitivity string    `json:"sensitivity"`
	State     SecretState `json:"state"`
	Metadata  Metadata    `json:"metadata"`
}

// SecretBinding vincula un Secret con un recurso objetivo
// (CodeRepository, DeploymentRepository, ApplicationEnvironment...).
// Invariants a nivel de dominio:
// - binding_requires_active_secret
type SecretBinding struct {
	ID        string             `json:"id"`
	SecretID  string             `json:"secretId"`
	TargetID  string             `json:"targetId"`
	TargetType string            `json:"targetType"`
	State     SecretBindingState `json:"state"`
	Metadata  Metadata           `json:"metadata"`
}
