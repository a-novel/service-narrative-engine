package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/otel"
)

type Ping struct{}

func NewPing() *Ping {
	return new(Ping)
}

func (handler *Ping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer().Start(r.Context(), "rest.Ping")
	defer span.End()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte("pong"))
	if err != nil {
		_ = otel.ReportError(span, err)

		return
	}

	otel.ReportSuccessNoContent(span)
}
