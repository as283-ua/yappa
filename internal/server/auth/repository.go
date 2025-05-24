package auth

import (
	"context"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo interface {
	GetUserData(ctx context.Context, user string) (db.User, error)
	CreateUser(ctx context.Context, user, cert string, pubKeyExchange []byte) error
	GetUsers(ctx context.Context, page, size int, name string) ([]string, error)
}

type PgxUserRepo struct {
	Pool *pgxpool.Pool
}

var Repo UserRepo

func (r PgxUserRepo) GetUserData(ctx context.Context, user string) (db.User, error) {
	queries := db.New(r.Pool)
	return queries.GetUserData(ctx, user)
}

func (r PgxUserRepo) CreateUser(ctx context.Context, user, cert string, pubKeyExchange []byte) error {
	queries := db.New(r.Pool)
	return queries.CreateUser(ctx, db.CreateUserParams{Username: user, Certificate: cert, PubKeyExchange: pubKeyExchange})
}

func (r PgxUserRepo) GetUsers(ctx context.Context, page, size int, name string) ([]string, error) {
	queries := db.New(r.Pool)
	return queries.GetUsers(ctx, db.GetUsersParams{
		Limit:    int32(size),
		Offset:   int32(page * size),
		Username: "%" + name + "%"})
}
