package router

import (
	"net/http"

	"github.com/forrest-bajbek/bombs/handlers"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/routes"
	"github.com/forrest-bajbek/bombs/services"
	"github.com/forrest-bajbek/bombs/token"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	handler    *handlers.Handler
	service    *services.Service
	tokenMaker *token.JWTMaker
}

func NewRouter(
	handler *handlers.Handler,
	service *services.Service,
	tokenMaker *token.JWTMaker,
) *Router {
	return &Router{
		handler:    handler,
		service:    service,
		tokenMaker: tokenMaker,
	}
}

func (s *Router) Mux() http.Handler {
	routes.Register()

	mux := chi.NewRouter()

	mux.Use(middlewares.Logging)
	mux.Use(func(next http.Handler) http.Handler {
		return middlewares.Session(next, s.tokenMaker)
	})

	// Health
	mux.Get(routes.Pattern(routes.Health), func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	// Auth
	// --------------------------------------------------------------------------------
	mux.Get(routes.Pattern(routes.LoginPage), s.handler.LoginPage)
	mux.Post(routes.Pattern(routes.Login), s.handler.LogIn)
	mux.Post(routes.Pattern(routes.Logout), s.handler.LogOut)

	// User Create
	// --------------------------------------------------------------------------------
	mux.Get(routes.Pattern(routes.UserCreatePage), s.handler.UserCreatePage)
	mux.Post(routes.Pattern(routes.UserCreate), s.handler.UserCreate)

	// Authenticated routes
	// --------------------------------------------------------------------------------
	mux.Group(func(mux chi.Router) {
		mux.Use(func(next http.Handler) http.Handler {
			return middlewares.IsAuthenticated(s.service, s.tokenMaker, next)
		})

		// Home
		mux.Get(routes.Pattern(routes.Home), s.handler.HomePage)

		// User
		// ----------------------------------------------------------------------------
		mux.Group(func(r chi.Router) {
			r.Use(middlewares.IsAdmin)
			r.Get(routes.Pattern(routes.UserInvitePage), s.handler.UserInvitePage)
			r.Post(routes.Pattern(routes.PartialUserInvite), s.handler.UserInvitePartialLink)
		})

		mux.Get(routes.Pattern(routes.UserProfilePage), s.handler.UserProfilePage)
		mux.Get(routes.Pattern(routes.UserProfileDeletePage), s.handler.UserProfileDeletePage)
		mux.Post(routes.Pattern(routes.UserProfileDelete), s.handler.UserProfileDelete)
		mux.Post(routes.Pattern(routes.PartialUserProfilePassword), s.handler.UserProfilePartialPassword)

		// Chat
		// ----------------------------------------------------------------------------
		mux.Get(routes.Pattern(routes.ChatCreatePage), s.handler.ChatCreatePage)
		mux.Post(routes.Pattern(routes.ChatCreate), s.handler.ChatCreate)

		mux.Get(routes.Pattern(routes.ChatPage), s.handler.ChatPage)
		mux.Post(routes.Pattern(routes.ChatMessageCreate), s.handler.MessageCreate)
		mux.Get(routes.Pattern(routes.ChatMessageEvents), s.handler.MessageEvents)
		mux.Get(routes.Pattern(routes.ChatFile), s.handler.ChatFile)

		// Attachments
		mux.Get(routes.Pattern(routes.PartialChatMessageFiles), s.handler.PartialChatMessageFiles)
		mux.Get(routes.Pattern(routes.PartialChatModalClose), s.handler.PartialChatModalClose)

		// Chat Profile
		// ----------------------------------------------------------------------------
		mux.Get(routes.Pattern(routes.ChatProfilePage), s.handler.ChatProfilePage)
		mux.Post(routes.Pattern(routes.ChatProfileDelete), s.handler.ChatProfileDelete)

		// Edit Chat Name
		mux.Get(routes.Pattern(routes.PartialChatNameDisplay), s.handler.PartialChatNameDisplay)
		mux.Get(routes.Pattern(routes.PartialChatNameForm), s.handler.PartialChatNameForm)
		mux.Put(routes.Pattern(routes.PartialChatNameFormSubmit), s.handler.PartialChatNameFormSubmit)

		// Chat User Remove
		mux.Delete(routes.Pattern(routes.PartialChatUserRemove), s.handler.PartialChatUserRemove)
		mux.Post(routes.Pattern(routes.PartialChatUserRemoveUndo), s.handler.PartialChatUserRemoveUndo)

		// Chat User Add
		mux.Post(routes.Pattern(routes.PartialChatUserSearch), s.handler.PartialChatUserAddSearchResult)
		mux.Post(routes.Pattern(routes.PartialChatUserAdd), s.handler.PartialChatUserAdd)
		mux.Delete(routes.Pattern(routes.PartialChatUserAddUndo), s.handler.PartialChatUserAddUndo)
	})

	return mux
}
