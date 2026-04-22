# Miscellaneous

## Libraries used in project

- [Templ](https://templ.guide/quick-start/installation)
- [Go-SQLite](https://github.com/ncruces/go-sqlite3)

## Useful articles

- [SQLite implementation benchmarks](https://github.com/cvilsmeier/go-sqlite-bench)
- [Great article on the Repository Pattern in Go](https://pawelgrzybek.com/repository-pattern-in-go-service/)

# Todo

1. Finish Message Send Partial
-- Write javascript to take text input, convert to JSON, and send via AJAX
-- Write Go endpoint to receive JSON and convert into row in message table
1. Figure out how to broadcast new messages to clients in same chat
-- Will probably be a new field on either the server or chatHandler Struct, specifically a dictionary
-- When a message is received, after it is successfully written to the database, send that message to a channel
-- that channel, when received, will handle sending the message to a number of active SSE connections
1. Figure out how do make messages populate from the bottom
