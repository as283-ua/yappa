package user

import (
	"context"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo interface {
	GetUsers(page, size int, name string) ([]string, error)
	GetUserData(username string) (db.GetUserDataRow, error)
}

type PgxUserRepo struct {
	Pool *pgxpool.Pool
	Ctx  context.Context
}

var Repo UserRepo

func (r PgxUserRepo) GetUsers(page, size int, name string) ([]string, error) {
	queries := db.New(r.Pool)
	return queries.GetUsers(r.Ctx, db.GetUsersParams{
		Limit:    int32(size),
		Offset:   int32(page * size),
		Username: "%" + name + "%"})
}

func (r PgxUserRepo) GetUserData(username string) (db.GetUserDataRow, error) {
	queries := db.New(r.Pool)
	return queries.GetUserData(r.Ctx, username)
}
