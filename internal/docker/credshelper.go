package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	credclient "github.com/docker/docker-credential-helpers/client"
)

type dockerConfigFile struct {
	CredHelpers map[string]string `json:"credHelpers"`
	Auths       map[string]struct {
		Auth string `json:"auth"`
	} `json:"auths"`
}

// authPayload is the JSON shape the Docker daemon expects in RegistryAuth.
type authPayload struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
}

func resolveRegistryAuth(image string) (string, error) {
	registry := strings.SplitN(image, "/", 2)[0]

	cfg, err := loadDockerConfig()
	if err != nil {
		return "", err
	}

	var payload authPayload

	if helperSuffix, ok := cfg.CredHelpers[registry]; ok {
		// Invoke e.g. docker-credential-gcloud get
		helper := credclient.NewShellProgramFunc("docker-credential-" + helperSuffix)
		creds, err := credclient.Get(helper, registry)
		if err != nil {
			return "", fmt.Errorf("credential helper %q for %s: %w", helperSuffix, registry, err)
		}
		payload = authPayload{
			Username:      creds.Username,
			Password:      creds.Secret,
			ServerAddress: creds.ServerURL,
		}
	} else if stored, ok := cfg.Auths[registry]; ok && stored.Auth != "" {
		// Fallback: base64(user:pass) stored directly in config
		decoded, err := base64.StdEncoding.DecodeString(stored.Auth)
		if err != nil {
			return "", fmt.Errorf("decoding stored auth for %s: %w", registry, err)
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("malformed stored auth for %s", registry)
		}
		payload = authPayload{
			Username:      parts[0],
			Password:      parts[1],
			ServerAddress: registry,
		}
	} else {
		return "", fmt.Errorf("no credentials found for %s", registry)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encoding auth payload: %w", err)
	}
	return base64.URLEncoding.EncodeToString(encoded), nil
}

func loadDockerConfig() (*dockerConfigFile, error) {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("finding home dir: %w", err)
		}
		configDir = filepath.Join(home, ".docker")
	}

	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("reading docker config: %w", err)
	}

	var cfg dockerConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing docker config: %w", err)
	}
	return &cfg, nil
}
