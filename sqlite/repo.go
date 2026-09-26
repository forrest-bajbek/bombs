package sqlite

import (
	"database/sql"

	"github.com/forrest-bajbek/bombs/utils"
)

type Repo struct {
	db        *sql.DB
	encrypter *utils.Encrypter
}

func NewRepo(db *sql.DB, encrypter *utils.Encrypter) *Repo {
	return &Repo{
		db:        db,
		encrypter: encrypter,
	}
}
