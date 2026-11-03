package spaces

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/FacileStudio/porte/directory"
)

// importRetry is the first wait after a failed import. It doubles up to
// directory.DefaultInterval, so an undeployed Registre is asked at most once
// every five minutes.
const importRetry = 10 * time.Second

// Run hands Registre Antenne's existing spaces once, retrying with backoff
// until Registre accepts them, and only then keeps the spaces in sync until
// ctx ends. The order is the safety: a sync before the import would remove
// every member Registre has not heard of yet. With no issuer it returns at
// once.
func (s *Service) Run(ctx context.Context) {
	if s.syncer == nil {
		return
	}
	wait := s.retry
	for {
		err := s.importSpaces(ctx)
		if err == nil {
			break
		}
		s.logger.Warn("directory: import failed, retrying", slog.Any("error", err), slog.Duration("in", wait))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, directory.DefaultInterval)
	}
	s.logger.Info("directory: spaces imported into registre")
	s.syncer.Run(ctx)
}

// importSpaces sends every local space and its members to Registre. Antenne
// has no personal spaces, so every space is a shared one. A member is named by
// the Registre subject of the user behind their address when there is one, and
// by the address otherwise.
func (s *Service) importSpaces(ctx context.Context) error {
	var rows []importRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT spaces.name, COALESCE(space_members.user_id, '') AS user_id, COALESCE(space_members.role, '') AS role,
		       COALESCE((SELECT porte_identities.subject
		                   FROM users JOIN porte_identities ON porte_identities.user_id = users.id
		                  WHERE users.email = lower(space_members.user_id)
		                    AND porte_identities.provider = ?
		                  ORDER BY porte_identities.created_at LIMIT 1), '') AS subject
		  FROM spaces LEFT JOIN space_members ON space_members.space_id = spaces.id
		 ORDER BY spaces.created_at, space_members.joined_at`, s.issuer).Scan(&rows).Error
	if err != nil {
		return err
	}
	_, err = s.client.Import(ctx, groupImports(rows))
	return err
}

// importRow is one membership of one local space, or a space with no member
// when UserID is empty.
type importRow struct {
	Name    string
	UserID  string
	Role    Role
	Subject string
}

// groupImports folds membership rows into one import per space, naming each
// member by subject when Antenne knows it and by address otherwise.
func groupImports(rows []importRow) []directory.ImportSpace {
	imports := []directory.ImportSpace{}
	index := map[string]int{}
	for _, row := range rows {
		at, seen := index[row.Name]
		if !seen {
			at = len(imports)
			index[row.Name] = at
			imports = append(imports, directory.ImportSpace{Name: row.Name, Members: []directory.ImportMember{}})
		}
		if row.UserID == "" {
			continue
		}
		member := directory.ImportMember{Subject: row.Subject, Role: string(row.Role)}
		if row.Subject == "" {
			member.Email = strings.ToLower(row.UserID)
		}
		imports[at].Members = append(imports[at].Members, member)
	}
	return imports
}
