package middlewares

import (
	"log"
	"net/http"
	"time"

	"github.com/forrest-bajbek/bombs/utils"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := utils.GenerateRandomHexToken(8)
		start := time.Now()
		log.Println("#### START", requestID, r.Method, r.URL.Path)
		// wrapped := &wrappedWriter{
		// 	ResponseWriter: w,
		// 	statusCode:     http.StatusOK,
		// }
		// next.ServeHTTP(wrapped, r)
		// log.Println("#### STOP", requestID, wrapped.statusCode, r.Method, r.URL.Path, time.Since(start))
		next.ServeHTTP(w, r)
		log.Println("#### STOP", requestID, r.Method, r.URL.Path, time.Since(start))
	})
}
