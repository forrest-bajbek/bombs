package hub

import (
	"sync"
	"time"

	"github.com/forrest-bajbek/bombs/types"
	"github.com/forrest-bajbek/bombs/utils"
)

// clientChannelBuffer is how far behind a connected client may fall
// before its messages start being dropped.
const clientChannelBuffer = 64

// typingTimeout is how long after their last keystroke a user stops
// counting as typing.
const typingTimeout = 3 * time.Second

// typingSweepInterval bounds how late an expired typist is cleared.
const typingSweepInterval = 250 * time.Millisecond

type Typist struct {
	UserID   int
	Username string
}

// TypingSignal reports a keystroke from a user in a chat.
type TypingSignal struct {
	ChatID   int
	UserID   int
	Username string
}

// TypingStatus is everyone currently typing in a chat. An empty Typists
// means nobody is.
type TypingStatus struct {
	Typists []Typist
}

// ClientEvent is one item delivered to a connected client; exactly one
// field is set.
type ClientEvent struct {
	Message *types.ChannelMessage
	Typing  *TypingStatus
}

type typistEntry struct {
	username string
	expires  time.Time
}

type MessageHub struct {
	Broadcast        chan types.ChannelMessage
	Typing           chan TypingSignal
	UnregisterClient chan string
	ClientChannels   map[string]chan ClientEvent
	ClientToChatMap  map[string]int
	ClientMutex      sync.RWMutex

	// syncClient asks Run to send a newly connected client the current
	// typing state, so a new tab or a reconnect sees who is already typing.
	syncClient chan string

	// typists is chatID -> userID -> entry. Only Run touches it, so it
	// needs no lock.
	typists map[int]map[int]typistEntry
}

func NewMessageHub() *MessageHub {
	return &MessageHub{
		Broadcast:        make(chan types.ChannelMessage),
		Typing:           make(chan TypingSignal),
		UnregisterClient: make(chan string),
		ClientChannels:   make(map[string]chan ClientEvent),
		ClientToChatMap:  make(map[string]int),
		ClientMutex:      sync.RWMutex{},
		syncClient:       make(chan string),
		typists:          make(map[int]map[int]typistEntry),
	}
}

func (h *MessageHub) CreateClientChannel(chatID int) (string, chan ClientEvent, error) {
	h.ClientMutex.Lock()
	clientID := utils.GenerateRandomHexToken(64)
	// Buffered so that a client which is slow to drain - replaying a long
	// history of photos, say - doesn't stall the fan-out for everyone else.
	clientChannel := make(chan ClientEvent, clientChannelBuffer)
	h.ClientChannels[clientID] = clientChannel // add to ClientChannels
	h.ClientToChatMap[clientID] = chatID       // map clientID to chatID
	h.ClientMutex.Unlock()

	// Sent after unlocking: Run takes the read lock to deliver the snapshot.
	h.syncClient <- clientID
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
	sweep := time.NewTicker(typingSweepInterval)
	defer sweep.Stop()

	for {
		select {
		case clientID := <-h.UnregisterClient:
			go h.DeleteClientChannel(clientID)

		case message := <-h.Broadcast:
			h.send(message.ChatID, ClientEvent{Message: &message})
			// Sending a message ends that user's typing, and the indicator
			// clears right behind the message it was announcing.
			if h.clearTypist(message.ChatID, message.UserID) {
				h.publishTyping(message.ChatID)
			}

		case signal := <-h.Typing:
			if h.markTypist(signal) {
				h.publishTyping(signal.ChatID)
			}

		case clientID := <-h.syncClient:
			h.syncTyping(clientID)

		case now := <-sweep.C:
			h.sweepTypists(now)
		}
	}
}

// markTypist records a keystroke and reports whether the user just started
// typing. Keystrokes from someone already typing only extend their expiry.
func (h *MessageHub) markTypist(signal TypingSignal) bool {
	chat, exists := h.typists[signal.ChatID]
	if !exists {
		chat = make(map[int]typistEntry)
		h.typists[signal.ChatID] = chat
	}
	_, wasTyping := chat[signal.UserID]
	chat[signal.UserID] = typistEntry{
		username: signal.Username,
		expires:  time.Now().Add(typingTimeout),
	}
	return !wasTyping
}

// clearTypist removes a user's typing entry and reports whether they had one.
func (h *MessageHub) clearTypist(chatID, userID int) bool {
	chat, exists := h.typists[chatID]
	if !exists {
		return false
	}
	if _, wasTyping := chat[userID]; !wasTyping {
		return false
	}
	delete(chat, userID)
	if len(chat) == 0 {
		delete(h.typists, chatID)
	}
	return true
}

func (h *MessageHub) sweepTypists(now time.Time) {
	for chatID, chat := range h.typists {
		changed := false
		for userID, entry := range chat {
			if now.After(entry.expires) {
				delete(chat, userID)
				changed = true
			}
		}
		if len(chat) == 0 {
			delete(h.typists, chatID)
		}
		if changed {
			h.publishTyping(chatID)
		}
	}
}

func (h *MessageHub) typingStatus(chatID int) *TypingStatus {
	status := &TypingStatus{}
	for userID, entry := range h.typists[chatID] {
		status.Typists = append(status.Typists, Typist{UserID: userID, Username: entry.username})
	}
	return status
}

func (h *MessageHub) publishTyping(chatID int) {
	h.send(chatID, ClientEvent{Typing: h.typingStatus(chatID)})
}

// syncTyping sends one client its chat's typing state. A client that
// connects while nobody is typing already shows nothing, so it gets nothing.
func (h *MessageHub) syncTyping(clientID string) {
	h.ClientMutex.RLock()
	defer h.ClientMutex.RUnlock()

	chatID, exists := h.ClientToChatMap[clientID]
	if !exists || len(h.typists[chatID]) == 0 {
		return
	}
	clientChannel, exists := h.ClientChannels[clientID]
	if !exists {
		return
	}
	select {
	case clientChannel <- ClientEvent{Typing: h.typingStatus(chatID)}:
	default:
	}
}

func (h *MessageHub) send(chatID int, event ClientEvent) {
	h.ClientMutex.RLock()
	defer h.ClientMutex.RUnlock()

	for clientID, clientChatID := range h.ClientToChatMap {
		if clientChatID != chatID {
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
		case clientChannel <- event:
		default:
		}
	}
}
