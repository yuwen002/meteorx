package config

import (
	"fmt"
	"os"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s: %s", e.Field, e.Message)
}

type Validator struct {
	errors []ValidationError
}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Require(field, value, msg string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{Field: field, Message: msg})
	}
	return v
}

func (v *Validator) RequireEnv(field, envKey, msg string) *Validator {
	if os.Getenv(envKey) == "" {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s (env: %s)", msg, envKey),
		})
	}
	return v
}

func (v *Validator) MustValid() error {
	if len(v.errors) > 0 {
		var msgs []string
		for _, e := range v.errors {
			msgs = append(msgs, e.Error())
		}
		return fmt.Errorf("%d validation error(s): %s", len(v.errors), strings.Join(msgs, "; "))
	}
	return nil
}

func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}
