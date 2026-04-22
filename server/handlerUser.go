package server

import (
	"context"
	"net/http"
	"text/template"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/user"
)

type UserHandler struct {
	service *user.Service
}

// Test Pages
// ------------------------------------------------------------------------------------
func (h *UserHandler) testPageTemplate(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/test.page.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	t.Execute(w, nil)
}

func (h *UserHandler) testPageTempl(w http.ResponseWriter, r *http.Request) {
	component := components.TestPage()
	component.Render(context.Background(), w)
}

// Actual Routes
// ------------------------------------------------------------------------------------
func (h *UserHandler) createUserPage(w http.ResponseWriter, r *http.Request) {
	component := components.UserCreatePage()
	component.Render(context.Background(), w)
}

func (h *UserHandler) logInPage(w http.ResponseWriter, r *http.Request) {
	component := components.LoginPage()
	component.Render(context.Background(), w)
}

func (h *UserHandler) logIn(w http.ResponseWriter, r *http.Request) {
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

	ctx := context.WithValue(r.Context(), middlewares.AuthUserID, userID)
	req := r.WithContext(ctx)
	http.Redirect(w, req, "/sessions", http.StatusFound)
}
