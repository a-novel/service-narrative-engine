package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"iter"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// streamEvent is a server-sent event as a test reads it, its data decoded from JSON.
type streamEvent struct {
	ID   string
	Type string
	Data any
}

// sequence yields items in order, then err when it is not nil.
func sequence[Item any](items []Item, err error) iter.Seq2[Item, error] {
	return func(yield func(Item, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}

		if err != nil {
			var zero Item

			yield(zero, err)
		}
	}
}

// requireEvents checks that a handler answered an event stream holding the expected events, in order.
func requireEvents(t *testing.T, response *http.Response, expect []streamEvent) {
	t.Helper()

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))

	data, err := io.ReadAll(response.Body)
	require.NoError(t, errors.Join(err, response.Body.Close()))

	var events []streamEvent

	for frame := range strings.SplitSeq(strings.TrimSuffix(string(data), "\n\n"), "\n\n") {
		if frame == "" {
			continue
		}

		var event streamEvent

		for line := range strings.SplitSeq(frame, "\n") {
			field, value, _ := strings.Cut(line, ": ")

			switch field {
			case "id":
				event.ID = value
			case "event":
				event.Type = value
			case "data":
				require.NoError(t, json.Unmarshal([]byte(value), &event.Data))
			}
		}

		events = append(events, event)
	}

	require.Equal(t, expect, events)
}
