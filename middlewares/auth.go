package middlewares

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/forrest-bajbek/bombs/routes"
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

// redirectToLogin clears the auth cookie and sends the client to the login
// page, carrying the page they were on as ?next= where that makes sense.
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:   "authToken",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	// htmx would follow a 302 and swap the login page into the fragment
	// target, so ask it for a full-page navigation instead. next is the
	// page the user is on, not the partial being requested.
	if r.Header.Get("HX-Request") == "true" {
		next := ""
		if u, err := url.Parse(r.Header.Get("HX-Current-URL")); err == nil {
			next = u.RequestURI()
		}
		w.Header().Set("HX-Redirect", routes.LoginURL(next))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only top-level GET navigations make useful next targets; image loads,
	// the SSE stream and form POSTs go to the bare login page.
	fetchMode := r.Header.Get("Sec-Fetch-Mode")
	if r.Method == http.MethodGet && (fetchMode == "navigate" || fetchMode == "") {
		http.Redirect(w, r, routes.LoginURL(r.URL.RequestURI()), http.StatusFound)
		return
	}
	http.Redirect(w, r, routes.LoginURL(""), http.StatusFound)
}

func IsAuthenticated(service *services.Service, tokenMaker *token.JWTMaker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(AuthUserID).(int)
		if !ok {
			// log.Printf("++ Auth: userID not found in request context.")
			// log.Printf("++ Auth: Removing cookie and redirecting to login.")
			redirectToLogin(w, r)
			return
		}
		// log.Printf("++ Auth: userID found in request context: %d", userID)
		user, err := service.GetUserByID(userID)
		if err != nil {
			// log.Printf("++ Auth: userID %d not found in databaes", userID)
			// log.Printf("++ Auth: Removing cookie and redirecting to login.")
			redirectToLogin(w, r)
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
			MaxAge:   int(time.Now().Add(15 * time.Minute).Unix()),
			HttpOnly: true,
			Secure:   os.Getenv("ENV") == "PROD",
			SameSite: http.SameSiteStrictMode,
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
			// log.Printf("### Admin: Cannot retrieve user struct from request context")
			writeUnauthed(w)
			return
		}
		// log.Printf("### Admin: Retrieved user %d struct from request context", user.ID)
		if !user.IsAdmin {
			// log.Printf("### Admin: User %d is not an admin", user.ID)
			writeUnauthed(w)
			return
		}
		// log.Printf("### Admin: User %d is an admin", user.ID)
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
