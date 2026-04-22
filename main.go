package main

import (
	"database/sql"
	"embed"
	"log"

	"github.com/forrest-bajbek/bombs/server"
	"github.com/forrest-bajbek/bombs/sqlite"
	"github.com/forrest-bajbek/bombs/user"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var embedMigrations embed.FS

func main() {

	// Database
	MainDb, err := sql.Open("sqlite3", "file:user.db")
	if err != nil {
		log.Fatal(err)
	}
	defer MainDb.Close()

	// Database Migrations
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		panic(err)
	}
	if err := goose.Up(MainDb, "migrations/MainDb"); err != nil {
		panic(err)
	}

	rUser := sqlite.NewUserRepository(MainDb)
	sUser := user.NewService(rUser)
	svr := server.NewServer(sUser)
	if err := svr.ListenAndServe(":9000"); err != nil {
		log.Fatal(err)
	}

}
