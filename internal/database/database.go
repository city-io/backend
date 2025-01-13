package database

import (
	"cityio/internal/models"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	// "gorm.io/gorm/logger"

	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	// "time"
)

var db *gorm.DB
var once sync.Once

func Nullable(v interface{}) interface{} {
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

func initDb() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found... Using environment variables instead.")
	}
	var dsn string
	if os.Getenv("ENVIRONMENT") == "production" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432",
			os.Getenv("PSQL_HOST"),
			os.Getenv("PSQL_USERNAME"),
			os.Getenv("PSQL_PASSWORD"),
			os.Getenv("PSQL_DATABASE"))
	} else {
		log.Println("Using development environment")
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432",
			os.Getenv("PSQL_HOST"),
			os.Getenv("PSQL_USERNAME"),
			os.Getenv("PSQL_PASSWORD"),
			os.Getenv("PSQL_DATABASE"))
	}

	// gormLogger := logger.New(
	// 	log.New(os.Stdout, "\r\n", log.LstdFlags),
	// 	logger.Config{
	// 		SlowThreshold: time.Second,
	// 		LogLevel:      logger.Info,
	// 		Colorful:      true,
	// 	},
	// )

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Logger: gormLogger,
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
	)
	if err != nil {
		log.Fatal("Failed to auto-migrate:", err)
	}

	psqlDb, err := db.DB()
	if err != nil {
		panic(err)
	}

	psqlDb.SetMaxOpenConns(10)
	psqlDb.SetMaxIdleConns(5)
	psqlDb.SetConnMaxLifetime(0)
}

func GetDb() *gorm.DB {
	once.Do(initDb)
	return db
}
