package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/000001_users.sql
var usersMigration string

var connectionURI = regexp.MustCompile(`postgres(?:ql)?://[^[:space:]'"]+`)

func redactConnectionError(message string) string {
	return connectionURI.ReplaceAllString(message, "postgresql://[redacted]")
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Printf("database connection failed: %s", redactConnectionError(err.Error()))
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		panic("cannot initialize migration metadata")
	}
	var applied bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, "000001_users").Scan(&applied); err != nil {
		panic("cannot read migration metadata")
	}
	if applied {
		fmt.Println("database is up to date")
		return
	}
	if _, err := conn.Exec(ctx, usersMigration); err != nil {
		panic("migration failed")
	}
	if _, err := conn.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, "000001_users"); err != nil {
		panic("cannot record migration")
	}
	fmt.Println("migration 000001_users applied")
}
