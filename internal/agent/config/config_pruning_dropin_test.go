package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/stretchr/testify/require"
)

func TestLoadPruningFromDropins(t *testing.T) {
	tests := []struct {
		name            string
		setupFiles      func(t *testing.T, configDir string)
		expectedEnabled bool
		wantErr         bool
	}{
		{
			name: "no pruning files - uses default",
			setupFiles: func(t *testing.T, configDir string) {
				// No files created
			},
			expectedEnabled: false, // Default from NewDefault()
		},
		{
			name: "base file enables pruning",
			setupFiles: func(t *testing.T, configDir string) {
				basePath := filepath.Join(configDir, "pruning.yaml")
				content := "enabled: true\n"
				require.NoError(t, os.WriteFile(basePath, []byte(content), 0644))
			},
			expectedEnabled: true,
		},
		{
			name: "base file disables pruning",
			setupFiles: func(t *testing.T, configDir string) {
				basePath := filepath.Join(configDir, "pruning.yaml")
				content := "enabled: false\n"
				require.NoError(t, os.WriteFile(basePath, []byte(content), 0644))
			},
			expectedEnabled: false,
		},
		{
			name: "dropin overrides base file",
			setupFiles: func(t *testing.T, configDir string) {
				// Base file enables
				basePath := filepath.Join(configDir, "pruning.yaml")
				require.NoError(t, os.WriteFile(basePath, []byte("enabled: true\n"), 0644))
				// Dropin disables
				dropinDir := filepath.Join(configDir, "pruning.d")
				require.NoError(t, os.MkdirAll(dropinDir, 0755))
				dropinPath := filepath.Join(dropinDir, "01-disable.yaml")
				require.NoError(t, os.WriteFile(dropinPath, []byte("enabled: false\n"), 0644))
			},
			expectedEnabled: false,
		},
		{
			name: "multiple dropins - later overrides earlier",
			setupFiles: func(t *testing.T, configDir string) {
				dropinDir := filepath.Join(configDir, "pruning.d")
				require.NoError(t, os.MkdirAll(dropinDir, 0755))
				// First dropin enables
				require.NoError(t, os.WriteFile(filepath.Join(dropinDir, "01-enable.yaml"), []byte("enabled: true\n"), 0644))
				// Second dropin disables (should win)
				require.NoError(t, os.WriteFile(filepath.Join(dropinDir, "02-disable.yaml"), []byte("enabled: false\n"), 0644))
			},
			expectedEnabled: false,
		},
		{
			name: "invalid YAML in base file",
			setupFiles: func(t *testing.T, configDir string) {
				basePath := filepath.Join(configDir, "pruning.yaml")
				require.NoError(t, os.WriteFile(basePath, []byte("invalid: yaml: content\n"), 0644))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFiles(t, tmpDir)

			cfg := NewDefault()
			cfg.ConfigDir = tmpDir
			cfg.readWriter = fileio.NewReadWriter()

			err := cfg.loadPruningFromDropins()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedEnabled, cfg.Pruning.Enabled)
		})
	}
}

func TestLoadWithOverridesIncludesPruningDropins(t *testing.T) {
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

	// Set pruning to false in config
	cfg.Pruning.Enabled = false

	// Create pruning dropin that enables pruning (should override config)
	dropinDir := filepath.Join(configDir, "pruning.d")
	require.NoError(t, os.MkdirAll(dropinDir, 0755))
	dropinPath := filepath.Join(dropinDir, "enable.yaml")
	require.NoError(t, os.WriteFile(dropinPath, []byte("enabled: true\n"), 0644))

	// Load dropins
	err := cfg.loadPruningFromDropins()
	require.NoError(t, err)
	// Dropin should override config, so pruning should be enabled
	require.True(t, cfg.Pruning.Enabled, "pruning dropin should override config setting")
}
