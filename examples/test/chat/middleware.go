package chat

import (
	"context"
	"log"
	"net/http"

	"github.com/colevoss/gosock/examples/test/db"
)

type Middleware struct {
	db *db.Db
}

func NewMiddleware(db *db.Db) *Middleware {
	return &Middleware{db}
}

func (m *Middleware) UserMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		passwordHeader := r.Header.Get("Auth")

		if passwordHeader == "" {
			log.Printf("Auth header not included")
			w.WriteHeader(404)
			return
		}

		user := m.db.FindUserByPassword(passwordHeader)

		if user == nil {
			log.Printf("User not found %s", passwordHeader)
			w.WriteHeader(404)
			return
		}

		log.Printf("Found user %+v", user)

		ctx := context.WithValue(r.Context(), "userId", user.Id)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserId(ctx context.Context) (string, bool) {
	id := ctx.Value("userId")

	if id == nil {
		return "", false
	}

	return id.(string), true
}
