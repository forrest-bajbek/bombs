package server

import (
	"net/http"

	"github.com/forrest-bajbek/bombs/user"
)

type Server struct {
	userHandler *UserHandler
}

func NewServer(us *user.Service) *Server {
	return &Server{
		userHandler: &UserHandler{service: us},
	}
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test/template", s.userHandler.testPageTemplate)
	mux.HandleFunc("GET /test/templ", s.userHandler.testPageTempl)

	mux.HandleFunc("GET /user/create", s.userHandler.createUserPage)
	mux.HandleFunc("GET /login", s.userHandler.logInPage)
	mux.HandleFunc("POST /login", s.userHandler.logIn)

	return http.ListenAndServe(addr, mux)
}
