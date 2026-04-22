package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/forrest-bajbek/bombs/components"
)

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	component := components.LoginPage()
	component.Render(context.Background(), w)
}

func (h *Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	userID, err := h.service.CheckPassword(username, password)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	authToken, _, err := h.tokenMaker.CreateAuthToken(userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	cookie := &http.Cookie{
		Name:     "authToken",
		Value:    authToken,
		Path:     "/",
		Expires:  time.Now().Add(15 * time.Minute),
		MaxAge:   int(time.Now().Add(15 * time.Minute).Unix()),
		HttpOnly: true,
		Secure:   r.TLS != nil, // Set only on HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/chat", http.StatusFound)
}

func (h *Handler) LogOut(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "authToken",
		Value:    "",
		Path:     "/",             // need this
		Expires:  time.Unix(0, 0), // legacy support
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusFound)
}
