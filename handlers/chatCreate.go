package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/types"
)

func (h *Handler) ChatCreatePage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
		return
	}

	component := components.ChatCreatePage(requestingUser, "")
	component.Render(context.Background(), w)
}

func (h *Handler) ChatCreate(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		component := components.ChatCreatePage(requestingUser, err.Error())
		component.Render(context.Background(), w)
		return
	}

	chatName := r.PostForm.Get("chatName")
	chatID, err := h.service.CreateChat(requestingUser.ID, chatName)
	if err != nil {
		component := components.ChatCreatePage(requestingUser, err.Error())
		component.Render(context.Background(), w)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/chat/%d", chatID), http.StatusFound)
}
