package core

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

var (
	// ErrNoGenerationCaps is returned by [GenerationCapsPolicy.Resolve] when none of a caller's roles sets
	// generation caps.
	ErrNoGenerationCaps = errors.New("no generation caps")
	// ErrInvalidGenerationCaps is returned by [NewGenerationCapsPolicy] for caps that name a role the
	// permissions lack, or that leave a role writing projects without caps.
	ErrInvalidGenerationCaps = errors.New("invalid generation caps")
)

// GenerationCaps limit a user's generations on genai: how many calls run at once, and the tokens the calls
// consume over rolling windows. genai sets no limit of its own, so empty caps limit nothing.
type GenerationCaps struct {
	// Concurrency caps the calls the user may have in flight at once. Nil sets no limit.
	Concurrency *int32
	// Windows cap the tokens the user's calls consume over rolling windows, shortest first.
	Windows []*GenerationCapWindow
}

// GenerationCapWindow caps the tokens a user's calls consume over one rolling window.
type GenerationCapWindow struct {
	// Duration is how far back the window reaches.
	Duration time.Duration

	// InputTokens caps the input tokens, cached ones included. Nil sets no limit.
	InputTokens *int64
	// OutputTokens caps the output tokens, reasoning ones included. Nil sets no limit.
	OutputTokens *int64
}

// genai returns the caps as a submit to genai carries them.
func (caps *GenerationCaps) genai() *servicegenai.GenerationCaps {
	windows := make([]*servicegenai.GenerationCapWindow, len(caps.Windows))

	for i, window := range caps.Windows {
		windows[i] = &servicegenai.GenerationCapWindow{
			Duration:     durationpb.New(window.Duration),
			InputTokens:  window.InputTokens,
			OutputTokens: window.OutputTokens,
		}
	}

	return &servicegenai.GenerationCaps{Concurrency: caps.Concurrency, Windows: windows}
}

// GenerationCapsPolicy gives each caller the caps of their roles. A caller whose roles grant
// [config.PermissionGenerationUncapped] takes empty caps. Any other caller takes the caps of their
// highest-priority role that the caps name, among the roles they hold and those they inherit.
type GenerationCapsPolicy struct {
	// roles maps each role of the permissions to the caps its holders take. A role absent from it sets no
	// caps.
	roles map[string]*rankedCaps
}

// rankedCaps are the caps of a role, with the priority that ranks them against a caller's other roles.
type rankedCaps struct {
	caps     *GenerationCaps
	priority int
}

// uncappedCaps are the empty caps of a role granting [config.PermissionGenerationUncapped]. They outrank
// every other role's.
var uncappedCaps = &rankedCaps{caps: &GenerationCaps{}, priority: math.MaxInt}

// NewGenerationCapsPolicy resolves the caps of every role of the permissions. It returns
// [ErrInvalidGenerationCaps] when the caps name a role the permissions lack, or leave a role that grants
// [config.PermissionProjectWrite] without caps. No writer then submits without caps.
func NewGenerationCapsPolicy(
	permissions serviceauthentication.Permissions, caps config.Caps,
) (*GenerationCapsPolicy, error) {
	named := make(map[string]*rankedCaps, len(caps.Roles))

	for name, roleCaps := range caps.Roles {
		priority, err := permissions.Priority(name)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidGenerationCaps, err)
		}

		named[name] = &rankedCaps{caps: newGenerationCaps(roleCaps), priority: priority}
	}

	for name, role := range permissions.Roles {
		if slices.Contains(role.Permissions, config.PermissionGenerationUncapped) {
			named[name] = uncappedCaps
		}
	}

	policy := &GenerationCapsPolicy{roles: make(map[string]*rankedCaps, len(permissions.Roles))}

	for name := range permissions.Roles {
		held := heldRoles(permissions, name)

		ranked := highestCaps(named, held)
		if ranked != nil {
			policy.roles[name] = ranked

			continue
		}

		writes := slices.ContainsFunc(held, func(role string) bool {
			return slices.Contains(permissions.Roles[role].Permissions, config.PermissionProjectWrite)
		})
		if writes {
			return nil, fmt.Errorf("%w: role %q writes projects without caps", ErrInvalidGenerationCaps, name)
		}
	}

	return policy, nil
}

// Resolve returns the caps of a caller holding roles, or [ErrNoGenerationCaps] when none of the roles sets
// any.
func (policy *GenerationCapsPolicy) Resolve(roles []string) (*GenerationCaps, error) {
	ranked := highestCaps(policy.roles, roles)
	if ranked == nil {
		return nil, ErrNoGenerationCaps
	}

	return ranked.caps, nil
}

// newGenerationCaps reads the caps of a role, with their windows shortest first.
func newGenerationCaps(roleCaps config.RoleCaps) *GenerationCaps {
	caps := &GenerationCaps{Concurrency: roleCaps.Concurrency}

	for _, duration := range slices.Sorted(maps.Keys(roleCaps.Windows)) {
		window := roleCaps.Windows[duration]
		caps.Windows = append(caps.Windows, &GenerationCapWindow{
			Duration: duration, InputTokens: window.InputTokens, OutputTokens: window.OutputTokens,
		})
	}

	return caps
}

// heldRoles returns a role and every role it inherits, directly or through another.
func heldRoles(permissions serviceauthentication.Permissions, name string) []string {
	var held []string

	seen := map[string]bool{}

	for stack := []string{name}; len(stack) > 0; {
		role := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if seen[role] {
			continue
		}

		seen[role] = true
		held = append(held, role)
		stack = append(stack, permissions.Roles[role].Inherits...)
	}

	return held
}

// highestCaps returns the caps of the highest-priority role among roles that byRole maps, the first one
// on a tie, or nil when it maps none.
func highestCaps(byRole map[string]*rankedCaps, roles []string) *rankedCaps {
	var best *rankedCaps

	for _, role := range roles {
		candidate := byRole[role]
		if candidate != nil && (best == nil || candidate.priority > best.priority) {
			best = candidate
		}
	}

	return best
}
