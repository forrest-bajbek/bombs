package types

import "fmt"

type Chat struct {
	ID   int
	Name string
}

func (c *Chat) ChatLink() string {
	return fmt.Sprintf("/chat/%d", c.ID)
}

func (c *Chat) ChatProfileDeleteLink() string {
	return fmt.Sprintf("/chat/%d/profile/delete", c.ID)
}

func (c *Chat) PartialChatNameDisplayLink() string {
	return fmt.Sprintf("/partial/chat/%d/name/display", c.ID)
}

func (c *Chat) PartialChatNameFormLink() string {
	return fmt.Sprintf("/partial/chat/%d/name/form", c.ID)
}

func (c *Chat) ChatProfileLink() string {
	return fmt.Sprintf("/chat/%d/profile", c.ID)
}

func (c *Chat) PartialChatMessageCreateLink() string {
	return fmt.Sprintf("/partial/chat/%d/message/create", c.ID)
}

type ChatUser struct {
	ID     int
	ChatID int
	UserID int
}
