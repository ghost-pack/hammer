package ci

import "time"

// CIPubSubMessage is published by CI to trigger CD.
// It follows the Routing Slip EIP pattern — each CD stage
// pops the first destination from the slip and publishes
// to the next one upon completion.
type CIPubSubMessage struct {
	// Identity
	Tenant      string    `json:"tenant"`
	CommitSha   string    `json:"commitSha"`
	Branch      string    `json:"branch"`
	PublishedAt time.Time `json:"publishedAt"`
	Env         string    `json:"env"`

	// OAM file for this commit — CD reads desired state from here
	OAMPath     string `json:"oamPath"`
	TraceParent string `json:"traceParent"`
	Reconcile   bool   `json:"reconcile"`
	Traceparent string `json:"traceparent,omitempty"`

	// Artifacts produced by CI, keyed by component name
	Artifacts map[string]Artifact `json:"artifacts"`

	// Routing slip — CD pops the first entry and publishes
	// to it upon successful completion, enabling dev→prod promotion
	RoutingSlip []RoutingSlipEntry `json:"routingSlip"`
}

// Artifact describes a deployable artifact produced by CI
type Artifact struct {
	Type       ArtifactType      `json:"type"`
	Properties map[string]string `json:"properties"`
}

type ArtifactType string

const (
	ArtifactTypeCloudRun   ArtifactType = "cloud-run"
	ArtifactTypeOpenTofu   ArtifactType = "opentofu"
	ArtifactTypeCloudBuild ArtifactType = "cloudbuild"
)

// RoutingSlipEntry describes the next CD stage to trigger
type RoutingSlipEntry struct {
	Env         string `json:"env"`
	PubSubTopic string `json:"pubSubTopic"` // fully qualified: projects/{project}/topics/{topic}
}
