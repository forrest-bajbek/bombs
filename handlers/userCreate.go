package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/types"
)

func (h *Handler) UserInvitePage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", 400)
	}

	component := components.UserInvitePage(requestingUser)
	component.Render(context.Background(), w)
}

func (h *Handler) UserInvitePartialLink(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", 400)
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	username := r.PostForm.Get("username")

	if username == "" {
		component := components.PartialUserInviteUserEmpty()
		component.Render(context.Background(), w)
		return
	}

	user_exists, err := h.service.UserExists(username)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if user_exists {
		component := components.PartialUserInviteUserAlreadyExists(username)
		component.Render(context.Background(), w)
		return
	}

	inviteToken, _, err := h.tokenMaker.CreateInviteToken(username)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	link := fmt.Sprintf("http://localhost:9000/user/create?inviteToken=%s", inviteToken)
	component := components.PartialUserInviteLink(link)
	component.Render(context.Background(), w)
}

func (h *Handler) UserCreatePage(w http.ResponseWriter, r *http.Request) {

	errInvalidInviteLink := errors.New("Invalid Invite Token")

	// Retrieve, validate, and parse invite token
	params := r.URL.Query()
	inviteToken := params.Get("inviteToken")
	inviteClaims, err := h.tokenMaker.ValidateInviteToken(inviteToken)
	if err != nil {
		http.Error(w, errInvalidInviteLink.Error(), 400)
		return
	}
	username := inviteClaims.Username

	// Make sure username doesn't already exist
	user_exists, err := h.service.UserExists(username)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if user_exists {
		http.Error(w, errInvalidInviteLink.Error(), 400)
		return
	}

	// Render Page
	component := components.UserCreatePage(username, inviteToken)
	component.Render(context.Background(), w)
}

func (h *Handler) UserCreate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	inviteToken := r.PostForm.Get("inviteToken")
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	inviteClaims, err := h.tokenMaker.ValidateInviteToken(inviteToken)
	if err != nil {
		errInvalidInviteLink := errors.New("Invalid Invite Token")
		http.Error(w, errInvalidInviteLink.Error(), 400)
		return
	}
	if username != inviteClaims.Username {
		errInvalidInviteLink := errors.New("Invalid Invite Token")
		http.Error(w, errInvalidInviteLink.Error(), 400)
		return
	}

	_, err = h.service.CreateUser(username, password)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	cookie := &http.Cookie{
		Name:   "authToken",
		Value:  "",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusFound)
}
