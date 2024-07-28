package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/pkg/executer"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/require"
)

var (
	testRetryTimeout  = 5 * time.Second
	testRetryInterval = 100 * time.Millisecond
)

func TestHookManager(t *testing.T) {
	require := require.New(t)
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(err)
	execPath := filepath.Join(cwd, "testdata", "executable_script.sh")
	varDirPath := filepath.Join(tmpDir, "var/lib/stuff")
	err = os.MkdirAll(varDirPath, 0755)
	require.NoError(err)

	log := log.NewPrefixLogger("test")

	type TestFiles struct {
		filePath string
		content  string
		op       v1alpha1.FileOperation
	}

	tests := []struct {
		name string
		hook *v1alpha1.DeviceConfigHookSpec
		// configFiles is a list of files that should be created before the hook is updated
		configFiles []TestFiles
		// desiredFiles is a list of files that should be created by the hook
		desiredFiles []TestFiles
	}{
		{
			name: "happy path file create",
			hook: &v1alpha1.DeviceConfigHookSpec{
				Name:        "test-hook",
				Description: "test hook",
				WatchPath:   varDirPath,
				Actions: []v1alpha1.ConfigHookAction{
					newTestConfigHookActionExecutable(t, cwd, execPath, []v1alpha1.FileOperation{v1alpha1.FileOperationCreate}, filepath.Join(varDirPath, "file1"), "file1-content"),
				},
			},
			configFiles: []TestFiles{
				{
					filePath: filepath.Join(varDirPath, "configFile1"),
					content:  "configFile1-content",
					op:       v1alpha1.FileOperationCreate,
				},
			},
			desiredFiles: []TestFiles{
				{
					filePath: filepath.Join(varDirPath, "file1"),
					content:  "file1-content",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			exec := executer.Executer(&executer.CommonExecuter{})
			hookManager, err := NewHookManager(log, exec)
			require.NoError(err)

			go hookManager.Run(ctx)

			require.Eventuallyf(func() bool {
				updated, err := hookManager.Update(tt.hook)
				require.NoError(err)
				return updated == true
			}, testRetryTimeout, testRetryInterval, "hook not updated")

			for _, file := range tt.configFiles {
				switch file.op {
				case v1alpha1.FileOperationCreate:
					createTestFile(t, file.filePath)
				case v1alpha1.FileOperationDelete:
					deleteTestFile(t, file.filePath)
				}
			}
			// the executable script should create the desired files giving us signal it is working as expected
			require.Eventuallyf(func() bool {
				for _, file := range tt.desiredFiles {
					if _, err := os.Stat(file.filePath); os.IsNotExist(err) {
						return false
					}
				}
				return true
			}, testRetryTimeout, testRetryInterval, "desired files not created")
		})
	}
}

func createTestFile(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)
}

func deleteTestFile(t *testing.T, path string) {
	t.Helper()
	err := os.Remove(path)
	require.NoError(t, err)
}

func newTestConfigHookActionExecutable(t *testing.T, workingDir string, execPath string, ops []v1alpha1.FileOperation, execArgs ...string) v1alpha1.ConfigHookAction {
	t.Helper()
	action := v1alpha1.ConfigHookAction{}
	actionExec := v1alpha1.ConfigHookActionExecutableSpec{
		Executable: v1alpha1.ConfigHookActionExecutable{
			Command: execPath,
			Args:    execArgs,
			WorkDir: workingDir,
		},
		TriggerOn: ops,
	}
	err := action.FromConfigHookActionExecutableSpec(actionExec)
	require.NoError(t, err)
	return action
}
