package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/types"
)

func (h *Handler) ChatPage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
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

	component := components.ChatPage(chat)
	component.Render(context.Background(), w)
}

func (h *Handler) MessageCreate(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	chat_id := r.PathValue("chat_id")
	if chat_id == "" {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}
	chatID, err := strconv.Atoi(chat_id)
	if err != nil {
		log.Printf("error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type MessageRequest struct {
		Text string `json:"text"`
	}
	var m MessageRequest
	err = json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		log.Printf("error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	messageID, err := h.service.CreateMessage(requestingUser.ID, chatID, m.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newMessage, err := h.service.GetMessageByID(requestingUser.ID, chatID, messageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.messageHub.Broadcast <- *newMessage
}

func (h *Handler) MessageEvents(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusFound)
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

	clientID, clientChannel, err := h.messageHub.CreateClientChannel(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rc := http.NewResponseController(w)
	ctx := r.Context()

	// // send most recent messages from database
	databaseMessages, err := h.service.GetMessagesByChatID(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, m := range *databaseMessages {
		rm := types.ResponseMessage{
			ChannelMessage: m,
			IsSender:       m.UserID == requestingUser.ID,
		}
		jsonData, err := json.Marshal(&rm)
		if err != nil {
			log.Printf("Error marshaling JSON: %v", err)
			return
		}
		_, err = fmt.Fprintf(w, "event:%s\nid: %d\ndata: %s\n\n", "newMessage", rm.MessageID, string(jsonData))
		if err != nil {
			log.Printf("Error sending event...")
			return
		}
		err = rc.Flush()
		if err != nil {
			log.Printf("Error flushing: %v", err)
			return
		}
	}

	defer func() {
		h.messageHub.UnregisterClient <- clientID
	}()

	for {
		select {
		case <-ctx.Done(): // client disconnected
			h.messageHub.UnregisterClient <- clientID
			return
		case m := <-clientChannel: // new message
			rm := types.ResponseMessage{
				ChannelMessage: m,
				IsSender:       m.UserID == requestingUser.ID,
			}
			jsonData, err := json.Marshal(&rm)
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				return
			}
			_, err = fmt.Fprintf(w, "event:%s\nid: %d\ndata: %s\n\n", "newMessage", rm.MessageID, string(jsonData))
			if err != nil {
				log.Print("Error sending event...")
				return
			}
			err = rc.Flush()
			if err != nil {
				log.Printf("Error flushing: %v", err)
				h.messageHub.UnregisterClient <- clientID
				return
			}
		}
	}
}
