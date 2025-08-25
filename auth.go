package main

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// basicAuthMiddleware wraps HTTP handlers with basic authentication
func basicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Tiny CRM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user from database
		user, err := repo.GetUserByUsername(username)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Tiny CRM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Tiny CRM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Authentication successful, call the next handler
		next(w, r)
	}
}

// hashPassword creates a bcrypt hash of the password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}