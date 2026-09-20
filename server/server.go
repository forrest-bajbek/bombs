package server

import (
	"net/http"

	"github.com/forrest-bajbek/bombs/handlers"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/services"
	"github.com/forrest-bajbek/bombs/token"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	handler    *handlers.Handler
	service    *services.Service
	tokenMaker *token.JWTMaker
}

func NewServer(
	handler *handlers.Handler,
	service *services.Service,
	tokenMaker *token.JWTMaker,
) *Server {
	return &Server{
		handler:    handler,
		service:    service,
		tokenMaker: tokenMaker,
	}
}

func (s *Server) ListenAndServe(addr string) error {
	r := chi.NewRouter()

	r.Use(middlewares.Logging)
	r.Use(func(next http.Handler) http.Handler {
		return middlewares.Session(next, s.tokenMaker)
	})

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	// Auth
	// --------------------------------------------------------------------------------
	r.Get("/login", s.handler.LoginPage)
	r.Post("/login", s.handler.LogIn)
	r.Post("/logout", s.handler.LogOut)

	// User Create
	// --------------------------------------------------------------------------------
	r.Get("/user/create", s.handler.UserCreatePage)
	r.Post("/user/create", s.handler.UserCreate)

	// Authenticated routes
	// --------------------------------------------------------------------------------
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return middlewares.IsAuthenticated(s.service, s.tokenMaker, next)
		})

		// Home
		r.Get("/", s.handler.HomePage)

		// User
		// ----------------------------------------------------------------------------
		r.Group(func(r chi.Router) {
			r.Use(middlewares.IsAdmin)
			r.Get("/user/invite", s.handler.UserInvitePage)
			r.Post("/user/invite", s.handler.UserInvitePartialLink)
		})

		r.Get("/user/profile", s.handler.UserProfilePage)
		r.Get("/user/profile/delete", s.handler.UserProfileDeletePage)
		r.Post("/user/profile/delete", s.handler.UserProfileDelete)
		r.Post("/user/profile/partial/password", s.handler.UserProfilePartialPassword)

		// Chat
		// ----------------------------------------------------------------------------
		r.Get("/chat/create", s.handler.ChatCreatePage)
		r.Post("/chat/create", s.handler.ChatCreate)

		r.Get("/chat/{chat_id}", s.handler.ChatPage)
		r.Post("/chat/{chat_id}/message/create", s.handler.MessageCreate)
		r.Get("/chat/{chat_id}/message/events", s.handler.MessageEvents)

		// Chat Profile
		// ----------------------------------------------------------------------------
		r.Get("/chat/{chat_id}/profile", s.handler.ChatProfilePage)
		r.Post("/chat/{chat_id}/profile/delete", s.handler.ChatProfileDelete)

		// Edit Chat Name
		r.Get("/partial/chat/{chat_id}/name/display", s.handler.PartialChatNameDisplay)
		r.Get("/partial/chat/{chat_id}/name/form", s.handler.PartialChatNameForm)
		r.Put("/partial/chat/{chat_id}/name/form", s.handler.PartialChatNameFormSubmit)

		// Chat User Remove
		r.Delete("/partial/chat/{chat_id}/user/{user_id}/remove", s.handler.PartialChatUserRemove)
		r.Post("/partial/chat/{chat_id}/user/{user_id}/remove/undo", s.handler.PartialChatUserRemoveUndo)

		// Chat User Add
		r.Post("/partial/chat/{chat_id}/user/search", s.handler.PartialChatUserAddSearchResult)
		r.Post("/partial/chat/{chat_id}/user/{user_id}/add", s.handler.PartialChatUserAdd)
		r.Delete("/partial/chat/{chat_id}/user/{user_id}/add/undo", s.handler.PartialChatUserAddUndo)
	})

	return http.ListenAndServe(addr, r)
}
