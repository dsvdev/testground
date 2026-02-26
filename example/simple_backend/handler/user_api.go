package handler

import (
	"context"
	"github.com/dsvdev/testground/example/simple_backend/model"
	"github.com/dsvdev/testground/example/simple_backend/repository"
)

type UserHandler struct {
	userRepo *repository.UserRepo
}

func NewUserHandler(userRepo *repository.UserRepo) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) GetUserById(ctx context.Context, userID int64) (*model.User, error) {
	return h.userRepo.GetUserByID(ctx, userID)
}

func (h *UserHandler) AddNewUser(ctx context.Context, name string) (*model.User, error) {
	return h.userRepo.NewUser(ctx, name)
}
