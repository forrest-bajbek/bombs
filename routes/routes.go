// Package routes is the single source of truth for every URL path in the
// app. Route names use dot notation and are hierarchical; HTMX endpoints
// that return a partial HTML fragment (rather than a full page or a
// redirect) are prefixed with "partial.".
package routes

import "github.com/alehano/reverse"

const (
	Health = "health"

	LoginPage = "login.page"
	Login     = "login"
	Logout    = "logout"

	Home = "home"

	UserCreatePage = "user.create.page"
	UserCreate     = "user.create"

	UserInvitePage    = "user.invite.page"
	PartialUserInvite = "partial.user.invite"

	UserProfilePage       = "user.profile.page"
	UserProfileDeletePage = "user.profile.delete.page"
	UserProfileDelete     = "user.profile.delete"

	PartialUserProfilePassword = "partial.user.profile.password"

	ChatCreatePage = "chat.create.page"
	ChatCreate     = "chat.create"
	ChatPage       = "chat.page"

	ChatMessageCreate = "chat.message.create"
	ChatMessageEvents = "chat.message.events"

	ChatProfilePage   = "chat.profile.page"
	ChatProfileDelete = "chat.profile.delete"

	PartialChatNameDisplay    = "partial.chat.name.display"
	PartialChatNameForm       = "partial.chat.name.form"
	PartialChatNameFormSubmit = "partial.chat.name.form.submit"

	PartialChatUserRemove     = "partial.chat.user.remove"
	PartialChatUserRemoveUndo = "partial.chat.user.remove.undo"
	PartialChatUserSearch     = "partial.chat.user.search"
	PartialChatUserAdd        = "partial.chat.user.add"
	PartialChatUserAddUndo    = "partial.chat.user.add.undo"
)

const (
	chatIDParam = "{chat_id}"
	userIDParam = "{user_id}"
)

// Register populates the global reverse.Urls store with every route in the
// app, keyed by the named constants above. It must run once before Pattern
// or URL are called — server.go calls it while building the router, and
// pulls chi's route patterns from the same table via Pattern, so the path
// for a given route only ever lives here.
func Register() {
	reverse.Add(Health, "/health")

	reverse.Add(LoginPage, "/login")
	reverse.Add(Login, "/login")
	reverse.Add(Logout, "/logout")

	reverse.Add(Home, "/")

	reverse.Add(UserCreatePage, "/user/create")
	reverse.Add(UserCreate, "/user/create")

	reverse.Add(UserInvitePage, "/user/invite")
	reverse.Add(PartialUserInvite, "/user/invite")

	reverse.Add(UserProfilePage, "/user/profile")
	reverse.Add(UserProfileDeletePage, "/user/profile/delete")
	reverse.Add(UserProfileDelete, "/user/profile/delete")

	reverse.Add(PartialUserProfilePassword, "/user/profile/partial/password")

	reverse.Add(ChatCreatePage, "/chat/create")
	reverse.Add(ChatCreate, "/chat/create")
	reverse.Add(ChatPage, "/chat/"+chatIDParam, chatIDParam)

	reverse.Add(ChatMessageCreate, "/chat/"+chatIDParam+"/message/create", chatIDParam)
	reverse.Add(ChatMessageEvents, "/chat/"+chatIDParam+"/message/events", chatIDParam)

	reverse.Add(ChatProfilePage, "/chat/"+chatIDParam+"/profile", chatIDParam)
	reverse.Add(ChatProfileDelete, "/chat/"+chatIDParam+"/profile/delete", chatIDParam)

	reverse.Add(PartialChatNameDisplay, "/partial/chat/"+chatIDParam+"/name/display", chatIDParam)
	reverse.Add(PartialChatNameForm, "/partial/chat/"+chatIDParam+"/name/form", chatIDParam)
	reverse.Add(PartialChatNameFormSubmit, "/partial/chat/"+chatIDParam+"/name/form", chatIDParam)

	reverse.Add(PartialChatUserRemove, "/partial/chat/"+chatIDParam+"/user/"+userIDParam+"/remove", chatIDParam, userIDParam)
	reverse.Add(PartialChatUserRemoveUndo, "/partial/chat/"+chatIDParam+"/user/"+userIDParam+"/remove/undo", chatIDParam, userIDParam)
	reverse.Add(PartialChatUserSearch, "/partial/chat/"+chatIDParam+"/user/search", chatIDParam)
	reverse.Add(PartialChatUserAdd, "/partial/chat/"+chatIDParam+"/user/"+userIDParam+"/add", chatIDParam, userIDParam)
	reverse.Add(PartialChatUserAddUndo, "/partial/chat/"+chatIDParam+"/user/"+userIDParam+"/add/undo", chatIDParam, userIDParam)
}

// Pattern returns the chi route pattern registered for name, e.g.
// Pattern(ChatPage) -> "/chat/{chat_id}".
func Pattern(name string) string {
	return reverse.Get(name)
}

// URL reverses name into a concrete path using params in order, e.g.
// URL(ChatPage, "42") -> "/chat/42".
func URL(name string, params ...string) string {
	return reverse.Rev(name, params...)
}
