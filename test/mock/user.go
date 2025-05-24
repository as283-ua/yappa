package mock

import (
	"context"
	"errors"

	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/jackc/pgx/v5"
)

type MockUserRepo struct {
	users  map[string]db.User
	serial int
}

func EmptyMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users:  map[string]db.User{},
		serial: 0,
	}
}

func (r MockUserRepo) GetUserData(ctx context.Context, user string) (db.User, error) {
	u, ok := r.users[user]
	if !ok {
		return u, pgx.ErrNoRows
	}
	return u, nil
}

func (r *MockUserRepo) CreateUser(ctx context.Context, user, cert string, pubKeyExchange []byte) error {
	_, err := r.GetUserData(ctx, user)
	if err == nil {
		return errors.New("user already exists")
	}
	r.users[user] = db.User{ID: int32(r.serial), Username: user, Certificate: cert, PubKeyExchange: pubKeyExchange}
	r.serial++
	return nil
}

func (r *MockUserRepo) GetUsers(ctx context.Context, page, size int, name string) ([]string, error) {
	initial := page * size
	final := page*size + size
	if final >= len(r.users) {
		final = len(r.users) - 1
	}
	res := make([]string, final-initial)
	for username, _ := range r.users {
		res = append(res, username)
	}
	return res, nil
}
