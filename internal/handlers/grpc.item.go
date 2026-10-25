package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-template/internal/core"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

// itemToProto converts a core item into its protobuf form, encoding timestamps
// as RFC 3339 strings that keep their fractional seconds.
func itemToProto(item *core.Item) *templatev0.Item {
	return &templatev0.Item{
		Id:          item.ID.String(),
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// parseOptionalItemID parses an item ID that a request may leave empty. Empty parses to uuid.Nil.
func parseOptionalItemID(id string) (uuid.UUID, error) {
	if id == "" {
		return uuid.Nil, nil
	}

	return uuid.Parse(id)
}
