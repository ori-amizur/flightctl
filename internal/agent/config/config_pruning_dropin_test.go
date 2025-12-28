package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/stretchr/testify/require"
)

func TestLoadPruningFromConfig(t *testing.T) {
	tests := []struct {
		name            string
		setupFiles      func(t *testing.T, configDir string) string
		expectedEnabled bool
		wantErr         bool
	}{
		{
			name: "no config files - uses default",
			setupFiles: func(t *testing.T, configDir string) string {
				configFile := filepath.Join(configDir, "config.yaml")
				content := `enrollment-service:
  service:
    server: https://enrollment.endpoint
    certificate-authority-data: abcd
  authentication:
    client-certificate-data: efgh
    client-key-data: ijkl
spec-fetch-interval: 0m10s
status-update-interval: 0m10s
`
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
				return configFile
			},
			expectedEnabled: false, // Default from NewDefault()
		},
		{
			name: "base config file enables pruning",
			setupFiles: func(t *testing.T, configDir string) string {
				configFile := filepath.Join(configDir, "config.yaml")
				content := `enrollment-service:
  service:
    server: https://enrollment.endpoint
    certificate-authority-data: abcd
  authentication:
    client-certificate-data: efgh
    client-key-data: ijkl
spec-fetch-interval: 0m10s
status-update-interval: 0m10s
pruning:
  enabled: true
`
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
				return configFile
			},
			expectedEnabled: true,
		},
		{
			name: "base config file disables pruning",
			setupFiles: func(t *testing.T, configDir string) string {
				configFile := filepath.Join(configDir, "config.yaml")
				content := `enrollment-service:
  service:
    server: https://enrollment.endpoint
    certificate-authority-data: abcd
  authentication:
    client-certificate-data: efgh
    client-key-data: ijkl
spec-fetch-interval: 0m10s
status-update-interval: 0m10s
pruning:
  enabled: false
`
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
				return configFile
			},
			expectedEnabled: false,
		},
		{
			name: "dropin overrides base config file",
			setupFiles: func(t *testing.T, configDir string) string {
				// Base file enables
				configFile := filepath.Join(configDir, "config.yaml")
				content := `enrollment-service:
  service:
    server: https://enrollment.endpoint
    certificate-authority-data: abcd
  authentication:
    client-certificate-data: efgh
    client-key-data: ijkl
spec-fetch-interval: 0m10s
status-update-interval: 0m10s
pruning:
  enabled: true
`
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
				// Dropin disables
				dropinDir := filepath.Join(configDir, "conf.d")
				require.NoError(t, os.MkdirAll(dropinDir, 0755))
				dropinPath := filepath.Join(dropinDir, "01-disable.yaml")
				require.NoError(t, os.WriteFile(dropinPath, []byte("pruning:\n  enabled: false\n"), 0644))
				return configFile
			},
			expectedEnabled: false,
		},
		{
			name: "multiple dropins - later overrides earlier",
			setupFiles: func(t *testing.T, configDir string) string {
				configFile := filepath.Join(configDir, "config.yaml")
				content := `enrollment-service:
  service:
    server: https://enrollment.endpoint
    certificate-authority-data: abcd
  authentication:
    client-certificate-data: efgh
    client-key-data: ijkl
spec-fetch-interval: 0m10s
status-update-interval: 0m10s
`
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
				dropinDir := filepath.Join(configDir, "conf.d")
				require.NoError(t, os.MkdirAll(dropinDir, 0755))
				// First dropin enables
				require.NoError(t, os.WriteFile(filepath.Join(dropinDir, "01-enable.yaml"), []byte("pruning:\n  enabled: true\n"), 0644))
				// Second dropin disables (should win)
				require.NoError(t, os.WriteFile(filepath.Join(dropinDir, "02-disable.yaml"), []byte("pruning:\n  enabled: false\n"), 0644))
				return configFile
			},
			expectedEnabled: false,
		},
		{
			name: "invalid yaml in base config file",
			setupFiles: func(t *testing.T, configDir string) string {
				configFile := filepath.Join(configDir, "config.yaml")
				require.NoError(t, os.WriteFile(configFile, []byte("invalid: yaml: content: [\n"), 0644))
				return configFile
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, "etc", "flightctl")
			dataDir := filepath.Join(tmpDir, "var", "lib", "flightctl")

			// Create necessary directories
			require.NoError(t, os.MkdirAll(configDir, 0755))
			require.NoError(t, os.MkdirAll(dataDir, 0755))

			cfg := NewDefault()
			cfg.ConfigDir = configDir
			cfg.DataDir = dataDir
			cfg.readWriter = fileio.NewReadWriter()

			configFile := tt.setupFiles(t, configDir)

			err := cfg.LoadWithOverrides(configFile)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedEnabled, cfg.Pruning.Enabled)
		})
	}
}

func TestLoadWithOverridesIncludesPruningFromConfD(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "etc", "flightctl")
	dataDir := filepath.Join(tmpDir, "var", "lib", "flightctl")

	// Create necessary directories
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(dataDir, 0755))

	cfg := NewDefault()
	cfg.ConfigDir = configDir
	cfg.DataDir = dataDir
	cfg.readWriter = fileio.NewReadWriter()

	// Create base config file with pruning disabled
	configFile := filepath.Join(configDir, "config.yaml")
	content := `enrollment-service:
  service:
    server: https://enrollment.endpoint
    certificate-authority-data: abcd
  authentication:
    client-certificate-data: efgh
    client-key-data: ijkl
spec-fetch-interval: 0m10s
status-update-interval: 0m10s
pruning:
  enabled: false
`
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

	// Create dropin in conf.d that enables pruning (should override config)
	dropinDir := filepath.Join(configDir, "conf.d")
	require.NoError(t, os.MkdirAll(dropinDir, 0755))
	dropinPath := filepath.Join(dropinDir, "enable.yaml")
	require.NoError(t, os.WriteFile(dropinPath, []byte("pruning:\n  enabled: true\n"), 0644))

	// Load config with overrides
	err := cfg.LoadWithOverrides(configFile)
	require.NoError(t, err)
	// Dropin should override config, so pruning should be enabled
	require.True(t, cfg.Pruning.Enabled, "pruning dropin should override config setting")
}
