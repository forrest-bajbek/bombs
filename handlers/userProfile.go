package handlers

import (
	"context"
	"net/http"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/types"
)

func (h *Handler) UserProfilePage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", 400)
	}

	component := components.UserProfilePage(requestingUser)
	component.Render(context.Background(), w)
}

func (h *Handler) UserProfileDeletePage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", 400)
	}
	component := components.UserProfileDeletePage(requestingUser)
	component.Render(context.Background(), w)
}

func (h *Handler) UserProfileDelete(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", 400)
	}

	err := h.service.DeleteMessageByUserID(requestingUser.ID)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	err = h.service.DeleteChatUserByUserID(requestingUser.ID)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	err = h.service.DeleteUser(requestingUser.ID)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *Handler) UserProfilePartialPassword(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", 400)
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	old_password := r.PostForm.Get("old_password")
	new_password := r.PostForm.Get("new_password")

	if old_password == new_password {
		component := components.PartialUserProfilePasswordFailure("Your new password must be different from your old password")
		component.Render(context.Background(), w)
		return
	}

	err = h.service.ChangePassword(requestingUser.Username, old_password, new_password)
	if err != nil {
		component := components.PartialUserProfilePasswordFailure(err.Error())
		component.Render(context.Background(), w)
		return
	}

	component := components.PartialUserProfilePasswordSuccess()
	component.Render(context.Background(), w)

}
