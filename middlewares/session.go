package middlewares

import (
	"context"
	"net/http"
	"strconv"

	"github.com/forrest-bajbek/bombs/token"
)

func Session(next http.Handler, tokenMaker *token.JWTMaker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("authToken")
		if err != nil {
			// log.Printf("++ Session: authToken missing")
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := cookie.Value
		// log.Printf("++ Session: authToken found")
		authClaims, err := tokenMaker.ValidateAuthToken(tokenStr)
		if err != nil {
			// log.Printf("++ Session: authToken not valid: %s", err.Error())
			next.ServeHTTP(w, r)
			return
		}

		// log.Printf("++ Session: Setting userID in request context")
		userID, err := strconv.Atoi(authClaims.UserID)
		if err != nil {
			// log.Printf("++ Session: Error converting userID to int")
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), AuthUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
