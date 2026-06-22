package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

func RestoreUser(ctx context.Context, cluster ports.ClusterProvider, user *domain.User) error {
	if _, err := cluster.Request("user", user.UserID, &messages.CreateUserMessage{User: *user, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore user actor", "username", user.Username, "error", err)
		return err
	}

	return nil
}

func CreateUser(ctx context.Context, cluster ports.ClusterProvider, user *CreateUserRequest) (string, error) {
	userID := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	res, err := cluster.Request("user", userID, &messages.CreateUserMessage{
		User: domain.User{
			UserID:   userID,
			Username: user.Username,
			Email:    user.Email,
			Password: string(hashedPassword),
			Gold:     constants.InitialPlayerGold,
			Food:     constants.InitialPlayerFood,
		},
		Restore: false,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create user actor", "user_id", userID, "error", err)
		return "", err
	}
	if _, ok := res.(messages.Ack); !ok {
		// The actor either failed to persist (UserCreationError) or returned
		// some unexpected response. Surface as a generic failure to the caller.
		return "", fmt.Errorf("user actor refused create: %T", res)
	}
	return userID, nil
}
