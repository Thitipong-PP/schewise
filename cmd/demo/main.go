// SPDX-License-Identifier: Apache-2.0
// Command demo registers the explicitly requested synthetic target using domain services.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Thitipong-PP/schewise/internal/config"
	"github.com/Thitipong-PP/schewise/internal/connections"
	"github.com/Thitipong-PP/schewise/internal/database"
	"github.com/Thitipong-PP/schewise/internal/rbac"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("application database unavailable")
	}
	defer func() { _ = db.Close() }()
	// Schema changes remain explicit: initialize the application before this command.
	profiles, err := connections.New(ctx, db, zap.NewNop(), connections.Options{SecretKey: cfg.SecretKey, StatementTimeout: cfg.StatementTimeout, HostAllow: cfg.HostAllow, HostDeny: cfg.HostDeny})
	if err != nil {
		return err
	}
	var actor uuid.UUID
	if err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email)=lower($1) AND is_active`, cfg.AdminEmail).Scan(&actor); err != nil {
		return errors.New("initialize the application and bootstrap the configured administrator first")
	}
	repo := rbac.NewRepository(db, zap.NewNop(), nil)
	allowed, err := repo.Engine().Can(ctx, actor, rbac.ConnectionManage, rbac.RootScope())
	if err != nil || !allowed {
		return errors.New("the configured installation user needs connection:manage at the root")
	}
	units, err := repo.VisibleUnits(ctx, actor)
	if err != nil {
		return errors.New("cannot list demo units")
	}
	var unit uuid.UUID
	for _, u := range units {
		if u.Name == "Demo" && u.ParentID != nil && *u.ParentID == rbac.RootScope().ID {
			unit = u.ID
			break
		}
	}
	if unit == uuid.Nil {
		created, err := repo.CreateUnit(ctx, actor, rbac.RootScope().ID, rbac.UnitInput{Name: "Demo", Description: "Synthetic PostgreSQL fixtures", Kind: "team"})
		if err != nil {
			return errors.New("cannot create Demo unit")
		}
		unit = created.ID
	}
	existing, err := profiles.List(ctx, actor, unit)
	if err != nil {
		return err
	}
	for _, c := range existing.Connections {
		if c.Name == "Commerce demo" {
			fmt.Println("Commerce demo is already registered. Use Connections to test or sync it.")
			return nil
		}
	}
	host := os.Getenv("SCHEWISE_DEMO_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := 5433
	if text := os.Getenv("SCHEWISE_DEMO_PORT"); text != "" {
		port, err = strconv.Atoi(text)
		if err != nil {
			return errors.New("SCHEWISE_DEMO_PORT must be an integer")
		}
	}
	password := os.Getenv("DEMO_DATABASE_PASSWORD")
	if password == "" {
		password = "synthetic-demo-reader-password"
	}
	created, err := profiles.Create(ctx, actor, connections.CreateInput{Input: connections.Input{UnitID: unit, Name: "Commerce demo", Engine: "postgres", Host: host, Port: port, Database: "schewise_demo", Username: "schewise_reader", SSLMode: "disable"}, Password: password})
	if err != nil {
		return err
	}
	synced, err := profiles.Run(ctx, actor, created.Connection.ID, true)
	if err != nil {
		return err
	}
	if !synced.Success {
		return errors.New(synced.Message)
	}
	fmt.Println("Registered and synced Commerce demo in the Demo unit.")
	return nil
}
