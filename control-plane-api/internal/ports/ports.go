package ports

import "context"

// Command side ports – no external provider details here.

type TeamCommands interface {
	CreateTeam(ctx context.Context, id, name, createdBy string) error
}

type ApplicationCommands interface {
	CreateApplication(ctx context.Context, id, name, teamID, createdBy string) error
}
