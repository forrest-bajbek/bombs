package main

import (
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

	// The message table lives in an attached in-memory database, which only
	// persists for as long as at least one connection keeps it attached.
	// Pinning the pool to a single, never-recycled connection guarantees
	// that connection - and therefore the in-memory database - is never
	// closed.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	// The in-memory database above doesn't survive a process restart, so it
	// must be (re)created unconditionally on every boot, on this same
	// pinned connection - see EnsureMemDB for why this can't be a normal
	// goose migration.
	if err := sqlite.EnsureMemDB(db); err != nil {
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

	tokenMaker := token.NewJWTMaker()
	encrypter := utils.NewEncrypter()

	// Repository contains all database logic
	repo := sqlite.NewRepo(db, encrypter)
	err = repo.EnsureAdmin()
	if err != nil {
		panic(err)
	}

	// Service is an "interface", or abstraction on top of the repository.
	// This allows you to implement multiple storage mechanisms and swap them interchangeably.
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
