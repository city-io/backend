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

func RestoreUser(user models.User) {
	log.Printf("Restoring user: %s", user.UserId)
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewUserActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	system.Root.Send(newPID, messages.RegisterUserMessage{
		User:    user,
		Restore: true,
	})
	state.AddUserPID(user.UserId, newPID)
}

func RegisterUser(user models.UserInput) (string, error) {
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

func LoginUser(user models.UserInput) (string, error) {
	db := database.GetDb()
	secretKey := []byte(os.Getenv("JWT_SECRET"))

	var account models.User
	var identifier string
	if user.Email != "" {
		identifier = user.Email
		db.Find(&account, "email = ?", identifier)
	} else {
		identifier = user.Username
		db.Find(&account, "username = ?", identifier)
	}

	if account.UserId == "" {
		return "", &messages.UserNotFoundError{UserId: user.Email}
	}

	err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(user.Password))
	if err != nil {
		return "", &messages.InvalidPasswordError{Identifier: identifier}
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
		return "", err
	}

	return signedToken, nil
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
