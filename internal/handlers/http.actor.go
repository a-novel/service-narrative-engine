package handlers

import (
	"context"
	"errors"
	"fmt"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ErrAnonymousActor means authenticated claims do not identify a user.
var ErrAnonymousActor = errors.New("anonymous actor")

func actorFromContext(ctx context.Context) (*core.Actor, error) {
	ctx, span := otel.Tracer().Start(ctx, "rest.ActorFromContext")
	defer span.End()

	claims, err := serviceauthentication.MustGetClaimsContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get actor claims: %w", err))
	}

	if claims.UserID == nil {
		return nil, otel.ReportError(span, ErrAnonymousActor)
	}

	return otel.ReportSuccess(span, &core.Actor{UserID: *claims.UserID}), nil
}
