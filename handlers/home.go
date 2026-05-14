package handlers

import (
	"context"
	"net/http"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/types"
)

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	chatPreviews, err := h.service.GetChatPreview(requestingUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.HomePage(requestingUser, chatPreviews)
	component.Render(context.Background(), w)
}
