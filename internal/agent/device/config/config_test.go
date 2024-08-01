package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceTokensInArgs(t *testing.T) {
	require := require.New(t)
	tests := []struct {
		name     string
		args     []string
		tokens   map[string]string
		wantErr  error
		expected []string
	}{
		{
			name:     "no tokens",
			args:     []string{"foo", "bar", "baz"},
			tokens:   map[string]string{},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "single token",
			args:     []string{"foo", "bar", "baz", "{{ .FilePath }}"},
			tokens:   map[string]string{"FilePath": "replaced"},
			expected: []string{"foo", "bar", "baz", "replaced"},
		},
		{
			name:     "multiple tokens",
			args:     []string{"foo", "bar", "baz", "{{ .FilePath }}", "{{ .FilePath }}"},
			tokens:   map[string]string{"FilePath": "replaced"},
			expected: []string{"foo", "bar", "baz", "replaced", "replaced"},
		},
		{
			name:     "single token no spaces",
			args:     []string{"foo", "bar", "baz", "{{.FilePath}}"},
			tokens:   map[string]string{"FilePath": "replaced"},
			expected: []string{"foo", "bar", "baz", "replaced"},
		},
		{
			name:    "token not found",
			args:    []string{"{{ .FilePerms }}", "foo", "bar", "baz"},
			tokens:  map[string]string{"FilePath": "replaced"},
			wantErr: ErrTokenNotSupported,
		},
		{
			name:     "multiple different tokens",
			args:     []string{"{{ .FilePath }}", "foo", "bar", "baz", "{{ .FilePerms }}"},
			tokens:   map[string]string{"FilePath": "replaced", "FilePerms": "0666"},
			expected: []string{"replaced", "foo", "bar", "baz", "0666"},
		},
		{
			name:    "invalid token format",
			args:    []string{"{{ FilePath }}", "foo", "bar", "baz"},
			tokens:  map[string]string{"FilePath": "replaced"},
			wantErr: ErrInvalidTokenFormat,
		},
		{
			name:     "multiple tokens in arg",
			args:     []string{"systemctl daemon-reload && systemctl enable {{ .FilePath }} && systemctl start {{ .FilePath }}"},
			tokens:   map[string]string{"FilePath": "/var/run/quadlet.container"},
			expected: []string{"systemctl daemon-reload && systemctl enable /var/run/quadlet.container && systemctl start /var/run/quadlet.container"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceTokensInArgs(tt.args, tt.tokens)
			if tt.wantErr != nil {
				require.ErrorIs(err, tt.wantErr)
				return
			}
			require.NoError(err)
			require.Equal(tt.expected, got)
		})
	}
}
