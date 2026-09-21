package server

import (
	"net/http"

	"github.com/forrest-bajbek/bombs/handlers"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/routes"
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
	routes.Register()

	r := chi.NewRouter()

	r.Use(middlewares.Logging)
	r.Use(func(next http.Handler) http.Handler {
		return middlewares.Session(next, s.tokenMaker)
	})

	// Health
	r.Get(routes.Pattern(routes.Health), func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	// Auth
	// --------------------------------------------------------------------------------
	r.Get(routes.Pattern(routes.LoginPage), s.handler.LoginPage)
	r.Post(routes.Pattern(routes.Login), s.handler.LogIn)
	r.Post(routes.Pattern(routes.Logout), s.handler.LogOut)

	// User Create
	// --------------------------------------------------------------------------------
	r.Get(routes.Pattern(routes.UserCreatePage), s.handler.UserCreatePage)
	r.Post(routes.Pattern(routes.UserCreate), s.handler.UserCreate)

	// Authenticated routes
	// --------------------------------------------------------------------------------
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return middlewares.IsAuthenticated(s.service, s.tokenMaker, next)
		})

		// Home
		r.Get(routes.Pattern(routes.Home), s.handler.HomePage)

		// User
		// ----------------------------------------------------------------------------
		r.Group(func(r chi.Router) {
			r.Use(middlewares.IsAdmin)
			r.Get(routes.Pattern(routes.UserInvitePage), s.handler.UserInvitePage)
			r.Post(routes.Pattern(routes.PartialUserInvite), s.handler.UserInvitePartialLink)
		})

		r.Get(routes.Pattern(routes.UserProfilePage), s.handler.UserProfilePage)
		r.Get(routes.Pattern(routes.UserProfileDeletePage), s.handler.UserProfileDeletePage)
		r.Post(routes.Pattern(routes.UserProfileDelete), s.handler.UserProfileDelete)
		r.Post(routes.Pattern(routes.PartialUserProfilePassword), s.handler.UserProfilePartialPassword)

		// Chat
		// ----------------------------------------------------------------------------
		r.Get(routes.Pattern(routes.ChatCreatePage), s.handler.ChatCreatePage)
		r.Post(routes.Pattern(routes.ChatCreate), s.handler.ChatCreate)

		r.Get(routes.Pattern(routes.ChatPage), s.handler.ChatPage)
		r.Post(routes.Pattern(routes.ChatMessageCreate), s.handler.MessageCreate)
		r.Get(routes.Pattern(routes.ChatMessageEvents), s.handler.MessageEvents)
		r.Get(routes.Pattern(routes.ChatFile), s.handler.ChatFile)

		// Attachments
		r.Get(routes.Pattern(routes.PartialChatMessageFiles), s.handler.PartialChatMessageFiles)
		r.Get(routes.Pattern(routes.PartialChatModalClose), s.handler.PartialChatModalClose)

		// Chat Profile
		// ----------------------------------------------------------------------------
		r.Get(routes.Pattern(routes.ChatProfilePage), s.handler.ChatProfilePage)
		r.Post(routes.Pattern(routes.ChatProfileDelete), s.handler.ChatProfileDelete)

		// Edit Chat Name
		r.Get(routes.Pattern(routes.PartialChatNameDisplay), s.handler.PartialChatNameDisplay)
		r.Get(routes.Pattern(routes.PartialChatNameForm), s.handler.PartialChatNameForm)
		r.Put(routes.Pattern(routes.PartialChatNameFormSubmit), s.handler.PartialChatNameFormSubmit)

		// Chat User Remove
		r.Delete(routes.Pattern(routes.PartialChatUserRemove), s.handler.PartialChatUserRemove)
		r.Post(routes.Pattern(routes.PartialChatUserRemoveUndo), s.handler.PartialChatUserRemoveUndo)

		// Chat User Add
		r.Post(routes.Pattern(routes.PartialChatUserSearch), s.handler.PartialChatUserAddSearchResult)
		r.Post(routes.Pattern(routes.PartialChatUserAdd), s.handler.PartialChatUserAdd)
		r.Delete(routes.Pattern(routes.PartialChatUserAddUndo), s.handler.PartialChatUserAddUndo)
	})

	return http.ListenAndServe(addr, r)
}
