package database

import (
	"cityio/internal/models"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var db *gorm.DB
var once sync.Once

func Nullable(v any) any {
	switch val := v.(type) {
	case string:
		if val != "" {
			return sql.NullString{String: val, Valid: true}
		} else {
			return sql.NullString{Valid: false}
		}
	case int64:
		return sql.NullInt64{Int64: val, Valid: true}
	case float64:
		return sql.NullFloat64{Float64: val, Valid: true}
	default:
		return nil
	}
}

func initDB() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found... Using environment variables instead.")
	}
	var dsn string
	if os.Getenv("ENVIRONMENT") == "production" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432",
			os.Getenv("PSQL_HOST_PROD"),
			os.Getenv("PSQL_USERNAME_PROD"),
			os.Getenv("PSQL_PASSWORD_PROD"),
			os.Getenv("PSQL_DATABASE_PROD"))
	} else {
		log.Println("Using development environment")
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432",
			os.Getenv("PSQL_HOST_DEV"),
			os.Getenv("PSQL_USERNAME_DEV"),
			os.Getenv("PSQL_PASSWORD_DEV"),
			os.Getenv("PSQL_DATABASE_DEV"))
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: 3 * time.Second,
			LogLevel:      logger.Warn,
			Colorful:      true,
		},
	)

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.Army{},
		&models.MapTile{},
		&models.City{},
		&models.Building{},
		&models.Training{},
	)
	if err != nil {
		log.Fatal("Failed to auto-migrate:", err)
	}

	psqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	psqlDB.SetMaxOpenConns(50)
	psqlDB.SetMaxIdleConns(25)
	psqlDB.SetConnMaxLifetime(0)
}

func GetDB() *gorm.DB {
	once.Do(initDB)
	return db
}
