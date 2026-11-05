// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

type ManagedUser struct {
	User
	IsActive         bool       `json:"is_active"`
	LastLoginAt      *time.Time `json:"last_login_at"`
	CanDeactivate    bool       `json:"can_deactivate"`
	DeactivateReason string     `json:"deactivate_reason"`
}
type ManagedUsers struct {
	Users      []ManagedUser `json:"users"`
	NextBefore *uuid.UUID    `json:"next_before,omitempty"`
}

func (s *Service) ManagedUsers(ctx context.Context, actor *User, query string, before uuid.UUID, limit int) (ManagedUsers, error) {
	page := ManagedUsers{Users: []ManagedUser{}}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return page, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := authorizeManagement(ctx, tx, actor, false); err != nil {
		return page, err
	}
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 254 || limit < 1 || limit > 200 {
		return page, ErrInvalidInput
	}
	pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(strings.TrimSpace(query))) + "%"
	count, err := rbac.NewSQLReader(tx, nil).ActiveRootAdministrators(ctx)
	if err != nil {
		return page, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT u.id,u.email,u.display_name,`+roleDisplay+`,u.is_active,u.last_login_at,EXISTS(SELECT 1 FROM assignments a WHERE a.user_id=u.id AND a.role_id=$4 AND a.scope_type='unit' AND a.scope_id=$5) FROM users u WHERE (lower(u.email) LIKE $1 OR lower(u.display_name) LIKE $1) AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR (u.email,u.id)>(SELECT email,id FROM users WHERE id=$2)) ORDER BY u.email,u.id LIMIT $3`, pattern, before, limit+1, uuid.MustParse(rbac.AdministratorRoleID), rbac.RootScope().ID)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ManagedUser
		var rootAdministrator bool
		if err := rows.Scan(&item.ID, &item.Email, &item.DisplayName, &item.Role, &item.IsActive, &item.LastLoginAt, &rootAdministrator); err != nil {
			return page, err
		}
		item.CanDeactivate = item.IsActive && (!rootAdministrator || count > 1)
		if item.IsActive && !item.CanDeactivate {
			item.DeactivateReason = "The last active root Administrator cannot be deactivated."
		}
		page.Users = append(page.Users, item)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	if len(page.Users) > limit {
		page.Users = page.Users[:limit]
		id := uuid.MustParse(page.Users[limit-1].ID)
		page.NextBefore = &id
	}
	return page, tx.Commit()
}
