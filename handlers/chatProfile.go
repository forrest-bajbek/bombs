package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/types"
)

// Chat Profile
// ------------------------------------------------------------------------------------
func (h *Handler) ChatProfilePage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	chat, err := h.service.GetChatByID(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chatUsers, err := h.service.GetChatUser(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.ChatProfilePage(chat, chatUsers, requestingUser.IsAdmin)
	component.Render(context.Background(), w)
}

// Chat Name
// ++++++++++++++++++++
func (h *Handler) PartialChatNameDisplay(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
		return
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	chat, err := h.service.GetChatByID(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatNameDisplay(chat.Name, chat.PartialChatNameFormLink())
	component.Render(context.Background(), w)
}

func (h *Handler) PartialChatNameForm(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
		return
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	chat, err := h.service.GetChatByID(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatNameForm(chat.Name, chat.PartialChatNameFormLink(), chat.PartialChatNameDisplayLink(), "")
	component.Render(context.Background(), w)
}

func (h *Handler) PartialChatNameFormSubmit(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
		return
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	chat, err := h.service.GetChatByID(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = r.ParseForm()
	if err != nil {
		component := components.PartialChatNameForm(chat.Name, chat.PartialChatNameFormLink(), chat.PartialChatNameDisplayLink(), err.Error())
		component.Render(context.Background(), w)
		return
	}
	updatedChatName := r.PostForm.Get("chatName")
	updatedChat, err := h.service.UpdateChat(requestingUser.ID, chat.ID, updatedChatName)
	if err != nil {
		component := components.PartialChatNameForm(updatedChatName, chat.PartialChatNameFormLink(), chat.PartialChatNameDisplayLink(), err.Error())
		component.Render(context.Background(), w)
		return
	}

	component := components.PartialChatNameDisplay(updatedChat.Name, updatedChat.PartialChatNameFormLink())
	component.Render(context.Background(), w)

}

// Chat Delete
// ++++++++++++++++++++
func (h *Handler) ChatProfileDelete(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
		return
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.service.DeleteChat(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// Chat User Remove
// ++++++++++++++++++++
func (h *Handler) PartialChatUserRemove(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user_id := r.PathValue("user_id")
	if user_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	userID, err := strconv.Atoi(user_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.service.DeleteChatUser(requestingUser.ID, chatID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatUserRemoveUndo(chatID, userID)
	component.Render(context.Background(), w)
}

func (h *Handler) PartialChatUserRemoveUndo(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user_id := r.PathValue("user_id")
	if user_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	userID, err := strconv.Atoi(user_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.service.AddChatUser(requestingUser.ID, chatID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatUserRemove(chatID, userID)
	component.Render(context.Background(), w)
}

// Chat User Add
// ++++++++++++++++++++
func (h *Handler) PartialChatUserAddSearchResult(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	search_term := r.PostForm.Get("search_term")

	users, err := h.service.SearchForNewUsers(requestingUser.ID, chatID, search_term)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatUserAddSearchResult(chatID, users)
	component.Render(context.Background(), w)
}

func (h *Handler) PartialChatUserAdd(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user_id := r.PathValue("user_id")
	if user_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	userID, err := strconv.Atoi(user_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.service.AddChatUser(requestingUser.ID, chatID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatUserAddUndo(chatID, userID)
	component.Render(context.Background(), w)
}

func (h *Handler) PartialChatUserAddUndo(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Error(w, "Could not retrieve user from context", http.StatusBadRequest)
	}

	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user_id := r.PathValue("user_id")
	if user_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	userID, err := strconv.Atoi(user_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.service.DeleteChatUser(requestingUser.ID, chatID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := components.PartialChatUserAdd(chatID, userID)
	component.Render(context.Background(), w)
}
