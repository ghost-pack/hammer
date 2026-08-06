package cloudbuild

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ghost-pack/hammer/internal/observability/tracing"
	"github.com/santhosh-tekuri/jsonschema/v6"
	otelCodes "go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
)

//go:embed schema/cloudbuild.json
var cloudBuildSchema []byte

func (p *Pipeline) lint(ctx context.Context) error {
	ctx, span := tracing.Tracer("cloudbuildlint").Start(ctx, "cloudbuildlint")
	defer span.End()

	var props properties
	if err := p.component.Properties.Decode(&props); err != nil {
		errorDecodingProperties := fmt.Errorf("decoding properties: %w", err)
		span.RecordError(errorDecodingProperties)
		span.SetStatus(otelCodes.Error, errorDecodingProperties.Error())
		return errorDecodingProperties
	}
	if props.Path == "" {
		props.Path = "./cloudbuild.yaml"
	}

	raw, err := os.ReadFile(props.Path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelCodes.Error, err.Error())
		return err
	}

	var doc interface{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}

	var schemaDoc any
	if err := json.Unmarshal(cloudBuildSchema, &schemaDoc); err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("cloudbuild.json", schemaDoc); err != nil {
		errorLoadingCloudBuildSchema := fmt.Errorf("adding schema resource: %w", err)
		span.RecordError(errorLoadingCloudBuildSchema)
		span.SetStatus(otelCodes.Error, errorLoadingCloudBuildSchema.Error())
		return errorLoadingCloudBuildSchema
	}

	schema, err := compiler.Compile("cloudbuild.json")
	if err != nil {
		cloudBuildSchemaCompilationFailure := fmt.Errorf("compiling schema: %w", err)
		span.RecordError(cloudBuildSchemaCompilationFailure)
		span.SetStatus(otelCodes.Error, cloudBuildSchemaCompilationFailure.Error())
		return cloudBuildSchemaCompilationFailure
	}
	schemaValidationError := schema.Validate(doc)
	if schemaValidationError != nil {
		span.RecordError(schemaValidationError)
		span.SetStatus(otelCodes.Error, schemaValidationError.Error())
		return schemaValidationError
	}
	return nil
}
