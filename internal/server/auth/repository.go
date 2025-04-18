package auth

import (
	"context"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo interface {
	GetUserByUsername(ctx context.Context, user string) (db.User, error)
	CreateUser(ctx context.Context, user, cert string) error
	ChangeEcdhTemp(ctx context.Context, user string, ecdh []byte) error
}

type PgxUserRepo struct {
	Pool *pgxpool.Pool
}

var Repo UserRepo

func (r PgxUserRepo) GetUserByUsername(ctx context.Context, user string) (db.User, error) {
	queries := db.New(r.Pool)
	return queries.GetUserByUsername(ctx, user)
}

func (r PgxUserRepo) CreateUser(ctx context.Context, user, cert string) error {
	queries := db.New(r.Pool)
	return queries.CreateUser(ctx, db.CreateUserParams{Username: user, Certificate: cert})
}

func (r PgxUserRepo) ChangeEcdhTemp(ctx context.Context, user string, ecdh []byte) error {
	queries := db.New(r.Pool)
	return queries.ChangeEcdhTemp(ctx, db.ChangeEcdhTempParams{Username: user, EcdhTemp: ecdh})
}
