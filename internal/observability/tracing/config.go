package tracing

import (
	"os"
	"strconv"
	"strings"
)

// ExporterKind identifies an exporter implementation.
type ExporterKind string

const (
	ExporterStdout ExporterKind = "stdout"
	ExporterOTLP   ExporterKind = "otlp"
	ExporterGCP    ExporterKind = "gcp"
	ExporterNone   ExporterKind = "none"
)

// Config controls tracing behavior.
type Config struct {
	// Exporter selects the span destination. See ExporterKind constants.
	Exporter ExporterKind

	// ServiceName is the OTel resource service.name attribute.
	ServiceName string

	// ServiceVersion is the OTel resource service.version attribute.
	ServiceVersion string

	// Environment is the deployment environment (prod/staging/dev/local).
	Environment string

	// Commit is the git commit SHA, included as a resource attribute.
	Commit string

	// SampleRatio is the fraction of traces to sample (0.0..1.0).
	// 1.0 = always sample, 0.0 = never. Default 1.0 (CLI use is low-volume).
	SampleRatio float64

	// OTLPEndpoint is the OTLP gRPC endpoint when Exporter == ExporterOTLP.
	// Defaults to localhost:4317. Set OTEL_EXPORTER_OTLP_ENDPOINT to override.
	OTLPEndpoint string

	// OTLPInsecure disables TLS for OTLP. Defaults to true (local dev).
	OTLPInsecure bool

	// GCPProject is the project ID for the Cloud Trace exporter.
	// Required when Exporter == ExporterGCP. Read from GOOGLE_CLOUD_PROJECT.
	GCPProject string
}

// ConfigFromEnv builds a Config from environment variables and sensible defaults.
//
// Environment variables consulted:
//   - OTEL_EXPORTER:                 stdout | otlp | gcp | none (default: auto)
//   - OTEL_SERVICE_NAME:             default "hammer"
//   - OTEL_SERVICE_VERSION:          default from build-time injection
//   - DEPLOY_ENV:                    prod/staging/dev/local (default: local)
//   - GIT_COMMIT:                    default ""
//   - OTEL_TRACES_SAMPLER_ARG:       float 0..1 (default 1.0)
//   - OTEL_EXPORTER_OTLP_ENDPOINT:   default localhost:4317
//   - OTEL_EXPORTER_OTLP_INSECURE:   bool (default true)
//   - GOOGLE_CLOUD_PROJECT / GCP_PROJECT
func ConfigFromEnv() Config {
	c := Config{
		ServiceName:    envOr("OTEL_SERVICE_NAME", "hammer"),
		ServiceVersion: envOr("OTEL_SERVICE_VERSION", "dev"),
		Environment:    envOr("DEPLOY_ENV", "local"),
		Commit:         os.Getenv("GIT_COMMIT"),
		SampleRatio:    envFloat("OTEL_TRACES_SAMPLER_ARG", 1.0),
		OTLPEndpoint:   envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTLPInsecure:   envBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		GCPProject:     firstNonEmpty(os.Getenv("GOOGLE_CLOUD_PROJECT"), os.Getenv("GCP_PROJECT")),
	}

	c.Exporter = pickExporter(os.Getenv("OTEL_EXPORTER"), c)
	return c
}

// pickExporter selects an exporter based on explicit setting or auto-detection.
func pickExporter(explicit string, c Config) ExporterKind {
	if explicit != "" {
		return ExporterKind(strings.ToLower(explicit))
	}
	// Auto-detect:
	//   - GCP project set + running in GCP-like env → GCP
	//   - OTLP endpoint set explicitly → OTLP
	//   - otherwise → stdout (local dev)
	if c.GCPProject != "" && (os.Getenv("K_SERVICE") != "" || os.Getenv("CLOUD_RUN_JOB") != "" || os.Getenv("BUILD_ID") != "") {
		return ExporterGCP
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		return ExporterOTLP
	}
	return ExporterStdout
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
