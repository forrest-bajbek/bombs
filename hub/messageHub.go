package hub

import (
	"sync"

	"github.com/forrest-bajbek/bombs/types"
	"github.com/forrest-bajbek/bombs/utils"
)

// clientChannelBuffer is how far behind a connected client may fall
// before its messages start being dropped.
const clientChannelBuffer = 64

type MessageHub struct {
	Broadcast        chan types.ChannelMessage
	UnregisterClient chan string
	ClientChannels   map[string]chan types.ChannelMessage
	ClientToChatMap  map[string]int
	ClientMutex      sync.RWMutex
}

func NewMessageHub() *MessageHub {
	return &MessageHub{
		Broadcast:        make(chan types.ChannelMessage),
		UnregisterClient: make(chan string),
		ClientChannels:   make(map[string]chan types.ChannelMessage),
		ClientToChatMap:  make(map[string]int),
		ClientMutex:      sync.RWMutex{},
	}
}

func (h *MessageHub) CreateClientChannel(chatID int) (string, chan types.ChannelMessage, error) {
	h.ClientMutex.Lock()
	defer h.ClientMutex.Unlock()
	clientID := utils.GenerateRandomHexToken(64)
	// Buffered so that a client which is slow to drain - replaying a long
	// history of photos, say - doesn't stall the fan-out for everyone else.
	clientChannel := make(chan types.ChannelMessage, clientChannelBuffer)
	h.ClientChannels[clientID] = clientChannel // add to ClientChannels
	h.ClientToChatMap[clientID] = chatID       // map clientID to chatID
	return clientID, clientChannel, nil
}

func (h *MessageHub) DeleteClientChannel(clientID string) {
	h.ClientMutex.Lock()
	defer h.ClientMutex.Unlock()
	if ch, exists := h.ClientChannels[clientID]; exists {
		close(ch)
		delete(h.ClientChannels, clientID)
	}
	delete(h.ClientToChatMap, clientID)
}

func (h *MessageHub) Run() {
	for {
		select {
		case clientID := <-h.UnregisterClient:
			go h.DeleteClientChannel(clientID)
		case message := <-h.Broadcast:
			h.broadcast(message)
		}
	}
}

func (h *MessageHub) broadcast(message types.ChannelMessage) {
	h.ClientMutex.RLock()
	defer h.ClientMutex.RUnlock()

	for clientID, chatID := range h.ClientToChatMap {
		if chatID != message.ChatID {
			continue
		}
		clientChannel, exists := h.ClientChannels[clientID]
		if !exists {
			continue
		}
		// Never block the hub on one client. A connection that has fallen
		// this far behind is already broken; dropping it is better than
		// wedging delivery for every other chat in the app.
		select {
		case clientChannel <- message:
		default:
		}
	}
}
