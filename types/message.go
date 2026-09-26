package types

import (
	"fmt"
	"strconv"
	"time"

	"github.com/forrest-bajbek/bombs/routes"
)

type Message struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ChatID    int       `json:"chat_id"`
	UserID    int       `json:"user_id"`
	Text      string    `json:"text"`
}

type ChannelMessage struct {
	MessageID        int           `json:"message_id"`
	MessageCreatedAt time.Time     `json:"message_created_at"`
	ChatID           int           `json:"chat_id"`
	UserID           int           `json:"user_id"`
	Username         string        `json:"username"`
	Text             string        `json:"text"`
	Files            []MessageFile `json:"files"`
}

type ResponseMessage struct {
	ChannelMessage
	IsSender bool `json:"is_sender"`
}

type ChatPreview struct {
	ChatID                 int
	ChatName               string
	LastMessageUsername    string
	LastMessageText        string
	LastMessageCreatedAt   string
	LastMessageCountPhotos int
}

func (c *ChatPreview) ChatURL() string {
	return routes.URL(routes.ChatPage, strconv.Itoa(c.ChatID))
}

func (c *ChatPreview) LastMessageTextPreview() string {
	if len(c.LastMessageText) <= 20 {
		return c.LastMessageText
	}
	return fmt.Sprintf("%s...", c.LastMessageText[:20])
}
