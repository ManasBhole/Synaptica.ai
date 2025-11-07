package ingestion

import (
	"errors"
	"fmt"
	"strings"

	"github.com/synaptica-ai/platform/pkg/common/models"
)

var (
	errInvalidSource = errors.New("invalid source")
	errEmptyData     = errors.New("missing data payload")
	errInvalidFormat = errors.New("invalid format")
)

type ValidationError struct {
	reason error
}

func (e ValidationError) Error() string {
	return e.reason.Error()
}

func (e ValidationError) Unwrap() error {
	return e.reason
}

func IsValidationError(err error) bool {
	var ve ValidationError
	return errors.As(err, &ve)
}

type Validator struct {
	allowedSources map[string]struct{}
	allowedFormats map[string]struct{}
}

func NewValidator(sources, formats []string) *Validator {
	vs := make(map[string]struct{})
	for _, src := range sources {
		if trimmed := strings.TrimSpace(strings.ToLower(src)); trimmed != "" {
			vs[trimmed] = struct{}{}
		}
	}

	vf := make(map[string]struct{})
	for _, f := range formats {
		if trimmed := strings.TrimSpace(strings.ToLower(f)); trimmed != "" {
			vf[trimmed] = struct{}{}
		}
	}

	return &Validator{allowedSources: vs, allowedFormats: vf}
}

func (v *Validator) Validate(req models.IngestRequest) error {
	if v == nil {
		return ValidationError{reason: errors.New("validator not initialised")}
	}

	source := strings.TrimSpace(strings.ToLower(req.Source))
	if source == "" {
		return ValidationError{reason: fmt.Errorf("source required: %w", errInvalidSource)}
	}
	if len(v.allowedSources) > 0 {
		if _, ok := v.allowedSources[source]; !ok {
			return ValidationError{reason: fmt.Errorf("source '%s' not allowed: %w", source, errInvalidSource)}
		}
	}

	format := strings.TrimSpace(strings.ToLower(req.Format))
	if format == "" {
		return ValidationError{reason: fmt.Errorf("format required: %w", errInvalidFormat)}
	}
	if len(v.allowedFormats) > 0 {
		if _, ok := v.allowedFormats[format]; !ok {
			return ValidationError{reason: fmt.Errorf("format '%s' not supported: %w", format, errInvalidFormat)}
		}
	}

	if req.Data == nil || len(req.Data) == 0 {
		return ValidationError{reason: errEmptyData}
	}

	return nil
}
