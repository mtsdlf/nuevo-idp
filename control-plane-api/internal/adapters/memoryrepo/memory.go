package memoryrepo

import (
	"context"
	"sync"

	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

type TeamRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Team
}

func NewTeamRepository() *TeamRepository {
	return &TeamRepository{items: make(map[string]*domain.Team)}
}

func (r *TeamRepository) GetByID(_ context.Context, id string) (*domain.Team, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.items[id]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, nil
}

func (r *TeamRepository) Save(_ context.Context, team *domain.Team) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *team
	r.items[team.ID] = &copy
	return nil
}

type ApplicationRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Application
}

func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{items: make(map[string]*domain.Application)}
}

func (r *ApplicationRepository) GetByID(_ context.Context, id string) (*domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.items[id]; ok {
		copy := *a
		return &copy, nil
	}
	return nil, nil
}

func (r *ApplicationRepository) Save(_ context.Context, app *domain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *app
	r.items[app.ID] = &copy
	return nil
}

type CodeRepositoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.CodeRepository
}

func NewCodeRepositoryRepository() *CodeRepositoryRepository {
	return &CodeRepositoryRepository{items: make(map[string]*domain.CodeRepository)}
}

func (r *CodeRepositoryRepository) GetByID(_ context.Context, id string) (*domain.CodeRepository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if cr, ok := r.items[id]; ok {
		copy := *cr
		return &copy, nil
	}
	return nil, nil
}

func (r *CodeRepositoryRepository) Save(_ context.Context, repo *domain.CodeRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *repo
	r.items[repo.ID] = &copy
	return nil
}

type EnvironmentRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Environment
}

func NewEnvironmentRepository() *EnvironmentRepository {
	return &EnvironmentRepository{items: make(map[string]*domain.Environment)}
}

func (r *EnvironmentRepository) GetByID(_ context.Context, id string) (*domain.Environment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if e, ok := r.items[id]; ok {
		copy := *e
		return &copy, nil
	}
	return nil, nil
}

func (r *EnvironmentRepository) Save(_ context.Context, env *domain.Environment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *env
	r.items[env.ID] = &copy
	return nil
}

type ApplicationEnvironmentRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.ApplicationEnvironment
}

func NewApplicationEnvironmentRepository() *ApplicationEnvironmentRepository {
	return &ApplicationEnvironmentRepository{items: make(map[string]*domain.ApplicationEnvironment)}
}

func (r *ApplicationEnvironmentRepository) GetByID(_ context.Context, id string) (*domain.ApplicationEnvironment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ae, ok := r.items[id]; ok {
		copy := *ae
		return &copy, nil
	}
	return nil, nil
}

func (r *ApplicationEnvironmentRepository) GetByApplicationAndEnvironment(_ context.Context, applicationID, environmentID string) (*domain.ApplicationEnvironment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ae := range r.items {
		if ae.ApplicationID == applicationID && ae.EnvironmentID == environmentID {
			copy := *ae
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *ApplicationEnvironmentRepository) Save(_ context.Context, appEnv *domain.ApplicationEnvironment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *appEnv
	r.items[appEnv.ID] = &copy
	return nil
}

type DeploymentRepositoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.DeploymentRepository
}

func NewDeploymentRepositoryRepository() *DeploymentRepositoryRepository {
	return &DeploymentRepositoryRepository{items: make(map[string]*domain.DeploymentRepository)}
}

func (r *DeploymentRepositoryRepository) GetByID(_ context.Context, id string) (*domain.DeploymentRepository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if dr, ok := r.items[id]; ok {
		copy := *dr
		return &copy, nil
	}
	return nil, nil
}

func (r *DeploymentRepositoryRepository) Save(_ context.Context, repo *domain.DeploymentRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *repo
	r.items[repo.ID] = &copy
	return nil
}

type SecretRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Secret
}

func NewSecretRepository() *SecretRepository {
	return &SecretRepository{items: make(map[string]*domain.Secret)}
}

func (r *SecretRepository) GetByID(_ context.Context, id string) (*domain.Secret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.items[id]; ok {
		copy := *s
		return &copy, nil
	}
	return nil, nil
}

func (r *SecretRepository) Save(_ context.Context, s *domain.Secret) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *s
	r.items[s.ID] = &copy
	return nil
}

type SecretBindingRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.SecretBinding
}

func NewSecretBindingRepository() *SecretBindingRepository {
	return &SecretBindingRepository{items: make(map[string]*domain.SecretBinding)}
}

func (r *SecretBindingRepository) GetByID(_ context.Context, id string) (*domain.SecretBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if b, ok := r.items[id]; ok {
		copy := *b
		return &copy, nil
	}
	return nil, nil
}

func (r *SecretBindingRepository) Save(_ context.Context, b *domain.SecretBinding) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *b
	r.items[b.ID] = &copy
	return nil
}

type GitOpsIntegrationRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.GitOpsIntegration
}

func NewGitOpsIntegrationRepository() *GitOpsIntegrationRepository {
	return &GitOpsIntegrationRepository{items: make(map[string]*domain.GitOpsIntegration)}
}

func (r *GitOpsIntegrationRepository) GetByID(_ context.Context, id string) (*domain.GitOpsIntegration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if gi, ok := r.items[id]; ok {
		copy := *gi
		return &copy, nil
	}
	return nil, nil
}

func (r *GitOpsIntegrationRepository) Save(_ context.Context, gi *domain.GitOpsIntegration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *gi
	r.items[gi.ID] = &copy
	return nil
}
