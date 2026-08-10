package oam

import (
	"fmt"
	"regexp"
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
	*es = append(*es, FieldError{
		Field: field,
		Msg:   msg,
	})
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

var namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func Validate(app *App) error {
	var errs ValidationErrors

	if app.Kind == "" {
		errs.Add("kind", "is required")
	} else if app.Kind != "Application" {
		errs.Add("kind", "must be Application")
	}

	if app.Metadata.Name == "" {
		errs.Add("metadata.name", "is required")
	} else {
		validateName(&errs, "metadata.name", app.Metadata.Name)
	}

	if app.Metadata.Annotations["team"] == "" {
		errs.Add("metadata.annotations.team", "is required")
	}

	// Components.
	componentNames := make(map[string]struct{})

	for i, component := range app.Spec.Components {
		field := fmt.Sprintf("spec.components[%d]", i)

		if component.Name == "" {
			errs.Add(field+".name", "is required")
		} else {
			validateName(&errs, field+".name", component.Name)

			if _, exists := componentNames[component.Name]; exists {
				errs.Add(
					field+".name",
					fmt.Sprintf("duplicate component name %q", component.Name),
				)
			}

			componentNames[component.Name] = struct{}{}
		}

		if component.Type == "" {
			errs.Add(field+".type", "is required")
		}
	}

	// Policies are optional.
	policyNames := make(map[string]struct{})
	environments := make(map[string]string)

	for i, policy := range app.Spec.Policies {
		field := fmt.Sprintf("spec.policies[%d]", i)

		if policy.Name == "" {
			errs.Add(field+".name", "is required")
		} else {
			validateName(&errs, field+".name", policy.Name)

			if _, exists := policyNames[policy.Name]; exists {
				errs.Add(
					field+".name",
					fmt.Sprintf("duplicate policy name %q", policy.Name),
				)
			}

			policyNames[policy.Name] = struct{}{}
		}

		if policy.Type == "" {
			errs.Add(field+".type", "is required")
			continue
		}

		switch policy.Type {
		case "environment":
			validateEnvironmentPolicy(
				&errs,
				field,
				policy,
				componentNames,
				environments,
			)

		default:
			errs.Add(
				field+".type",
				fmt.Sprintf("unsupported policy type %q", policy.Type),
			)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateEnvironmentPolicy(
	errs *ValidationErrors,
	field string,
	policy Policy,
	componentNames map[string]struct{},
	environments map[string]string,
) {
	environment := policy.Properties.Environment

	if environment == "" {
		errs.Add(
			field+".properties.environment",
			"is required",
		)
	} else {
		validateName(
			errs,
			field+".properties.environment",
			environment,
		)

		if existingPolicy, exists := environments[environment]; exists {
			errs.Add(
				field+".properties.environment",
				fmt.Sprintf(
					"environment %q is already defined by policy %q",
					environment,
					existingPolicy,
				),
			)
		}

		environments[environment] = policy.Name
	}

	overrideComponents := make(map[string]struct{})

	for i, override := range policy.Properties.Overrides {
		overrideField := fmt.Sprintf(
			"%s.properties.overrides[%d]",
			field,
			i,
		)

		if override.Component == "" {
			errs.Add(
				overrideField+".component",
				"is required",
			)
			continue
		}

		if _, exists := componentNames[override.Component]; !exists {
			errs.Add(
				overrideField+".component",
				fmt.Sprintf(
					"references unknown component %q",
					override.Component,
				),
			)
		}

		if _, exists := overrideComponents[override.Component]; exists {
			errs.Add(
				overrideField+".component",
				fmt.Sprintf(
					"duplicate override for component %q",
					override.Component,
				),
			)
		}

		overrideComponents[override.Component] = struct{}{}
	}
}

func validateName(errs *ValidationErrors, field, value string) {
	if len(value) > 63 {
		errs.Add(field, "must be 63 characters or fewer")
	}

	if !namePattern.MatchString(value) {
		errs.Add(
			field,
			"must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number",
		)
	}
}
