Add these to get local Jaeger going (run with docker compose):
```aiignore
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER=otlp
```