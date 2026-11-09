// SPDX-License-Identifier: Apache-2.0
// Package connections owns encrypted PostgreSQL profiles and catalog snapshots.
package connections

import (
	"time"

	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
)

type Input struct {
	UnitID   uuid.UUID `json:"unit_id"`
	Name     string    `json:"name"`
	Engine   string    `json:"engine"`
	Host     string    `json:"host"`
	Port     int       `json:"port"`
	Database string    `json:"database"`
	Username string    `json:"username"`
	SSLMode  string    `json:"ssl_mode"`
}
type CreateInput struct {
	Input
	Password string `json:"password"`
}
type PasswordInput struct {
	Password string `json:"password"`
}
type DeleteInput struct {
	Name string `json:"name"`
}
type Details struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	SSLMode  string `json:"ssl_mode"`
}
type View struct {
	ID            uuid.UUID     `json:"id"`
	Unit          *rbac.UnitRef `json:"unit,omitempty"`
	Name          string        `json:"name"`
	Engine        string        `json:"engine"`
	Status        string        `json:"status"`
	LastError     *string       `json:"last_error"`
	LastSyncedAt  *time.Time    `json:"last_synced_at"`
	Details       *Details      `json:"details,omitempty"`
	CanManage     bool          `json:"can_manage"`
	CanOperate    bool          `json:"can_operate"`
	CanReadSchema bool          `json:"can_read_schema"`
}
type List struct {
	Connections []View `json:"connections"`
}
type Result struct {
	Connection View `json:"connection"`
}
type UserRef struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
type DeleteImpact struct {
	ConnectionID    uuid.UUID `json:"connection_id"`
	Name            string    `json:"name"`
	SnapshotCount   int       `json:"snapshot_count"`
	AssignmentCount int       `json:"assignment_count"`
	Users           []UserRef `json:"users"`
}
type Column struct {
	Name                string  `json:"name"`
	Ordinal             int     `json:"ordinal"`
	DataType            string  `json:"data_type"`
	Nullable            bool    `json:"nullable"`
	DefaultExpression   *string `json:"default_expression"`
	Identity            string  `json:"identity"`
	Generated           string  `json:"generated"`
	GeneratedExpression *string `json:"generated_expression"`
	Comment             *string `json:"comment"`
}
type Reference struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}
type Constraint struct {
	Name              string     `json:"name"`
	Kind              string     `json:"kind"`
	Columns           []string   `json:"columns"`
	Definition        string     `json:"definition"`
	Reference         *Reference `json:"reference"`
	OnUpdate          string     `json:"on_update"`
	OnDelete          string     `json:"on_delete"`
	Deferrable        bool       `json:"deferrable"`
	InitiallyDeferred bool       `json:"initially_deferred"`
	Validated         bool       `json:"validated"`
	Comment           *string    `json:"comment"`
}
type Index struct {
	Name            string   `json:"name"`
	Columns         []string `json:"columns"`
	IncludedColumns []string `json:"included_columns"`
	Unique          bool     `json:"unique"`
	Primary         bool     `json:"primary"`
	Valid           bool     `json:"valid"`
	Method          string   `json:"method"`
	Definition      string   `json:"definition"`
	Predicate       *string  `json:"predicate"`
	Comment         *string  `json:"comment"`
}
type Table struct {
	Name        string       `json:"name"`
	Kind        string       `json:"kind"`
	Comment     *string      `json:"comment"`
	Columns     []Column     `json:"columns"`
	Constraints []Constraint `json:"constraints"`
	Indexes     []Index      `json:"indexes"`
}
type Schema struct {
	Name    string  `json:"name"`
	Comment *string `json:"comment"`
	Tables  []Table `json:"tables"`
}
type Content struct {
	Schemas []Schema `json:"schemas"`
}
type RowEstimate struct {
	Schema string  `json:"schema"`
	Table  string  `json:"table"`
	Rows   float64 `json:"rows"`
}
type Stats struct {
	Schemas     int           `json:"schemas"`
	Tables      int           `json:"tables"`
	Columns     int           `json:"columns"`
	Constraints int           `json:"constraints"`
	Indexes     int           `json:"indexes"`
	Estimates   []RowEstimate `json:"estimates"`
	ObservedAt  time.Time     `json:"observed_at"`
}
type SnapshotInfo struct {
	ID           uuid.UUID `json:"id"`
	ConnectionID uuid.UUID `json:"connection_id"`
	TakenAt      time.Time `json:"taken_at"`
	ContentHash  string    `json:"content_hash"`
	Stats        Stats     `json:"stats"`
}
type Snapshot struct {
	SnapshotInfo
	Content Content `json:"content"`
}
type Snapshots struct {
	Snapshots  []SnapshotInfo `json:"snapshots"`
	NextBefore *uuid.UUID     `json:"next_before,omitempty"`
}
type OperationResult struct {
	Connection      View   `json:"connection"`
	Success         bool   `json:"success"`
	Code            string `json:"code"`
	Message         string `json:"message"`
	CreatedSnapshot bool   `json:"created_snapshot"`
}
