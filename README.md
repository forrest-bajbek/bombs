# Bombs, a chat app

The prevailing security model for modern chat applications (Simplex, Signal, Telegram, Matrix) is "just encrypt bro". While encryption is important, it makes an implicit assertion: "messages should be recoverable for as long as possible". This is a strange requirement for something as nonchalant as text messages.

Most applications:

1. store all of your messages forever, even if they "let" you delete things.
1. cache your messages and media for performance reasons, making many copies of your messages.
1. send push notifications, which send your messages unencrypted through centralized APIs owned by Apple and Google.
1. store your messages on hard drives that are often backed up
1. delete your messages in such a way that forensic analysts can easily recover

Despite cultural expectations, _you are not legally obligated to save every text message you've ever sent over your entire life_, and _using software that doesn't retain message history isn't illegal_.

Bombs works differently than most apps.

1. When a user types "bomb", "bombs", or "💣" in a chat, it deletes all messages in the chat.
1. Nothing is cached. The chat history is re-downloaded every time you navigate to it, even photos.
1. There are no push notifications (you check your phone all the time anyway)
1. Your account and chat room configuration is stored in SQLite. But your messages and photos are stored in-memory.
1. Message and photo tables set [`PRAGMA memdb.auto_vacuum = FULL`](https://sqlite.org/lang_vacuum.html).

Plus:

- Deleting your account automatically deletes all your messages and attachments in all chat rooms.
- Deleting a chat room automatically deletes all messages and attachments in the chat room.
- Deleting an individual message automatically deletes all attachments for that message.

In other words, Bombs makes deleting messages easy, and it actually deletes things.

If the last few years of politics has taught us anything, it's that people are rarely in court for things they texted over the last 3 days. If a judge orders you to hand over your Bombs server, eagerly give them the whole computer. If the prosecution wants to recover the immediately preceding 72 hours of message history from unpowered volatile memory, let them. You weren't retroactively required to store everything you ever said online just in case some prosecutor wanted a great case against you.

## Architecture

Bombs is an MVC Web App written in Golang.

- [Chi Router](https://github.com/go-chi/chi) for routing and middleware.
- [Templ](https://templ.guide/) for HTML templating ,
- [HTMX](https://htmx.org/) for interactivity.
- [SQLite](https://github.com/ncruces/go-sqlite3) (built from a WASM port that doesn't require `glibc`).

# Deployment

Bombs will run on [any OS target supported by Go](https://github.com/golang/go/blob/740fcaabb9e2ee54b2575ad04bd308a87cc3c406/src/internal/syslist/syslist.go). You don't need Docker to run the server. Just build the binary with `go build .`, then run the binary. However, doing it this way, you'll have to figure out networking, TLS, and the reverse proxy.

For convenience, the repo provides a `docker-compose.yaml` configured with `Caddy2` for reverse proxy and automatic TLS. It also includes an example Caddyfile.

## Docker Compose

Create `Caddyfile` by making a copy of `Caddyfile.example`. Be sure to replace the example domain with your own. Also, make a `.env` file by copying `env.example`. Be sure to generate your own values.

# Miscellaneous

## Libraries used in project

- [Templ](https://templ.guide/quick-start/installation)
- [Go-SQLite](https://github.com/ncruces/go-sqlite3)

## Useful articles

- [SQLite implementation benchmarks](https://github.com/cvilsmeier/go-sqlite-bench)
- [Great article on the Repository Pattern in Go](https://pawelgrzybek.com/repository-pattern-in-go-service/)
