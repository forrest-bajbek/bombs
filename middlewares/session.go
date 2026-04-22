package middlewares

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

func Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Incoming request
		// ----------------------------------------------------------------------------
		sessionCookie, err := r.Cookie("session")
		if err != nil {
			// No cookie. Serve request
			next.ServeHTTP(w, r)
		} else {
			// Extract userID from cookie, add to request context, then serve request
			userID := sessionCookie.Value
			ctx := context.WithValue(r.Context(), AuthUserID, userID)
			req := r.WithContext(ctx)
			next.ServeHTTP(w, req)
		}

		// Outgoing request
		// ----------------------------------------------------------------------------
		// Check for userID in context
		userID, ok := r.Context().Value(AuthUserID).(int)
		if ok {
			// If userID exists, set session cookie
			cookie := &http.Cookie{
				Name:     "session",
				Value:    strconv.Itoa(userID),
				Path:     "/",
				Expires:  time.Now().Add(24 * time.Hour), // Optional
				HttpOnly: true,                           // Helps prevent XSS
				Secure:   r.TLS != nil,                   // Set only on HTTPS
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, cookie)
		} else {
			// If userID doesn't exist, ensure session cookie gets deleted
			cookie := &http.Cookie{
				Name:   "session",
				Value:  "",
				MaxAge: -1,
			}
			http.SetCookie(w, cookie)
		}
	})
}
