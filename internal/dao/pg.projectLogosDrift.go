package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.projectLogosDrift.sql
var projectLogosDriftQuery string

// ProjectLogosDriftRequest is the input to [PgProjectLogosDrift.Exec].
type ProjectLogosDriftRequest struct {
	// OwnerID is the user whose logos to compare.
	OwnerID uuid.UUID

	// Logos maps the ID of each project whose logo a client holds to the checksum it holds, the SHA-256 of the
	// logo's bytes in lowercase hex.
	Logos map[string]string

	// Limit caps the rows.
	Limit int
}

// ProjectLogosDriftRow is one drift: a logo of the owner that drifted, or a held ID whose project holds no logo of
// the owner.
type ProjectLogosDriftRow struct {
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`

	// Checksum is the SHA-256 of the drifted logo's bytes in lowercase hex, nil on a removed row.
	Checksum *string `bun:"checksum"`
}

// PgProjectLogosDrift compares the logos a client holds with those of an owner's projects, by their checksum, in
// one query. It returns every logo of the owner the client holds with another checksum or not at all, and every
// held ID whose project is not the owner's or holds no logo.
type PgProjectLogosDrift struct{}

func NewPgProjectLogosDrift() *PgProjectLogosDrift {
	return &PgProjectLogosDrift{}
}

func (dao *PgProjectLogosDrift) Exec(
	ctx context.Context, request *ProjectLogosDriftRequest,
) ([]*ProjectLogosDriftRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgProjectLogosDrift")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*ProjectLogosDriftRow

	err = db.NewRaw(projectLogosDriftQuery, request.OwnerID, request.Logos, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("logos.drifted", len(entities)))

	return entities, nil
}
