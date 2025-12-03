package controllers

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"cityio/internal/constants"
	"cityio/internal/logger"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type UserController struct {
	cluster ports.ClusterProvider
	log     logger.Logger
}

func NewUserController(cl ports.ClusterProvider, l logger.Logger) *UserController {
	return &UserController{
		cluster: cl,
		log:     l,
	}
}

func (u *UserController) Restore(user *models.User) error {
	_, err := u.cluster.Request("user", user.UserID, &messages.CreateUserMessage{
		User:    *user,
		Restore: true,
	})
	if err != nil {
		u.log.Error("failed to restore user actor", "username", user.Username, "error", err)
		return err
	}

	return nil
}

func (u *UserController) Create(user *models.CreateUserRequest) (string, error) {
	userID := uuid.New().String()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	u.cluster.Request("user", userID, &messages.CreateUserMessage{
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

// func LoginUser(user models.LoginUserRequest) (models.LoginUserResponse, error) {
// 	db := database.GetDB()
// 	secretKey := []byte(os.Getenv("JWT_SECRET"))

// 	var account models.User
// 	err := db.Where("username = ?", user.Identifier).Or("email = ?", user.Identifier).First(&account).Error
// 	if err != nil {
// 		// TODO: make error message specific to login
// 		return models.LoginUserResponse{}, &messages.UserNotFoundError{UserId: user.Identifier}
// 	}

// 	if account.UserId == "" {
// 		// TODO: make error message specific to login
// 		return models.LoginUserResponse{}, &messages.UserNotFoundError{UserId: user.Identifier}
// 	}

// 	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(user.Password))
// 	if err != nil {
// 		return models.LoginUserResponse{}, &messages.InvalidPasswordError{Identifier: user.Identifier}
// 	}

// 	claims := jwt.MapClaims{
// 		"username": account.Username,
// 		"email":    account.Email,
// 		"userId":   account.UserId,
// 		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // expires in a week
// 		"iat":      time.Now().Unix(),
// 	}
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

// 	signedToken, err := token.SignedString(secretKey)
// 	if err != nil {
// 		return models.LoginUserResponse{}, err
// 	}

// 	var capital models.City
// 	err = db.Where("owner = ?", account.UserId).First(&capital).Error
// 	if err != nil {
// 		return models.LoginUserResponse{}, err
// 	}

// 	return models.LoginUserResponse{
// 		Token:    signedToken,
// 		UserId:   account.UserId,
// 		Username: account.Username,
// 		Email:    account.Email,
// 		Capital:  &capital,
// 	}, nil
// }

// func ValidateToken(tokenString string) (models.UserClaims, *models.City, error) {
// 	secretKey := []byte(os.Getenv("JWT_SECRET"))
// 	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
// 		return secretKey, nil
// 	})
// 	if err != nil {
// 		return models.UserClaims{}, nil, err
// 	}

// 	claims, ok := token.Claims.(jwt.MapClaims)
// 	if !ok || !token.Valid {
// 		return models.UserClaims{}, nil, &messages.InvalidTokenError{}
// 	}

// 	var capital models.City
// 	err = db.Where("owner = ?", claims["userId"]).First(&capital).Error
// 	if err != nil {
// 		return models.UserClaims{}, nil, err
// 	}

// 	return models.UserClaims{
// 		Username: claims["username"].(string),
// 		Email:    claims["email"].(string),
// 		UserId:   claims["userId"].(string),
// 	}, &capital, nil
// }

// func GetUser(userId string) (models.User, error) {
// 	response, err := actors.Request[messages.GetUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetUserPIDMessage{
// 		UserId: userId,
// 	})
// 	if err != nil {
// 		return models.User{}, err
// 	}
// 	if response.PID == nil {
// 		return models.User{}, &messages.UserNotFoundError{UserId: userId}
// 	}

// 	var userResponse *messages.GetUserResponseMessage
// 	userResponse, err = actors.Request[messages.GetUserResponseMessage](system.Root, response.PID, messages.GetUserMessage{})

// 	return userResponse.User, nil
// }

// func GetUserAccount(userId string) (models.UserAccountOutput, error) {
// 	user, err := GetUser(userId)
// 	if err != nil {
// 		return models.UserAccountOutput{}, err
// 	}

// 	return models.UserAccountOutput{
// 		Username: user.Username,
// 		Gold:     user.Gold,
// 		Food:     user.Food,
// 		Allies:   user.Allies,
// 	}, nil
// }

// func DeleteUser(userId string) error {
// 	response, err := actors.Request[messages.GetUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetUserPIDMessage{
// 		UserId: userId,
// 	})
// 	if err != nil {
// 		log.Printf("Error getting user pid: %s", err)
// 		return err
// 	}
// 	if response.PID == nil {
// 		return &messages.UserNotFoundError{UserId: userId}
// 	}

// 	var deleteResponse *messages.DeleteUserResponseMessage
// 	deleteResponse, err = actors.Request[messages.DeleteUserResponseMessage](system.Root, response.PID, messages.DeleteUserMessage{})
// 	if err != nil {
// 		log.Printf("Error deleting user: %s", err)
// 		return err
// 	}
// 	if deleteResponse.Error != nil {
// 		log.Printf("Error deleting user: %s", deleteResponse.Error)
// 		return deleteResponse.Error
// 	}

// 	var removeResponse *messages.DeleteUserPIDResponseMessage
// 	removeResponse, err = actors.Request[messages.DeleteUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.DeleteUserPIDMessage{
// 		UserId: userId,
// 	})

// 	if err != nil {
// 		log.Printf("Error removing user pid: %s", err)
// 		return err
// 	}
// 	if removeResponse.Error != nil {
// 		log.Printf("Error removing user pid: %s", removeResponse.Error)
// 		return removeResponse.Error
// 	}

// 	return nil
// }
