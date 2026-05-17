package server

import (
	"net/http"

	"github.com/forrest-bajbek/bombs/handlers"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/services"
	"github.com/forrest-bajbek/bombs/token"
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
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	// Auth
	// --------------------------------------------------------------------------------
	mux.Handle("GET /login", http.HandlerFunc(s.handler.LoginPage))
	mux.Handle("POST /login", http.HandlerFunc(s.handler.LogIn))
	mux.Handle("POST /logout", http.HandlerFunc(s.handler.LogOut))

	// Home
	// --------------------------------------------------------------------------------
	mux.Handle("GET /", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.HomePage)))

	// User
	// --------------------------------------------------------------------------------
	mux.Handle("GET /user/invite", middlewares.IsAuthenticated(s.service, s.tokenMaker, middlewares.IsAdmin(http.HandlerFunc(s.handler.UserInvitePage))))
	mux.Handle("POST /user/invite", middlewares.IsAuthenticated(s.service, s.tokenMaker, middlewares.IsAdmin(http.HandlerFunc(s.handler.UserInvitePartialLink))))
	mux.Handle("GET /user/create", http.HandlerFunc(s.handler.UserCreatePage))
	mux.Handle("POST /user/create", http.HandlerFunc(s.handler.UserCreate))

	mux.Handle("GET /user/profile", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.UserProfilePage)))
	mux.Handle("GET /user/profile/delete", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.UserProfileDeletePage)))
	mux.Handle("POST /user/profile/delete", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.UserProfileDelete)))
	mux.Handle("POST /user/profile/partial/password", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.UserProfilePartialPassword)))

	// Chat
	// --------------------------------------------------------------------------------
	mux.Handle("GET /chat/create", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.ChatCreatePage)))
	mux.Handle("POST /chat/create", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.ChatCreate)))

	mux.Handle("GET /chat/{chat_id}", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.ChatPage)))
	mux.Handle("POST /chat/{chat_id}/message/create", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.MessageCreate)))
	mux.Handle("GET /chat/{chat_id}/message/events", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.MessageEvents)))

	// Chat Profile
	// --------------------------------------------------------------------------------
	mux.Handle("GET /chat/{chat_id}/profile", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.ChatProfilePage)))
	mux.Handle("POST /chat/{chat_id}/profile/delete", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.ChatProfileDelete)))

	// Edit Chat Name
	mux.Handle("GET /partial/chat/{chat_id}/name/display", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatNameDisplay)))
	mux.Handle("GET /partial/chat/{chat_id}/name/form", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatNameForm)))
	mux.Handle("PUT /partial/chat/{chat_id}/name/form", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatNameFormSubmit)))

	// Chat User Remove
	mux.Handle("DELETE /partial/chat/{chat_id}/user/{user_id}/remove", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatUserRemove)))
	mux.Handle("POST /partial/chat/{chat_id}/user/{user_id}/remove/undo", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatUserRemoveUndo)))

	// Chat User Add
	mux.Handle("POST /partial/chat/{chat_id}/user/search", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatUserAddSearchResult)))
	mux.Handle("POST /partial/chat/{chat_id}/user/{user_id}/add", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatUserAdd)))
	mux.Handle("DELETE /partial/chat/{chat_id}/user/{user_id}/add/undo", middlewares.IsAuthenticated(s.service, s.tokenMaker, http.HandlerFunc(s.handler.PartialChatUserAddUndo)))

	return http.ListenAndServe(addr, middlewares.Logging(middlewares.Session(mux, s.tokenMaker)))
}
