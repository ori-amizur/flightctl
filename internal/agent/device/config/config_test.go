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
			name:     "single token MOAR spaces",
			args:     []string{"foo", "bar", "baz", "   {{ .FilePath    }} "},
			tokens:   map[string]string{"FilePath": "replaced"},
			expected: []string{"foo", "bar", "baz", "replaced"},
		},
		{
			name:     "token not found",
			args:     []string{"{{ .FilePerms }}", "foo", "bar", "baz"},
			tokens:   map[string]string{"FilePath": "replaced"},
			expected: []string{"{{ .FilePerms }}", "foo", "bar", "baz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceTokensInArgs(tt.args, tt.tokens)
			require.Equal(tt.expected, got)
		})
	}
}
