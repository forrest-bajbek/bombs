package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/forrest-bajbek/bombs/components"
	"github.com/forrest-bajbek/bombs/middlewares"
	"github.com/forrest-bajbek/bombs/routes"
	"github.com/forrest-bajbek/bombs/types"
	"github.com/forrest-bajbek/bombs/utils"
)

// heartbeatInterval keeps idle SSE streams from being reaped by proxies.
const heartbeatInterval = 25 * time.Second

var bombWords = []string{"💣", "bomb", "bombs"}

func pathInt(r *http.Request, name string) (int, error) {
	raw := r.PathValue(name)
	if raw == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	return strconv.Atoi(raw)
}

func (h *Handler) ChatPage(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, routes.URL(routes.LoginPage), http.StatusFound)
		return
	}

	chatID, err := pathInt(r, "chat_id")
	if err != nil {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
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

// parseMessageUpload streams the multipart body rather than calling
// ParseMultipartForm, which would spill anything over its memory limit into
// temp files. This app keeps messages in memory and sets secure_delete
// precisely so nothing lands on disk; writing photos to /tmp would quietly
// undo that.
func parseMessageUpload(w http.ResponseWriter, r *http.Request) (string, []types.NewFile, error) {
	r.Body = http.MaxBytesReader(w, r.Body, types.MaxRequestBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		return "", nil, errors.New("Could not read message.")
	}

	var text string
	var files []types.NewFile

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, errors.New("Upload was too large or could not be read.")
		}

		switch part.FormName() {
		case "text":
			b, err := io.ReadAll(io.LimitReader(part, 4096))
			part.Close()
			if err != nil {
				return "", nil, errors.New("Could not read message text.")
			}
			text = string(b)

		case "files":
			// A file picker the user never touched still submits a part,
			// with an empty filename and no body.
			if part.FileName() == "" {
				part.Close()
				continue
			}
			if len(files) >= types.MaxFilesPerMessage {
				part.Close()
				return "", nil, fmt.Errorf("You can attach at most %d photos.", types.MaxFilesPerMessage)
			}
			// One byte past the cap, so an oversized upload is caught
			// without buffering the whole thing.
			b, err := io.ReadAll(io.LimitReader(part, types.MaxFileBytes+1))
			part.Close()
			if err != nil {
				return "", nil, errors.New("Could not read uploaded photo.")
			}
			if len(b) == 0 {
				continue
			}
			f, err := utils.ProcessUpload(b)
			if err != nil {
				return "", nil, err
			}
			files = append(files, f)

		default:
			part.Close()
		}
	}

	return text, files, nil
}

func (h *Handler) MessageCreate(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, routes.URL(routes.LoginPage), http.StatusFound)
		return
	}
	chatID, err := pathInt(r, "chat_id")
	if err != nil {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}

	// Errors render as a 200 with the form swapped back in. htmx ignores
	// the body of a 4xx/5xx by default, so a non-2xx would leave the user
	// staring at an unchanged form with no explanation.
	renderInputBar := func(errorMessage string) {
		components.MessageInputBar(chatID, errorMessage).Render(context.Background(), w)
	}

	text, files, err := parseMessageUpload(w, r)
	if err != nil {
		renderInputBar(err.Error())
		return
	}

	// Handle Bombs
	if slices.Contains(bombWords, strings.ToLower(strings.TrimSpace(text))) {
		// Any photos attached to a bomb are dropped on the floor - the
		// message that would own them is about to be erased anyway.
		if err := h.service.BombChat(requestingUser.ID, chatID); err != nil {
			renderInputBar(err.Error())
			return
		}
		bombMessage := types.ChannelMessage{
			MessageID:        -1,
			MessageCreatedAt: time.Now(),
			ChatID:           chatID,
			UserID:           -1,
			Username:         "bombs",
			Text:             "💣",
		}
		renderInputBar("")
		h.messageHub.Broadcast <- bombMessage
		return
	}

	messageID, err := h.service.CreateMessage(requestingUser.ID, chatID, text, files)
	if err != nil {
		renderInputBar(err.Error())
		return
	}
	newMessage, err := h.service.GetMessageByID(requestingUser.ID, chatID, messageID)
	if err != nil {
		renderInputBar(err.Error())
		return
	}

	// Respond before broadcasting so the sender's request isn't waiting on
	// the hub to fan out to every other client.
	renderInputBar("")
	h.messageHub.Broadcast <- *newMessage
}

// writeSSEEvent frames pre-rendered HTML as a single SSE event.
//
// A data field can't contain a line break, so the HTML is split and each
// line sent as its own data field; the browser rejoins them. Carriage
// returns are normalized first because a bare CR also terminates a field,
// which would otherwise let message content end the event early and inject
// a forged one. The whole frame goes out in one Write so it can't be torn.
func writeSSEEvent(w io.Writer, event string, id int, html []byte) error {
	var b bytes.Buffer

	b.WriteString("event: ")
	b.WriteString(event)
	b.WriteByte('\n')

	// The bomb pseudo-message has no real id; emitting one would poison
	// the browser's Last-Event-ID and break replay on reconnect.
	if id > 0 {
		fmt.Fprintf(&b, "id: %d\n", id)
	}

	html = bytes.ReplaceAll(html, []byte("\r\n"), []byte("\n"))
	html = bytes.ReplaceAll(html, []byte("\r"), []byte("\n"))
	for _, line := range bytes.Split(html, []byte("\n")) {
		b.WriteString("data: ")
		b.Write(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	_, err := w.Write(b.Bytes())
	return err
}

// writeMessage renders one bubble for one recipient and sends it. The HTML
// differs per recipient (IsSender drives which side of the thread it sits
// on), so this has to happen per client rather than once in the hub.
func writeMessage(w io.Writer, m types.ChannelMessage, requestingUserID int) error {
	rm := types.ResponseMessage{
		ChannelMessage: m,
		IsSender:       m.UserID == requestingUserID,
	}

	var buf bytes.Buffer
	if err := components.ChatMessageBubble(&rm).Render(context.Background(), &buf); err != nil {
		return err
	}

	event := "newMessage"
	id := rm.MessageID
	if strings.TrimSpace(rm.Text) == "💣" {
		event = "bomb"
		id = 0
	}

	return writeSSEEvent(w, event, id, buf.Bytes())
}

func (h *Handler) MessageEvents(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, routes.URL(routes.LoginPage), http.StatusFound)
		return
	}
	chatID, err := pathInt(r, "chat_id")
	if err != nil {
		http.Error(w, "Cannot parse url", http.StatusInternalServerError)
		return
	}

	clientID, clientChannel, err := h.messageHub.CreateClientChannel(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		h.messageHub.UnregisterClient <- clientID
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	rc := http.NewResponseController(w)
	ctx := r.Context()

	// Replay history. Anything the browser already has is skipped, so a
	// dropped connection resumes instead of appending the whole
	// conversation a second time.
	lastEventID, _ := strconv.Atoi(r.Header.Get("Last-Event-ID"))
	databaseMessages, err := h.service.GetMessagesByChatID(requestingUser.ID, chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, m := range *databaseMessages {
		if m.MessageID <= lastEventID {
			continue
		}
		if err := writeMessage(w, m, requestingUser.ID); err != nil {
			return
		}
	}
	if err := rc.Flush(); err != nil {
		return
	}

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done(): // client disconnected
			return

		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}

		case m, open := <-clientChannel:
			if !open {
				return
			}
			if err := writeMessage(w, m, requestingUser.ID); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				log.Printf("error flushing message stream: %v", err)
				return
			}
		}
	}
}

// ChatFile serves an attachment's bytes. The cookie that authenticates the
// request proves who is asking; the query behind GetFile is what proves
// they're allowed to see this particular file.
func (h *Handler) ChatFile(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, routes.URL(routes.LoginPage), http.StatusFound)
		return
	}
	chatID, err := pathInt(r, "chat_id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fileID, err := pathInt(r, "file_id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// 404 rather than 403, so file ids can't be probed for existence.
	file, err := h.service.GetFile(requestingUser.ID, chatID, fileID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Re-checked on the way out, not just on the way in: this is the
	// header that decides how a browser will interpret these bytes.
	if !types.AllowedImageMimeTypes[file.MimeType] {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", file.MimeType)
	// Without nosniff, a file that looks like an image to the server's
	// sniffer but like HTML to the browser becomes stored XSS on the
	// origin that holds the auth cookie.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Only takes effect if the URL is opened as a document; harmless for
	// <img>, and neutralizes scripts if content-type enforcement fails.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Disposition", "inline")
	// Bombing a chat is supposed to erase it. A copy sitting in every
	// member's disk cache would make that untrue.
	w.Header().Set("Cache-Control", "no-store")

	// Empty name so it can't infer a type from an extension; the
	// Content-Type set above is kept.
	http.ServeContent(w, r, "", file.CreatedAt, bytes.NewReader(file.Content))
}

func (h *Handler) PartialChatMessageFiles(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(middlewares.AuthUser).(*types.User)
	if !ok {
		http.Redirect(w, r, routes.URL(routes.LoginPage), http.StatusFound)
		return
	}
	chatID, err := pathInt(r, "chat_id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	messageID, err := pathInt(r, "message_id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	m, err := h.service.GetMessageByID(requestingUser.ID, chatID, messageID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// A missing or garbled index opens the first photo; one outside the
	// message's photos is a tampered URL.
	index, err := strconv.Atoi(r.URL.Query().Get("i"))
	if err != nil {
		index = 0
	}
	if index < 0 || index >= len(m.Files) {
		http.NotFound(w, r)
		return
	}

	rm := types.ResponseMessage{
		ChannelMessage: *m,
		IsSender:       m.UserID == requestingUser.ID,
	}
	components.PartialMessageFileModal(&rm, index).Render(context.Background(), w)
}

// PartialChatModalClose empties the modal container. It must answer 200
// with an empty body rather than 204, which htmx treats as "change
// nothing" - the modal would stay open.
func (h *Handler) PartialChatModalClose(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
