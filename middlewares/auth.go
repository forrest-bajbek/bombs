package middlewares

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/forrest-bajbek/bombs/services"
	"github.com/forrest-bajbek/bombs/token"
	"github.com/forrest-bajbek/bombs/types"
)

const AuthUserID = "middleware.auth.userID"
const AuthUser = "middleware.auth.user"

func writeUnauthed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
}

func IsAuthenticated(service *services.Service, tokenMaker *token.JWTMaker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(AuthUserID).(int)
		if !ok {
			// log.Printf("++ Auth: userID not found in request context.")
			// log.Printf("++ Auth: Removing cookie and redirecting to login.")
			cookie := &http.Cookie{
				Name:   "authToken",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		// log.Printf("++ Auth: userID found in request context: %d", userID)
		user, err := service.GetUserByID(userID)
		if err != nil {
			// log.Printf("++ Auth: userID %d not found in databaes", userID)
			// log.Printf("++ Auth: Removing cookie and redirecting to login.")
			cookie := &http.Cookie{
				Name:   "authToken",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// log.Printf("++ Auth: Updating authToken with new expiration.")
		authToken, _, err := tokenMaker.CreateAuthToken(user.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		cookie := &http.Cookie{
			Name:     "authToken",
			Value:    authToken,
			Path:     "/",
			Expires:  time.Now().Add(15 * time.Minute),
			HttpOnly: true,
			Secure:   r.TLS != nil, // Set only on HTTPS
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)

		// log.Printf("++ Auth: Writing user struct to request context.")
		ctx := context.WithValue(r.Context(), AuthUser, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func IsAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(AuthUser).(*types.User)
		if !ok {
			log.Printf("### Admin: Cannot retrieve user struct from request context")
			writeUnauthed(w)
			return
		}
		log.Printf("### Admin: Retrieved user %d struct from request context", user.ID)
		if !user.IsAdmin {
			log.Printf("### Admin: User %d is not an admin", user.ID)
			writeUnauthed(w)
			return
		}
		log.Printf("### Admin: User %d is an admin", user.ID)
		next.ServeHTTP(w, r)
	})
}

// func IsAuthenticated(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		authorization := r.Header.Get("Authorization")

// 		if !strings.HasPrefix(authorization, "Bearer ") {
// 			writeUnauthed(w)
// 			return
// 		}

// 		encodedToken := strings.TrimPrefix(authorization, "Bearer ")

// 		token, err := base64.StdEncoding.DecodeString(encodedToken)
// 		if err != nil {
// 			writeUnauthed(w)
// 			return
// 		}

// 		userID := string(token)
// 		ctx := context.WithValue(r.Context(), AuthUserID, userID)
// 		req := r.WithContext(ctx)

// 		next.ServeHTTP(w, req)
// 	})
// }
