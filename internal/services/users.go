package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

func RestoreUser(ctx context.Context, cluster ports.ClusterProvider, user *models.User) error {
	if _, err := cluster.Request("user", user.UserID, &messages.CreateUserMessage{User: *user, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore user actor", "username", user.Username, "error", err)
		return err
	}

	return nil
}

func CreateUser(ctx context.Context, cluster ports.ClusterProvider, user *models.CreateUserRequest) (string, error) {
	userID := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	cluster.Request("user", userID, &messages.CreateUserMessage{ //nolint:errcheck // fire-and-forget
		User: models.User{
			UserID:   userID,
			Username: user.Username,
			Email:    user.Email,
			Password: string(hashedPassword),
			Gold:     constants.InitialPlayerGold,
			Food:     constants.InitialPlayerFood,
		},
		Restore: false,
	})

	return userID, nil
}
