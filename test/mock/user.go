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

func (r MockUserRepo) GetUserByUsername(ctx context.Context, user string) (db.User, error) {
	u, ok := r.users[user]
	if !ok {
		return u, pgx.ErrNoRows
	}
	return u, nil
}

func (r *MockUserRepo) CreateUser(ctx context.Context, user, cert string) error {
	_, err := r.GetUserByUsername(ctx, user)
	if err == nil {
		return errors.New("user already exists")
	}
	r.users[user] = db.User{ID: int32(r.serial), Username: user, Certificate: cert}
	r.serial++
	return nil
}

func (r *MockUserRepo) ChangeEcdhTemp(ctx context.Context, username string, ecdh []byte) error {
	user, err := r.GetUserByUsername(ctx, username)
	if err != nil {
		return errors.New("user doesn't exist")
	}

	user.EcdhTemp = ecdh
	r.users[username] = user
	return nil
}
