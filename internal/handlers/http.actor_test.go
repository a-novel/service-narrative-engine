package handlers_test

import (
	"net/http"

	"github.com/google/uuid"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

type testClaimsState uint8

const (
	testClaimsValid testClaimsState = iota
	testClaimsAnonymous
	testClaimsMissing
)

var testActor = core.Actor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000042")}

func withTestClaims(request *http.Request, state testClaimsState) *http.Request {
	if state == testClaimsMissing {
		return request
	}

	claims := &serviceauthentication.Claims{}
	if state == testClaimsValid {
		claims.UserID = &testActor.UserID
	}

	return request.WithContext(serviceauthentication.SetClaimsContext(request.Context(), claims))
}
