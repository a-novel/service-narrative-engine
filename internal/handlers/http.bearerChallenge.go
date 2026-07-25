package handlers

import "net/http"

// #nosec G101 -- This is an HTTP authentication challenge, not a credential.
const bearerChallenge = `Bearer realm="narrative-engine", error="invalid_token"`

type challengeWriter struct {
	http.ResponseWriter
}

func (writer *challengeWriter) WriteHeader(status int) {
	if status == http.StatusUnauthorized {
		writer.Header().Set("WWW-Authenticate", bearerChallenge)
	}

	writer.ResponseWriter.WriteHeader(status)
}

// BearerChallenge adds the RFC 6750 challenge to unauthorized responses.
func BearerChallenge(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&challengeWriter{ResponseWriter: w}, r)
	})
}
