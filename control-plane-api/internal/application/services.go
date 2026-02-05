package application

import (
	"context"
	"fmt"
	"time"

	"github.com/nuevo-idp/control-plane-api/internal/domain"
	perrors "github.com/nuevo-idp/platform/errors"
)

var (
	ErrTeamAlreadyExists              = perrors.Conflict("team_already_exists", "team already exists", nil)
	ErrApplicationAlreadyExists       = perrors.Conflict("application_already_exists", "application already exists", nil)
	ErrApplicationEnvironmentNotFound = perrors.NotFound("application_environment_not_found", "application environment not found", nil)
)

type TeamRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Team, error)
	Save(ctx context.Context, team *domain.Team) error
}

type ApplicationRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	Save(ctx context.Context, app *domain.Application) error
}

type CodeRepositoryRepository interface {
	GetByID(ctx context.Context, id string) (*domain.CodeRepository, error)
	Save(ctx context.Context, repo *domain.CodeRepository) error
}

type EnvironmentRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Environment, error)
	Save(ctx context.Context, env *domain.Environment) error
}

type ApplicationEnvironmentRepository interface {
	GetByID(ctx context.Context, id string) (*domain.ApplicationEnvironment, error)
	GetByApplicationAndEnvironment(ctx context.Context, applicationID, environmentID string) (*domain.ApplicationEnvironment, error)
	Save(ctx context.Context, appEnv *domain.ApplicationEnvironment) error
}

type SecretRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Secret, error)
	Save(ctx context.Context, s *domain.Secret) error
}

type SecretBindingRepository interface {
	GetByID(ctx context.Context, id string) (*domain.SecretBinding, error)
	Save(ctx context.Context, b *domain.SecretBinding) error
}

type DeploymentRepositoryRepository interface {
	GetByID(ctx context.Context, id string) (*domain.DeploymentRepository, error)
	Save(ctx context.Context, repo *domain.DeploymentRepository) error
}

type GitOpsIntegrationRepository interface {
	GetByID(ctx context.Context, id string) (*domain.GitOpsIntegration, error)
	Save(ctx context.Context, gi *domain.GitOpsIntegration) error
}

type Services struct {
	Teams                   TeamRepository
	Applications            ApplicationRepository
	CodeRepositories        CodeRepositoryRepository
	Environments            EnvironmentRepository
	ApplicationEnvironments ApplicationEnvironmentRepository
	Secrets                 SecretRepository
	SecretBindings          SecretBindingRepository
	DeploymentRepositories  DeploymentRepositoryRepository
	GitOpsIntegrations      GitOpsIntegrationRepository
}

func (s *Services) GetApplication(ctx context.Context, id string) (*domain.Application, error) {
	if s.Applications == nil {
		return nil, perrors.Internal("application_repository_not_configured", "application repository not configured", nil)
	}

	app, err := s.Applications.GetByID(ctx, id)
	if err != nil {
		return nil, perrors.Internal("application_repository_error", "error loading application", err)
	}
	if app == nil {
		return nil, perrors.NotFound("application_not_found", "application not found", nil)
	}

	return app, nil
}

func (s *Services) GetEnvironment(ctx context.Context, id string) (*domain.Environment, error) {
	if s.Environments == nil {
		return nil, perrors.Internal("environment_repository_not_configured", "environment repository not configured", nil)
	}

	env, err := s.Environments.GetByID(ctx, id)
	if err != nil {
		return nil, perrors.Internal("environment_repository_error", "error loading environment", err)
	}
	if env == nil {
		return nil, perrors.NotFound("environment_not_found", "environment not found", nil)
	}

	return env, nil
}

func (s *Services) GetApplicationEnvironment(ctx context.Context, id string) (*domain.ApplicationEnvironment, error) {
	if s.ApplicationEnvironments == nil {
		return nil, perrors.Internal("application_environment_repository_not_configured", "application environment repository not configured", nil)
	}

	ae, err := s.ApplicationEnvironments.GetByID(ctx, id)
	if err != nil {
		return nil, perrors.Internal("application_environment_repository_error", "error loading application environment", err)
	}
	if ae == nil {
		return nil, perrors.NotFound("application_environment_not_found", "application environment not found", nil)
	}

	return ae, nil
}

func (s *Services) CreateTeam(ctx context.Context, id, name, createdBy string) error {
	if s.Teams == nil {
		return perrors.Internal("team_repository_not_configured", "team repository not configured", nil)
	}

	if existing, _ := s.Teams.GetByID(ctx, id); existing != nil {
		return ErrTeamAlreadyExists
	}

	team := &domain.Team{
		ID:    id,
		Name:  name,
		State: domain.TeamStateDraft,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.Teams.Save(ctx, team); err != nil {
		return fmt.Errorf("saving team: %w", err)
	}

	return nil
}

func (s *Services) CreateApplication(ctx context.Context, id, name, teamID, createdBy string) error {
	if s.Applications == nil || s.Teams == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if existing, _ := s.Applications.GetByID(ctx, id); existing != nil {
		return ErrApplicationAlreadyExists
	}

	team, err := s.Teams.GetByID(ctx, teamID)
	if err != nil || team == nil {
		return perrors.NotFound("team_not_found", "team not found", err)
	}

	app := &domain.Application{
		ID:     id,
		Name:   name,
		TeamID: teamID,
		State:  domain.ApplicationStateProposed,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.Applications.Save(ctx, app); err != nil {
		return fmt.Errorf("saving application: %w", err)
	}

	return nil
}

// ApproveApplication transitions an Application from Proposed to Approved.
// Este método modela el "onApplicationApproved" del estado deseado: una vez
// en Approved, un workflow de onboarding puede ser disparado.
func (s *Services) ApproveApplication(ctx context.Context, id, approvedBy string) error {
	if s.Applications == nil {
		return perrors.Internal("application_repository_not_configured", "application repository not configured", nil)
	}

	app, err := s.Applications.GetByID(ctx, id)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	if app.State != domain.ApplicationStateProposed {
		return perrors.Domain("application_invalid_state_for_approval", "application can only be approved from Proposed state", nil)
	}

	app.State = domain.ApplicationStateApproved
	_ = approvedBy

	if err := s.Applications.Save(ctx, app); err != nil {
		return fmt.Errorf("saving approved application: %w", err)
	}

	return nil
}

// StartApplicationOnboarding mueve una Application de Approved a Onboarding.
// Modela la transición realizada por el workflow ApplicationOnboarding una vez
// cumplidas sus precondiciones.
func (s *Services) StartApplicationOnboarding(ctx context.Context, id, startedBy string) error {
	if s.Applications == nil {
		return perrors.Internal("application_repository_not_configured", "application repository not configured", nil)
	}

	app, err := s.Applications.GetByID(ctx, id)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	if app.State != domain.ApplicationStateApproved {
		return perrors.Domain("application_invalid_state_for_onboarding", "application can only start onboarding from Approved state", nil)
	}

	app.State = domain.ApplicationStateOnboarding
	_ = startedBy

	if err := s.Applications.Save(ctx, app); err != nil {
		return fmt.Errorf("starting application onboarding: %w", err)
	}

	return nil
}

// ActivateApplication mueve una Application de Onboarding a Active.
// Modela la transición realizada por el workflow ApplicationActivation
// una vez que todos los ApplicationEnvironment están activos.
func (s *Services) ActivateApplication(ctx context.Context, id, activatedBy string) error {
	if s.Applications == nil {
		return perrors.Internal("application_repository_not_configured", "application repository not configured", nil)
	}

	app, err := s.Applications.GetByID(ctx, id)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	if app.State != domain.ApplicationStateOnboarding {
		return perrors.Domain("application_invalid_state_for_activation", "application can only be activated from Onboarding state", nil)
	}

	app.State = domain.ApplicationStateActive
	_ = activatedBy

	if err := s.Applications.Save(ctx, app); err != nil {
		return fmt.Errorf("activating application: %w", err)
	}

	return nil
}

// DeclareCodeRepository creates a CodeRepository in Declared state for an existing Application.
func (s *Services) DeclareCodeRepository(ctx context.Context, id, applicationID, createdBy string) error {
	if s.CodeRepositories == nil || s.Applications == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if existing, _ := s.CodeRepositories.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("code_repository_already_exists", "code repository already exists", nil)
	}

	app, err := s.Applications.GetByID(ctx, applicationID)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	repo := &domain.CodeRepository{
		ID:            id,
		ApplicationID: applicationID,
		State:         domain.CodeRepositoryStateDeclared,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.CodeRepositories.Save(ctx, repo); err != nil {
		return fmt.Errorf("saving code repository: %w", err)
	}

	return nil
}

// CreateEnvironment declares a new global Environment in Planned state.
func (s *Services) CreateEnvironment(ctx context.Context, id, name, createdBy string) error {
	if s.Environments == nil {
		return perrors.Internal("environment_repository_not_configured", "environment repository not configured", nil)
	}

	if existing, _ := s.Environments.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("environment_already_exists", "environment already exists", nil)
	}

	env := &domain.Environment{
		ID:    id,
		Name:  name,
		State: domain.EnvironmentStatePlanned,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.Environments.Save(ctx, env); err != nil {
		return fmt.Errorf("saving environment: %w", err)
	}

	return nil
}

// DeclareDeploymentRepository declara un DeploymentRepository asociado a una Application.
func (s *Services) DeclareDeploymentRepository(ctx context.Context, id, applicationID, deploymentModel, createdBy string) error {
	if s.DeploymentRepositories == nil || s.Applications == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if existing, _ := s.DeploymentRepositories.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("deployment_repository_already_exists", "deployment repository already exists", nil)
	}

	app, err := s.Applications.GetByID(ctx, applicationID)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	repo := &domain.DeploymentRepository{
		ID:              id,
		ApplicationID:   applicationID,
		DeploymentModel: deploymentModel,
		State:           domain.DeploymentRepositoryStateDeclared,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.DeploymentRepositories.Save(ctx, repo); err != nil {
		return fmt.Errorf("saving deployment repository: %w", err)
	}

	return nil
}

// DeclareApplicationEnvironment creates the relation between an Application and an Environment
// in Declared state, enforcing uniqueness of the pair at the domain level.
func (s *Services) DeclareApplicationEnvironment(ctx context.Context, id, applicationID, environmentID, createdBy string) error {
	if s.ApplicationEnvironments == nil || s.Applications == nil || s.Environments == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if existing, _ := s.ApplicationEnvironments.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("application_environment_already_exists", "application environment already exists", nil)
	}

	// Ensure application exists
	app, err := s.Applications.GetByID(ctx, applicationID)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	// Ensure environment exists
	env, err := s.Environments.GetByID(ctx, environmentID)
	if err != nil || env == nil {
		return perrors.NotFound("environment_not_found", "environment not found", err)
	}

	// Enforce unique_application_environment_pair
	if existingPair, _ := s.ApplicationEnvironments.GetByApplicationAndEnvironment(ctx, applicationID, environmentID); existingPair != nil {
		return perrors.Conflict("application_environment_pair_already_exists", "application environment pair already exists", nil)
	}

	appEnv := &domain.ApplicationEnvironment{
		ID:            id,
		ApplicationID: applicationID,
		EnvironmentID: environmentID,
		State:         domain.ApplicationEnvironmentStateDeclared,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.ApplicationEnvironments.Save(ctx, appEnv); err != nil {
		return fmt.Errorf("saving application environment: %w", err)
	}

	return nil
}

// CompleteApplicationEnvironmentProvisioning marks an ApplicationEnvironment as Active
// after a successful provisioning workflow. For now it allows transitions from
// Declared or Provisioning into Active.
func (s *Services) CompleteApplicationEnvironmentProvisioning(ctx context.Context, id, completedBy string) error {
	if s.ApplicationEnvironments == nil {
		return perrors.Internal("application_environment_repository_not_configured", "application environment repository not configured", nil)
	}

	appEnv, err := s.ApplicationEnvironments.GetByID(ctx, id)
	if err != nil || appEnv == nil {
		return ErrApplicationEnvironmentNotFound
	}

	if appEnv.State != domain.ApplicationEnvironmentStateDeclared && appEnv.State != domain.ApplicationEnvironmentStateProvisioning {
		return perrors.Domain("application_environment_invalid_state_for_activation", "application environment cannot be activated from current state", nil)
	}

	appEnv.State = domain.ApplicationEnvironmentStateActive
	// For ahora mantenemos solo Created* en Metadata; podríamos extender con Updated* más adelante.
	_ = completedBy

	if err := s.ApplicationEnvironments.Save(ctx, appEnv); err != nil {
		return fmt.Errorf("completing application environment provisioning: %w", err)
	}

	return nil
}

// DeprecateApplication marca una Application como Deprecated desde Active.
// Este estado es precondición para el workflow de ApplicationDecommissioning.
func (s *Services) DeprecateApplication(ctx context.Context, id, deprecatedBy string) error {
	if s.Applications == nil {
		return perrors.Internal("application_repository_not_configured", "application repository not configured", nil)
	}

	app, err := s.Applications.GetByID(ctx, id)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	if app.State != domain.ApplicationStateActive {
		return perrors.Domain("application_invalid_state_for_deprecation", "application can only be deprecated from Active state", nil)
	}

	app.State = domain.ApplicationStateDeprecated
	_ = deprecatedBy

	if err := s.Applications.Save(ctx, app); err != nil {
		return fmt.Errorf("deprecating application: %w", err)
	}

	return nil
}

// DeclareGitOpsIntegration crea la relación GitOpsIntegration entre una Application
// y un DeploymentRepository concreto.
func (s *Services) DeclareGitOpsIntegration(ctx context.Context, id, applicationID, deploymentRepoID, createdBy string) error {
	if s.GitOpsIntegrations == nil || s.DeploymentRepositories == nil || s.Applications == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if existing, _ := s.GitOpsIntegrations.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("gitops_integration_already_exists", "gitops integration already exists", nil)
	}

	app, err := s.Applications.GetByID(ctx, applicationID)
	if err != nil || app == nil {
		return perrors.NotFound("application_not_found", "application not found", err)
	}

	dep, err := s.DeploymentRepositories.GetByID(ctx, deploymentRepoID)
	if err != nil || dep == nil {
		return perrors.NotFound("deployment_repository_not_found", "deployment repository not found", err)
	}

	if dep.ApplicationID != applicationID {
		return perrors.Domain("deployment_repository_wrong_application", "deployment repository does not belong to application", nil)
	}

	gi := &domain.GitOpsIntegration{
		ID:                     id,
		ApplicationID:          applicationID,
		DeploymentRepositoryID: deploymentRepoID,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.GitOpsIntegrations.Save(ctx, gi); err != nil {
		return fmt.Errorf("saving gitops integration: %w", err)
	}

	return nil
}

// CreateSecret creates a Secret in Declared state.
// Invariants:
// - secret_must_have_owner_team
func (s *Services) CreateSecret(ctx context.Context, id, ownerTeamID, purpose, sensitivity, createdBy string) error {
	if s.Secrets == nil || s.Teams == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if ownerTeamID == "" {
		return perrors.Validation("owner_team_required", "owner team is required", nil)
	}

	if existing, _ := s.Secrets.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("secret_already_exists", "secret already exists", nil)
	}

	team, err := s.Teams.GetByID(ctx, ownerTeamID)
	if err != nil || team == nil {
		return perrors.NotFound("owner_team_not_found", "owner team not found", err)
	}

	secret := &domain.Secret{
		ID:          id,
		OwnerTeam:   ownerTeamID,
		Purpose:     purpose,
		Sensitivity: sensitivity,
		State:       domain.SecretStateDeclared,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.Secrets.Save(ctx, secret); err != nil {
		return fmt.Errorf("saving secret: %w", err)
	}

	return nil
}

// StartSecretRotation mueve un Secret de Active a Rotating, lo que modela la
// precondición del workflow SecretRotation.
func (s *Services) StartSecretRotation(ctx context.Context, id, startedBy string) error {
	if s.Secrets == nil {
		return perrors.Internal("secret_repository_not_configured", "secret repository not configured", nil)
	}

	sec, err := s.Secrets.GetByID(ctx, id)
	if err != nil || sec == nil {
		return perrors.NotFound("secret_not_found", "secret not found", err)
	}

	if sec.State != domain.SecretStateActive {
		return perrors.Domain("secret_invalid_state_for_start_rotation", "secret can only start rotation from Active state", nil)
	}

	sec.State = domain.SecretStateRotating
	_ = startedBy

	if err := s.Secrets.Save(ctx, sec); err != nil {
		return fmt.Errorf("starting secret rotation: %w", err)
	}

	return nil
}

// CompleteSecretRotation mueve un Secret de Rotating a Active una vez que la
// rotación fue validada externamente. Modela el paso final del workflow
// SecretRotation.
func (s *Services) CompleteSecretRotation(ctx context.Context, id, completedBy string) error {
	if s.Secrets == nil {
		return perrors.Internal("secret_repository_not_configured", "secret repository not configured", nil)
	}

	sec, err := s.Secrets.GetByID(ctx, id)
	if err != nil || sec == nil {
		return perrors.NotFound("secret_not_found", "secret not found", err)
	}

	if sec.State != domain.SecretStateRotating {
		return perrors.Domain("secret_invalid_state_for_complete_rotation", "secret can only complete rotation from Rotating state", nil)
	}

	sec.State = domain.SecretStateActive
	_ = completedBy

	if err := s.Secrets.Save(ctx, sec); err != nil {
		return fmt.Errorf("completing secret rotation: %w", err)
	}

	return nil
}

// DeclareSecretBinding creates a binding from Secret to a target resource.
// Invariants:
// - binding_requires_active_secret
func (s *Services) DeclareSecretBinding(ctx context.Context, id, secretID, targetID, targetType, createdBy string) error {
	if s.SecretBindings == nil || s.Secrets == nil {
		return perrors.Internal("repositories_not_configured", "repositories not configured", nil)
	}

	if existing, _ := s.SecretBindings.GetByID(ctx, id); existing != nil {
		return perrors.Conflict("secret_binding_already_exists", "secret binding already exists", nil)
	}

	secret, err := s.Secrets.GetByID(ctx, secretID)
	if err != nil || secret == nil {
		return perrors.NotFound("secret_not_found", "secret not found", err)
	}

	if secret.State != domain.SecretStateActive {
		return perrors.Domain("binding_requires_active_secret", "binding requires active secret", nil)
	}

	binding := &domain.SecretBinding{
		ID:         id,
		SecretID:   secretID,
		TargetID:   targetID,
		TargetType: targetType,
		State:      domain.SecretBindingStateDeclared,
		Metadata: domain.Metadata{
			CreatedBy: createdBy,
			CreatedAt: time.Now().UTC(),
		},
	}

	if err := s.SecretBindings.Save(ctx, binding); err != nil {
		return fmt.Errorf("saving secret binding: %w", err)
	}

	return nil
}
