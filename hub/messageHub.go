package hub

import (
	"sync"

	"github.com/forrest-bajbek/bombs/types"
	"github.com/forrest-bajbek/bombs/utils"
)

type MessageHub struct {
	Broadcast        chan types.MessageDetail
	UnregisterClient chan string
	ClientChannels   map[string]chan types.MessageDetail
	ClientToChatMap  map[string]int
	ClientMutex      sync.RWMutex
}

func NewMessageHub() *MessageHub {
	return &MessageHub{
		Broadcast:        make(chan types.MessageDetail),
		UnregisterClient: make(chan string),
		ClientChannels:   make(map[string]chan types.MessageDetail),
		ClientToChatMap:  make(map[string]int),
		ClientMutex:      sync.RWMutex{},
	}
}

func (h *MessageHub) CreateClientChannel(chatID int) (string, chan types.MessageDetail, error) {
	h.ClientMutex.Lock()
	defer h.ClientMutex.Unlock()
	clientID := utils.GenerateRandomHexToken(64)    // Generate clientID
	clientChannel := make(chan types.MessageDetail) // clientChannel
	h.ClientChannels[clientID] = clientChannel      // add to ClientChannels
	h.ClientToChatMap[clientID] = chatID            // map clientID to chatID
	return clientID, clientChannel, nil
}

func (h *MessageHub) DeleteClientChannel(clientID string) {
	h.ClientMutex.Lock()
	defer h.ClientMutex.Unlock()
	if ch, exists := h.ClientChannels[clientID]; exists {
		close(ch)                          // close clientChannel
		delete(h.ClientChannels, clientID) // delete mapping to clientChannel
	}
	if _, exists := h.ClientChannels[clientID]; exists {
		delete(h.ClientToChatMap, clientID) // delete mapping to chatID
	}
}

func (h *MessageHub) Run() {
	for {
		select {
		case clientID := <-h.UnregisterClient:
			go h.DeleteClientChannel(clientID)
		case message := <-h.Broadcast:
			for clientID, chatID := range h.ClientToChatMap {
				if chatID == message.ChatID {
					if clientChannel, exists := h.ClientChannels[clientID]; exists {
						clientChannel <- message
					}
				}
			}
		}
	}
}
