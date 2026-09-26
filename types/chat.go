package types

import (
	"strconv"

	"github.com/forrest-bajbek/bombs/routes"
)

type Chat struct {
	ID   int
	Name string
}

func (c *Chat) ChatLink() string {
	return routes.URL(routes.ChatPage, strconv.Itoa(c.ID))
}

func (c *Chat) ChatProfileDeleteLink() string {
	return routes.URL(routes.ChatProfileDelete, strconv.Itoa(c.ID))
}

func (c *Chat) PartialChatNameDisplayLink() string {
	return routes.URL(routes.PartialChatNameDisplay, strconv.Itoa(c.ID))
}

func (c *Chat) PartialChatNameFormLink() string {
	return routes.URL(routes.PartialChatNameForm, strconv.Itoa(c.ID))
}

func (c *Chat) ChatProfileLink() string {
	return routes.URL(routes.ChatProfilePage, strconv.Itoa(c.ID))
}

type ChatUser struct {
	ID     int
	ChatID int
	UserID int
}
