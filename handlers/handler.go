package handlers

import (
	"github.com/forrest-bajbek/bombs/hub"
	"github.com/forrest-bajbek/bombs/services"
	"github.com/forrest-bajbek/bombs/token"
)

type Handler struct {
	service    *services.Service
	tokenMaker *token.JWTMaker
	messageHub *hub.MessageHub
}

func NewHandler(
	service *services.Service,
	tokenMaker *token.JWTMaker,
	messageHub *hub.MessageHub,
) *Handler {
	return &Handler{
		service:    service,
		tokenMaker: tokenMaker,
		messageHub: messageHub,
	}
}
