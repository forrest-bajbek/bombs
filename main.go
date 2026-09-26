package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/forrest-bajbek/bombs/handlers"
	"github.com/forrest-bajbek/bombs/hub"
	"github.com/forrest-bajbek/bombs/server"
	"github.com/forrest-bajbek/bombs/services"
	"github.com/forrest-bajbek/bombs/sqlite"
	"github.com/forrest-bajbek/bombs/token"
	"github.com/forrest-bajbek/bombs/utils"
	_ "github.com/joho/godotenv/autoload"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var embedMigrations embed.FS

func main() {
	// Database
	BOMBS_DATA_FOLDER := os.Getenv("BOMBS_DATA_FOLDER")
	if BOMBS_DATA_FOLDER == "" {
		BOMBS_DATA_FOLDER = "/bombs/data"
	}
	if string(BOMBS_DATA_FOLDER[len(BOMBS_DATA_FOLDER)-1]) == "/" {
		BOMBS_DATA_FOLDER = string(BOMBS_DATA_FOLDER[:len(BOMBS_DATA_FOLDER)-1])
	}
	sqlite_file := fmt.Sprintf("file:%s/bombs.db", BOMBS_DATA_FOLDER)
	db, err := sql.Open("sqlite3", sqlite_file)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Setting pool to 1 connection with no expiration keeps one conneciton open
	// Required to keep in-memory tables alive.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	// SQLite WASM driver defaults to 256MB of heap for whole database.
	// Setting maxSQLiteMemory to 1GB
	const maxSQLiteMemory = 1 << 30 // 1GB
	if err := db.PingContext(sqlite3.WithMaxMemory(context.Background(), maxSQLiteMemory)); err != nil {
		panic(err)
	}

	// Database Migrations
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		panic(err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		panic(err)
	}

	// Create in-memory tables
	if err := sqlite.EnsureMemDB(db); err != nil {
		panic(err)
	}

	tokenMaker := token.NewJWTMaker()
	encrypter := utils.NewEncrypter()

	// Repository contains all database logic
	repo := sqlite.NewRepo(db, encrypter)
	err = repo.EnsureAdmin()
	if err != nil {
		panic(err)
	}

	// Storage mechanism
	service := services.NewService(repo)

	// Set up channels for existing chats
	messageHub := hub.NewMessageHub()
	go messageHub.Run()

	// Handlers
	handler := handlers.NewHandler(service, tokenMaker, messageHub)

	// Server contains routing logic
	s := server.NewServer(handler, service, tokenMaker)

	if err := s.ListenAndServe(":9000"); err != nil {
		log.Fatal(err)
	}
}
