package pgrepo

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/nuevo-idp/control-plane-api/internal/domain"
)

type TeamRepository struct {
    pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
    return &TeamRepository{pool: pool}
}

func (r *TeamRepository) GetByID(ctx context.Context, id string) (*domain.Team, error) {
    const query = `SELECT id, name, state, created_by, created_at FROM teams WHERE id = $1`

    var (
        team      domain.Team
        createdBy string
        createdAt time.Time
    )

    row := r.pool.QueryRow(ctx, query, id)
    if err := row.Scan(&team.ID, &team.Name, &team.State, &createdBy, &createdAt); err != nil {
        if errors.Is(err, pgx.ErrNoRows) { //nolint:typecheck // golangci-lint en contenedor no resuelve pgx correctamente, pero el build real sí
            return nil, nil
        }
        return nil, fmt.Errorf("scanning team: %w", err)
    }

    team.Metadata.CreatedBy = createdBy
    team.Metadata.CreatedAt = createdAt
    return &team, nil
}

func (r *TeamRepository) Save(ctx context.Context, team *domain.Team) error {
    const stmt = `INSERT INTO teams (id, name, state, created_by, created_at)
                  VALUES ($1, $2, $3, $4, $5)
                  ON CONFLICT (id) DO UPDATE
                  SET name = EXCLUDED.name,
                      state = EXCLUDED.state`

    _, err := r.pool.Exec(ctx, stmt,
        team.ID,
        team.Name,
        team.State,
        team.Metadata.CreatedBy,
        team.Metadata.CreatedAt,
    )
    if err != nil {
        return fmt.Errorf("saving team: %w", err)
    }
    return nil
}
