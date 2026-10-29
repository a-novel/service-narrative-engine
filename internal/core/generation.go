package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

const (
	// generationPurpose is the purpose genai runs and caps every generation of the service under.
	generationPurpose = "studio.generation"
	// generationProvider is the provider genai sends every call to.
	generationProvider = "openai"
)

// GenerationQuotaError is returned by [GenerationSubmit] when genai admits none of a generation's calls. It
// is [ErrGenerationQuota], and names the cap genai refused the first call for.
type GenerationQuotaError struct {
	// Cap is genai's message naming the cap, such as its concurrency or a token window.
	Cap string
}

func (err *GenerationQuotaError) Error() string {
	return ErrGenerationQuota.Error() + ": " + err.Cap
}

// Is reports the error as [ErrGenerationQuota].
func (err *GenerationQuotaError) Is(target error) bool {
	return target == ErrGenerationQuota
}

// genaiTimeout bounds each call to genai, which answers a submit within 20 seconds and a read within 10.
const genaiTimeout = 25 * time.Second

// genaiError joins a sentinel to an error genai returned: [ErrGenaiUnavailable] when genai could not take
// the call, and the sentinel of a miss when it holds no such generation.
func genaiError(err, notFound error) error {
	switch status.Code(err) {
	case codes.Unavailable:
		return errors.Join(err, ErrGenaiUnavailable)
	case codes.NotFound:
		return errors.Join(err, notFound)
	default:
		return err
	}
}

const (
	// genaiTierDefault is the tier genai caps a user's generations by.
	genaiTierDefault = "default"
	// genaiTierAdmin is the tier of an admin, which genai does not cap.
	genaiTierAdmin = "admin"
)

const (
	// GenerationStatusPending is a generation genai holds and has not started yet.
	GenerationStatusPending = "pending"
	// GenerationStatusRunning is a generation the provider runs.
	GenerationStatusRunning = "running"
	// GenerationStatusSucceeded is a generation whose output passed the output schema of its kind.
	GenerationStatusSucceeded = "succeeded"
	// GenerationStatusFailed is a generation that ended without a valid output.
	GenerationStatusFailed = "failed"
	// GenerationStatusCancelled is a generation cancelled before it settled.
	GenerationStatusCancelled = "cancelled"
)

// GenerationStatuses lists where a genai generation may stand.
var GenerationStatuses = []string{
	GenerationStatusPending, GenerationStatusRunning, GenerationStatusSucceeded, GenerationStatusFailed,
	GenerationStatusCancelled,
}

// Generation is a generation as submit leaves it: each of its calls with its genai generation, or with
// the cap it waits on.
type Generation struct {
	// Kind is the kind of the generation, one of [GenerationKinds].
	Kind string
	// Calls lists the generation's model calls, in the order of its context plan.
	Calls []*GenerationCall
}

// GenerationCall is one model call of a generation.
type GenerationCall struct {
	// Result is the genai generation that runs the call. It is nil while the call waits.
	Result *GenerationResult
	// Waiting names the cap genai refused the call for. It is empty once genai admitted the call.
	Waiting string
	// Context says what the call reads, with the handles that map its output back.
	Context *ContextCall
}

// GenerationResult is one genai generation, as a submit, a read or a cancel finds it.
type GenerationResult struct {
	ID uuid.UUID
	// Status is where the generation stands, one of [GenerationStatuses].
	Status string

	// Error says why a failed generation failed: genai's reason, or why its output was refused.
	Error string
	// Output is the model's output, which passed the full output schema of its kind. It is empty unless
	// the generation succeeded.
	Output json.RawMessage
}

// errGenaiStatus is returned when genai reports a status the service does not know.
var errGenaiStatus = errors.New("unknown genai status")

// genaiStatuses maps each genai status to where a generation stands.
var genaiStatuses = map[servicegenai.GenerationStatus]string{
	servicegenai.GenerationStatusPending:   GenerationStatusPending,
	servicegenai.GenerationStatusRunning:   GenerationStatusRunning,
	servicegenai.GenerationStatusSucceeded: GenerationStatusSucceeded,
	servicegenai.GenerationStatusFailed:    GenerationStatusFailed,
	servicegenai.GenerationStatusCancelled: GenerationStatusCancelled,
}

// newGenerationResult reads a genai generation. A generation genai reports succeeded fails here when the
// provider's response holds no valid output.
func newGenerationResult(generation *servicegenai.Generation) (*GenerationResult, error) {
	id, err := uuid.Parse(generation.GetId())
	if err != nil {
		return nil, fmt.Errorf("parse genai id: %w", err)
	}

	status, ok := genaiStatuses[generation.GetStatus()]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errGenaiStatus, generation.GetStatus())
	}

	// genai also says why a cancelled generation stopped, which its status already tells.
	result := &GenerationResult{ID: id, Status: status}
	if status == GenerationStatusFailed {
		result.Error = generation.GetError()
	}

	if status != GenerationStatusSucceeded {
		return result, nil
	}

	attempts := generation.GetAttempts()
	if len(attempts) == 0 {
		return &GenerationResult{ID: id, Status: GenerationStatusFailed, Error: "no response"}, nil
	}

	output, reason := readOutput(attempts[len(attempts)-1].GetResponse())
	if reason != "" {
		return &GenerationResult{ID: id, Status: GenerationStatusFailed, Error: reason}, nil
	}

	result.Output = output

	return result, nil
}

// openaiResponse is the part of an OpenAI Responses object a read checks: whether the model finished,
// what it wrote, and the output format the call asked for, named after the kind of the generation.
//
//nolint:tagliatelle // OpenAI's own field names.
type openaiResponse struct {
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
	Text struct {
		Format struct {
			Name string `json:"name"`
		} `json:"format"`
	} `json:"text"`
}

// readOutput returns the output a provider's response holds once it passed the full output schema of its
// kind. It returns instead the reason the response holds no valid output: cut short, refused or malformed.
func readOutput(raw []byte) (json.RawMessage, string) {
	response := &openaiResponse{}

	err := json.Unmarshal(raw, response)
	if err != nil {
		return nil, "malformed response: " + err.Error()
	}

	if response.IncompleteDetails != nil {
		return nil, "incomplete output: " + response.IncompleteDetails.Reason
	}

	if response.Status != "completed" {
		return nil, "response " + response.Status
	}

	var text string

	for _, item := range response.Output {
		for _, content := range item.Content {
			switch content.Type {
			case "refusal":
				return nil, "refused: " + content.Refusal
			case "output_text":
				text += content.Text
			}
		}
	}

	schema, err := schemas.Output(response.Text.Format.Name)
	if err != nil {
		return nil, "unknown output format: " + response.Text.Format.Name
	}

	err = schema.Validate(json.RawMessage(text))
	if err != nil {
		return nil, "malformed output: " + err.Error()
	}

	return json.RawMessage(text), ""
}
