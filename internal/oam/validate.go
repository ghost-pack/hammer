package oam

import (
	"fmt"
	"strings"
)

type ValidationErrors []FieldError

type FieldError struct {
	Field string
	Msg   string
}

func (e FieldError) Error() string {
	return e.Field + ": " + e.Msg
}

func (es *ValidationErrors) Add(field, msg string) {
	*es = append(*es, FieldError{Field: field, Msg: msg})
}

func (es ValidationErrors) Error() string {
	if len(es) == 0 {
		return "no validation errors"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d validation error(s):\n", len(es)))
	for _, e := range es {
		b.WriteString("  - ")
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return b.String()
}

func Validate(app *App) error {
	var errs ValidationErrors

	if app.Kind == "" {
		errs.Add("kind", "is required")
	}

	if app.Metadata.Name == "" {
		errs.Add("name", "is required")
	}

	if len(app.Metadata.Name) > 63 {
		errs.Add("name", "name must be less than 63 characters")
	}

	if app.Metadata.Annotations["team"] == "" {
		errs.Add("team", "is required")
	}

	for _, component := range app.Spec.Components {
		if component.Name == "" {
			errs.Add("component", "name is required")
		}
		if component.Name == "" {
			errs.Add("component", "name is required")
		}
		if len(component.Name) > 63 {
			errs.Add("component", "name is too long")
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
