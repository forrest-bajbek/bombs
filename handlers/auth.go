package handlers

import (
	"context"
	"net/http"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/routes"
)

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	component := components.LoginPage("", "", "", routes.SafeNext(r.URL.Query().Get("next")))
	component.Render(context.Background(), w)
}

func (h *Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		component := components.LoginPage("", "", err.Error(), "")
		component.Render(context.Background(), w)
		return
	}

	next := routes.SafeNext(r.PostForm.Get("next"))
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	userID, err := h.service.CheckPassword(username, password)
	if err != nil {
		component := components.LoginPage(username, password, err.Error(), next)
		component.Render(context.Background(), w)
		return
	}

	authToken, _, err := h.tokenMaker.CreateAuthToken(userID)
	if err != nil {
		component := components.LoginPage("", "", err.Error(), next)
		component.Render(context.Background(), w)
		return
	}

	middlewares.SetAuthCookies(w, authToken)
	if next == "" {
		next = routes.URL(routes.Home)
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func (h *Handler) LogOut(w http.ResponseWriter, r *http.Request) {
	middlewares.ClearAuthCookies(w)
	http.Redirect(w, r, routes.URL(routes.LoginPage), http.StatusFound)
}
