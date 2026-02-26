package repository

import (
	"context"
	"github.com/dsvdev/testground/example/simple_backend/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(ctx context.Context, connStr string) (*UserRepo, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return &UserRepo{
		pool: pool,
	}, nil
}

func (r *UserRepo) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	rows, err := r.pool.Query(ctx, "SELECT * FROM users WHERE id = @userId", pgx.NamedArgs{
		"userId": userID,
	})

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var user model.User
	if err := rows.Scan(&user.ID, &user.Name); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) NewUser(ctx context.Context, name string) (*model.User, error) {
	row, err := r.pool.Query(ctx, "INSERT INTO users (name) VALUES (@name) returning id", pgx.NamedArgs{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	defer row.Close()
	if !row.Next() {
		return nil, nil
	}
	var user model.User
	if err := row.Scan(&user.ID); err != nil {
		return nil, err
	}
	user.Name = name
	return &user, nil
}
