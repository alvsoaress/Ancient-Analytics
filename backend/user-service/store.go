package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID        string    `json:"id"`
	SteamID   string    `json:"steamId"`
	CreatedAt time.Time `json:"createdAt"`
	LastLogin time.Time `json:"lastLoginAt"`
}

type UserStore interface {
	UpsertSteamUser(context.Context, string) (User, error)
	GetBySteamID(context.Context, string) (User, error)
	Ping(context.Context) error
}

type PostgresUserStore struct{ pool *pgxpool.Pool }

func OpenUserStore(ctx context.Context, databaseURL string) (*PostgresUserStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresUserStore{pool: pool}, nil
}

func (s *PostgresUserStore) Close()                         { s.pool.Close() }
func (s *PostgresUserStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *PostgresUserStore) UpsertSteamUser(ctx context.Context, steamID string) (User, error) {
	const query = `
		INSERT INTO users (steam_id) VALUES ($1)
		ON CONFLICT (steam_id) DO UPDATE SET updated_at = now(), last_login_at = now()
		RETURNING id::text, steam_id, created_at, last_login_at`
	return scanUser(s.pool.QueryRow(ctx, query, steamID))
}

func (s *PostgresUserStore) GetBySteamID(ctx context.Context, steamID string) (User, error) {
	const query = `SELECT id::text, steam_id, created_at, last_login_at FROM users WHERE steam_id = $1`
	user, err := scanUser(s.pool.QueryRow(ctx, query, steamID))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

type rowScanner interface{ Scan(...any) error }

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.SteamID, &user.CreatedAt, &user.LastLogin)
	return user, err
}
