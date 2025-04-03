package test

import (
	"context"
	"testing"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	uri := "postgres://yappa:pass@localhost:5432/yappa-chat"
	pool, err := pgxpool.New(ctx, uri)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	return pool
}

func TestCreateUserDb(t *testing.T) {
	t.Skip("Skipping db test")
	ctx := context.Background()
	pool := setupDB(t, ctx)
	defer pool.Close()

	queries := db.New(pool)

	username := "testuser"
	if err := queries.CreateUser(ctx, db.CreateUserParams{Username: username, Certificate: "test"}); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Log("User created successfully")

	user, err := queries.GetUserByUsername(ctx, username)
	if err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}

	t.Log(user)
}
