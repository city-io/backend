package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"log"
	"os"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func RestoreUser(user models.User) error {
	log.Printf("Restoring user: %s", user.Username)
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewUserActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	system.Root.Send(newPID, messages.RegisterUserMessage{
		User:    user,
		Restore: true,
	})
	// TODO: add confirmation message
	state.AddUserPID(user.UserId, newPID)
	return nil
}

func RegisterUser(user models.RegisterUserRequest) (string, error) {
	userId := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewUserActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	future := system.Root.RequestFuture(newPID, messages.RegisterUserMessage{
		User: models.User{
			UserId:   userId,
			Username: user.Username,
			Email:    user.Email,
			Password: string(hashedPassword),
		},
		Restore: false,
	}, time.Second*2)

	response, err := future.Result()
	if err != nil {
		return "", err
	}

	if response, ok := response.(messages.RegisterUserResponseMessage); ok {
		if response.Error != nil {
			return "", response.Error
		}
	} else {
		return "", &messages.InternalError{}
	}

	state.AddUserPID(userId, newPID)
	return userId, nil
}

func LoginUser(user models.LoginUserRequest) (models.LoginUserResponse, error) {
	db := database.GetDb()
	secretKey := []byte(os.Getenv("JWT_SECRET"))

	var account models.User
	err := db.Where("username = ?", user.Identifier).Or("email = ?", user.Identifier).First(&account).Error
	if err != nil {
		// TODO: make error message specific to login
		return models.LoginUserResponse{}, &messages.UserNotFoundError{UserId: user.Identifier}
	}

	if account.UserId == "" {
		// TODO: make error message specific to login
		return models.LoginUserResponse{}, &messages.UserNotFoundError{UserId: user.Identifier}
	}

	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(user.Password))
	if err != nil {
		return models.LoginUserResponse{}, &messages.InvalidPasswordError{Identifier: user.Identifier}
	}

	claims := jwt.MapClaims{
		"username": account.Username,
		"email":    account.Email,
		"userId":   account.UserId,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // expires in a week
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		return models.LoginUserResponse{}, err
	}

	return models.LoginUserResponse{
		Token:    signedToken,
		UserId:   account.UserId,
		Username: account.Username,
		Email:    account.Email,
	}, nil
}

func ValidateToken(tokenString string) (models.UserClaims, error) {
	secretKey := []byte(os.Getenv("JWT_SECRET"))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return models.UserClaims{}, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return models.UserClaims{}, &messages.InvalidTokenError{}
	}

	return models.UserClaims{
		Username: claims["username"].(string),
		Email:    claims["email"].(string),
		UserId:   claims["userId"].(string),
	}, nil
}

func GetUser(userId string) (models.User, error) {
	userPID, exists := state.GetUserPID(userId)
	if !exists {
		return models.User{}, &messages.UserNotFoundError{UserId: userId}
	}

	future := system.Root.RequestFuture(userPID, messages.GetUserMessage{}, time.Second*2)
	result, err := future.Result()
	if err != nil {
		return models.User{}, err
	}

	response, ok := result.(messages.GetUserResponseMessage)
	if !ok {
		return models.User{}, &messages.UserNotFoundError{UserId: userId}
	}

	return response.User, nil
}
