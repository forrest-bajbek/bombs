// Package routes is the single source of truth for every URL path in the
// app. Route names use dot notation and are hierarchical; HTMX endpoints
// that return a partial HTML fragment (rather than a full page or a
// redirect) are prefixed with "partial.".
package routes

import (
	"net/url"
	"strings"

	"github.com/alehano/reverse"
)

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

	// ChatFile serves attachment bytes. It's a real resource rather than
	// an HTML fragment, so it takes no "partial." prefix.
	ChatFile = "chat.file"

	PartialChatMessageFiles = "partial.chat.message.files"
	PartialChatModalClose   = "partial.chat.modal.close"

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
	chatIDParam    = "{chat_id}"
	userIDParam    = "{user_id}"
	messageIDParam = "{message_id}"
	fileIDParam    = "{file_id}"
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

	// chat_id is in the path so that authorizing a download is a single
	// join against chat_user, with no lookup to discover which chat the
	// file belongs to.
	reverse.Add(ChatFile, "/chat/"+chatIDParam+"/file/"+fileIDParam, chatIDParam, fileIDParam)

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

	reverse.Add(PartialChatMessageFiles, "/partial/chat/"+chatIDParam+"/message/"+messageIDParam+"/files", chatIDParam, messageIDParam)
	reverse.Add(PartialChatModalClose, "/partial/chat/"+chatIDParam+"/modal/close", chatIDParam)
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

// LoginURL returns the login page path, carrying next as a ?next= query
// parameter when it is a safe local path (see SafeNext).
func LoginURL(next string) string {
	loginURL := URL(LoginPage)
	if next = SafeNext(next); next != "" {
		loginURL += "?next=" + url.QueryEscape(next)
	}
	return loginURL
}

// SafeNext returns next if it is a local path that is safe to redirect to
// after login, and "" otherwise. It rejects anything that could send the
// browser off-site (e.g. "//evil.com", "/\evil.com", "https://evil.com")
// as well as the login page itself, to avoid a redirect loop.
func SafeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
		return ""
	}
	for _, c := range next {
		if c < 0x20 || c == 0x7f {
			return ""
		}
	}
	u, err := url.Parse(next)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return ""
	}
	if u.Path == URL(LoginPage) {
		return ""
	}
	return next
}
