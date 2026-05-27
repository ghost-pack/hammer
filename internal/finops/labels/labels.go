package labels

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ghost-pack/hammer/internal/oam"
)

const (
	KeyApp  = "app"
	KeyKind = "kind"
	KeyEnv  = "env"
	KeyTeam = "team"
	//KeyCostCenter    = "cost-center"
	KeyManagedBy     = "managed-by"
	KeyHammerVersion = "hammer-version"
	KeyRepo          = "repo"
	KeyCommit        = "commit"
	KeyTier          = "tier"
	KeyDataClass     = "data-class"
	KeyLifecycle     = "lifecycle"
)

const ManagedByValue = "hammer"

var (
	invalidChars = regexp.MustCompile(`[^a-z0-9_]`)
	validKey     = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	validValue   = regexp.MustCompile(`^[a-z0-9_-]{0,63}$`)
)

var requiredKeys = []string{
	KeyApp, KeyKind, KeyEnv, KeyTeam /* KeyCostCenter, */, KeyManagedBy, KeyHammerVersion, KeyRepo, KeyCommit,
}

type Labels map[string]string

type Builder struct {
	App           *oam.App
	Env           string
	HammerVersion string
	Repo          string
	Commit        string
}

func (b *Builder) Build() (Labels, error) {
	l := Labels{
		KeyApp:  normalize(b.App.Metadata.Name),
		KeyKind: normalize(b.App.Kind),
		KeyEnv:  normalize(b.Env),
		KeyTeam: normalize(b.App.Metadata.Annotations["team"]),
		//KeyCostCenter: normalize(b.App.Metadata.Annotations["cost-center"]),
		KeyManagedBy:     normalize(ManagedByValue),
		KeyHammerVersion: normalize(b.HammerVersion),
		KeyRepo:          normalize(b.Repo),
		KeyCommit:        normalize(b.Commit),
	}
	// todo: later check for lifecycle and tier.
	if err := l.Validate(); err != nil {
		return nil, fmt.Errorf("invalid labels: %w", err)
	}
	return l, nil
}

func normalize(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "@", "_")
	s = strings.ReplaceAll(s, "+", "-")
	s = invalidChars.ReplaceAllString(s, "-")
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}

func (l Labels) Validate() error {
	for _, k := range requiredKeys {
		if v, ok := l[k]; !ok || v == "" {
			return fmt.Errorf("missing required label: %q", k)
		}
	}
	if len(l) > 64 {
		return fmt.Errorf("too many labels: %d (max 64)", len(l))
	}
	for k, v := range l {
		if !validKey.MatchString(k) {
			return fmt.Errorf("invalid label key: %q", k)
		}
		if !validValue.MatchString(v) {
			return fmt.Errorf("invalid label value for %q: %q", k, v)
		}
	}
	return nil
}
