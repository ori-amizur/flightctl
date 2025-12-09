package pruning

import (
	"context"
	"errors"
	"testing"

	"github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/client"
	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/internal/agent/device/spec"
	"github.com/flightctl/flightctl/pkg/executer"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/flightctl/flightctl/pkg/poll"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestManager_extractImageReferences(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := log.NewPrefixLogger("test")
	mockExec := executer.NewMockExecuter(ctrl)
	readWriter := fileio.NewReadWriter()
	podmanClient := client.NewPodman(log, mockExec, readWriter, poll.Config{})
	mockSpecManager := spec.NewMockManager(ctrl)
	config := config.Pruning{Enabled: true}

	m := NewManager(podmanClient, mockSpecManager, readWriter, log, config).(*manager)

	testCases := []struct {
		name        string
		device      *v1beta1.Device
		setupDevice func(*v1beta1.Device)
		want        []string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:   "nil device",
			device: nil,
			want:   []string{},
		},
		{
			name:   "device with nil spec",
			device: &v1beta1.Device{},
			want:   []string{},
		},
		{
			name: "device with image application",
			device: &v1beta1.Device{
				Spec: &v1beta1.DeviceSpec{
					Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
						{
							Name:    lo.ToPtr("app1"),
							AppType: v1beta1.AppTypeContainer,
						},
					}),
				},
			},
			setupDevice: func(d *v1beta1.Device) {
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				apps := lo.FromPtr(d.Spec.Applications)
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))
			},
			want: []string{"quay.io/example/app:v1.0"},
		},
		{
			name: "device with multiple image applications",
			device: &v1beta1.Device{
				Spec: &v1beta1.DeviceSpec{
					Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
						{
							Name:    lo.ToPtr("app1"),
							AppType: v1beta1.AppTypeContainer,
						},
						{
							Name:    lo.ToPtr("app2"),
							AppType: v1beta1.AppTypeCompose,
						},
					}),
				},
			},
			setupDevice: func(d *v1beta1.Device) {
				apps := lo.FromPtr(d.Spec.Applications)
				imageSpec1 := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app1:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec1))

				imageSpec2 := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app2:v2.0",
				}
				require.NoError(apps[1].FromImageApplicationProviderSpec(imageSpec2))
			},
			want: []string{"quay.io/example/app1:v1.0", "quay.io/example/app2:v2.0"},
		},
		{
			name: "device with image application and volume",
			device: &v1beta1.Device{
				Spec: &v1beta1.DeviceSpec{
					Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
						{
							Name:    lo.ToPtr("app1"),
							AppType: v1beta1.AppTypeContainer,
						},
					}),
				},
			},
			setupDevice: func(d *v1beta1.Device) {
				volume := v1beta1.ApplicationVolume{
					Name: "vol1",
				}
				imageVolSpec := v1beta1.ImageVolumeProviderSpec{
					Image: v1beta1.ImageVolumeSource{
						Reference: "quay.io/example/volume:v1.0",
					},
				}
				require.NoError(volume.FromImageVolumeProviderSpec(imageVolSpec))

				apps := lo.FromPtr(d.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image:   "quay.io/example/app:v1.0",
					Volumes: &[]v1beta1.ApplicationVolume{volume},
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))
			},
			want: []string{"quay.io/example/app:v1.0", "quay.io/example/volume:v1.0"},
		},
		{
			name: "device with inline compose application",
			device: &v1beta1.Device{
				Spec: &v1beta1.DeviceSpec{
					Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
						{
							Name:    lo.ToPtr("app1"),
							AppType: v1beta1.AppTypeCompose,
						},
					}),
				},
			},
			setupDevice: func(d *v1beta1.Device) {
				inlineSpec := v1beta1.InlineApplicationProviderSpec{
					Inline: []v1beta1.ApplicationContent{
						{
							Path: "docker-compose.yaml",
							Content: lo.ToPtr(`version: '3'
services:
  web:
    image: quay.io/example/web:v1.0
  db:
    image: quay.io/example/db:v2.0
`),
						},
					},
				}
				apps := lo.FromPtr(d.Spec.Applications)
				require.NoError(apps[0].FromInlineApplicationProviderSpec(inlineSpec))
			},
			want: []string{"quay.io/example/web:v1.0", "quay.io/example/db:v2.0"},
		},
		{
			name: "device with no applications",
			device: &v1beta1.Device{
				Spec: &v1beta1.DeviceSpec{
					Applications: nil,
				},
			},
			want: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupDevice != nil {
				tc.setupDevice(tc.device)
			}

			got, err := m.extractImageReferences(context.Background(), tc.device)
			if tc.wantErr {
				require.Error(err)
				if tc.wantErrMsg != "" {
					require.Contains(err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(err)
				require.ElementsMatch(tc.want, got)
			}
		})
	}
}

func TestManager_getImageReferencesFromSpecs(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := log.NewPrefixLogger("test")
	mockExec := executer.NewMockExecuter(ctrl)
	readWriter := fileio.NewReadWriter()
	podmanClient := client.NewPodman(log, mockExec, readWriter, poll.Config{})
	mockSpecManager := spec.NewMockManager(ctrl)
	config := config.Pruning{Enabled: true}

	m := NewManager(podmanClient, mockSpecManager, readWriter, log, config).(*manager)

	// Helper to mock image existence checks for nested target extraction
	// For most tests, we'll mock that images don't exist locally (so nested extraction is skipped)
	mockImageNotExists := func(imageRef string) {
		mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", imageRef}).
			Return("", "", 1).AnyTimes() // exit code 1 = image doesn't exist
		mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", imageRef}).
			Return("", "", 1).AnyTimes() // exit code 1 = artifact doesn't exist
	}

	testCases := []struct {
		name       string
		setupMocks func(*executer.MockExecuter, *spec.MockManager)
		want       []string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success with current and rollback specs",
			setupMocks: func(mockExec *executer.MockExecuter, mock *spec.MockManager) {
				// Mock nested target extraction - images don't exist locally (so extraction is skipped)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:current"}).
					Return("", "", 1).AnyTimes()
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:current"}).
					Return("", "", 1).AnyTimes()
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:rollback"}).
					Return("", "", 1).AnyTimes()
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:rollback"}).
					Return("", "", 1).AnyTimes()
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:current",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				rollbackDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
					},
				}
				rollbackApps := lo.FromPtr(rollbackDevice.Spec.Applications)
				rollbackImageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:rollback",
				}
				require.NoError(rollbackApps[0].FromImageApplicationProviderSpec(rollbackImageSpec))

				mock.EXPECT().Read(spec.Current).Return(currentDevice, nil)
				mock.EXPECT().Read(spec.Rollback).Return(rollbackDevice, nil)
			},
			want: []string{"quay.io/example/app:current", "quay.io/example/app:rollback"},
		},
		{
			name: "success with current spec only (no rollback)",
			setupMocks: func(mockExec *executer.MockExecuter, mock *spec.MockManager) {
				// Mock nested target extraction - image doesn't exist locally (so extraction is skipped)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:current"}).
					Return("", "", 1).AnyTimes()
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:current"}).
					Return("", "", 1).AnyTimes()
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:current",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				// Mock nested target extraction - image doesn't exist locally (so extraction is skipped)
				mockImageNotExists("quay.io/example/app:current")

				mock.EXPECT().Read(spec.Current).Return(currentDevice, nil)
				mock.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback spec not found"))
			},
			want: []string{"quay.io/example/app:current"},
		},
		{
			name: "error reading current spec",
			setupMocks: func(mockExec *executer.MockExecuter, mock *spec.MockManager) {
				mock.EXPECT().Read(spec.Current).Return(nil, errors.New("failed to read current spec"))
			},
			wantErr:    true,
			wantErrMsg: "reading current spec",
		},
		{
			name: "error extracting images from current spec",
			setupMocks: func(mockExec *executer.MockExecuter, mock *spec.MockManager) {
				// Return a device with invalid spec structure (missing app type will cause error during extraction)
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: "", // Invalid: missing app type
							},
						}),
					},
				}
				mock.EXPECT().Read(spec.Current).Return(currentDevice, nil)
			},
			wantErr:    true,
			wantErrMsg: "extracting images from current spec",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks(mockExec, mockSpecManager)

			got, err := m.getImageReferencesFromSpecs(context.Background())
			if tc.wantErr {
				require.Error(err)
				if tc.wantErrMsg != "" {
					require.Contains(err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(err)
				require.ElementsMatch(tc.want, got)
			}
		})
	}
}

func TestManager_determineEligibleImages(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := log.NewPrefixLogger("test")
	mockExec := executer.NewMockExecuter(ctrl)
	readWriter := fileio.NewReadWriter()
	mockSpecManager := spec.NewMockManager(ctrl)
	config := config.Pruning{Enabled: true}

	testCases := []struct {
		name       string
		setupMocks func(*executer.MockExecuter, *spec.MockManager)
		want       *EligibleItems
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success with unused images",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager FIRST - needed for getImageReferencesFromSpecs
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				// Mock nested target extraction - image doesn't exist locally (so extraction is skipped)
				// This must come before ListImages because getImageReferencesFromSpecs is called first
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = image doesn't exist
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = artifact doesn't exist

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).Times(2) // Called for apps and OS
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).Times(2)

				// Mock Podman ListImages (called after getImageReferencesFromSpecs)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/app:v1.0\nquay.io/example/unused:v1.0\n", "", 0)

				// Mock Podman ListArtifacts (version check + list)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("quay.io/example/artifact:v1.0\n", "", 0)
			},
			want: &EligibleItems{
				Images:    []string{"quay.io/example/unused:v1.0"},
				Artifacts: []string{"quay.io/example/artifact:v1.0"},
			},
		},
		{
			name: "all images in use - no eligible images",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock Podman ListImages
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/app:v1.0\n", "", 0)

				// Mock Podman ListArtifacts
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "", 0)

				// Mock nested target extraction - image doesn't exist locally (so extraction is skipped)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes()
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes()

				// Mock spec manager
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).Times(2)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).Times(2)
			},
			want: &EligibleItems{Images: []string{}, Artifacts: []string{}}, // All images are in use
		},
		{
			name: "OS images excluded from eligible list",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock Podman ListImages - includes OS image
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/app:v1.0\nquay.io/example/os:v1.0\nquay.io/example/unused:v1.0\n", "", 0)

				// Mock Podman ListArtifacts
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "", 0)

				// Mock spec manager
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).Times(2)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).Times(2)
			},
			want: &EligibleItems{Images: []string{"quay.io/example/unused:v1.0"}, Artifacts: []string{}}, // OS image excluded, unused image eligible
		},
		{
			name: "rollback images preserved",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager - current and rollback
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				currentImageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v2.0", // Current version
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(currentImageSpec))

				rollbackDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				rollbackApps := lo.FromPtr(rollbackDevice.Spec.Applications)
				rollbackImageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0", // Previous version
				}
				require.NoError(rollbackApps[0].FromImageApplicationProviderSpec(rollbackImageSpec))

				// Mock nested target extraction FIRST - images don't exist locally (so extraction is skipped)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v2.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = image doesn't exist
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:v2.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = artifact doesn't exist
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = image doesn't exist
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = artifact doesn't exist

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).Times(2)
				mockSpec.EXPECT().Read(spec.Rollback).Return(rollbackDevice, nil).Times(2)

				// Mock Podman ListImages (called after getImageReferencesFromSpecs)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/app:v1.0\nquay.io/example/app:v2.0\nquay.io/example/unused:v1.0\n", "", 0)

				// Mock Podman ListArtifacts
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "", 0)
			},
			want: &EligibleItems{Images: []string{"quay.io/example/unused:v1.0"}, Artifacts: []string{}}, // Both current and rollback app images preserved
		},
		{
			name: "empty device - all images eligible",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock Podman ListImages
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/unused1:v1.0\nquay.io/example/unused2:v1.0\n", "", 0)

				// Mock Podman ListArtifacts
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "", 0)

				// Mock spec manager - device with no applications
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: nil,
						Os:           nil, // No OS spec
					},
				}

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).Times(2)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).Times(2)
			},
			want: &EligibleItems{Images: []string{"quay.io/example/unused1:v1.0", "quay.io/example/unused2:v1.0"}, Artifacts: []string{}},
		},
		{
			name: "partial failure - continues with available data",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock Podman ListImages - succeeds
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/app:v1.0\nquay.io/example/unused:v1.0\n", "", 0)

				// Mock Podman ListArtifacts - fails
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "error: failed to list artifacts", 1)

				// Mock spec manager
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).Times(2)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).Times(2)
			},
			want: &EligibleItems{Images: []string{"quay.io/example/unused:v1.0"}, Artifacts: []string{}}, // Continues with partial results
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks(mockExec, mockSpecManager)

			podmanClient := client.NewPodman(log, mockExec, readWriter, poll.Config{})
			m := NewManager(podmanClient, mockSpecManager, readWriter, log, config).(*manager)

			got, err := m.determineEligibleImages(context.Background())
			if tc.wantErr {
				require.Error(err)
				if tc.wantErrMsg != "" {
					require.Contains(err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(err)
				require.NotNil(got)
				require.ElementsMatch(tc.want.Images, got.Images)
				require.ElementsMatch(tc.want.Artifacts, got.Artifacts)
			}
		})
	}
}

// TestManager_validateRequiredImages was removed - validateRequiredImages function was redundant
// as determineEligibleImages already handles all validation correctly by only considering
// images that exist locally and building a preserve set from required images in specs.

func TestManager_validateRollbackCapability(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := log.NewPrefixLogger("test")
	mockExec := executer.NewMockExecuter(ctrl)
	readWriter := fileio.NewReadWriter()
	mockSpecManager := spec.NewMockManager(ctrl)
	config := config.Pruning{Enabled: true}

	testCases := []struct {
		name       string
		setupMocks func(*executer.MockExecuter, *spec.MockManager)
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success - all images exist",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager - current spec
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found"))

				// Mock Podman ImageExists calls
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/os:v1.0"}).
					Return("", "", 0)
			},
			wantErr: false,
		},
		{
			name: "failure - current image missing",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager - current spec
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found"))

				// Mock Podman ImageExists - image doesn't exist
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 1)
				// Try as artifact (uses artifact inspect, not artifact exists)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:v1.0"}).
					Return("", "", 1)
			},
			wantErr:    true,
			wantErrMsg: "rollback capability compromised",
		},
		{
			name: "success - no rollback spec",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager - current spec only
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil)
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found"))

				// Mock Podman ImageExists calls
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 0)
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks(mockExec, mockSpecManager)

			podmanClient := client.NewPodman(log, mockExec, readWriter, poll.Config{})
			m := NewManager(podmanClient, mockSpecManager, readWriter, log, config).(*manager)

			err := m.validateRollbackCapability(context.Background())
			if tc.wantErr {
				require.Error(err)
				if tc.wantErrMsg != "" {
					require.Contains(err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestManager_removeEligibleImages(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func(*executer.MockExecuter)
		images     []string
		wantCount  int
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success - all images removed",
			setupMocks: func(mockExec *executer.MockExecuter) {
				// First image: check exists, then remove
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 0) // Image exists
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "rm", "quay.io/example/app:v1.0"}).
					Return("", "", 0) // Image removal succeeds
				// Second image: check exists, then remove
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v2.0"}).
					Return("", "", 0) // Image exists
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "rm", "quay.io/example/app:v2.0"}).
					Return("", "", 0) // Image removal succeeds
			},
			images:    []string{"quay.io/example/app:v1.0", "quay.io/example/app:v2.0"},
			wantCount: 2, // Two images removed
			wantErr:   false,
		},
		{
			name: "success - image doesn't exist (skipped)",
			setupMocks: func(mockExec *executer.MockExecuter) {
				// Image doesn't exist - should be skipped
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 1) // Image doesn't exist
			},
			images:    []string{"quay.io/example/app:v1.0"},
			wantCount: 0, // No removal (image doesn't exist)
			wantErr:   false,
		},
		{
			name: "all removals fail",
			setupMocks: func(mockExec *executer.MockExecuter) {
				// Image exists but removal fails
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 0) // Image exists
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "rm", "quay.io/example/app:v1.0"}).
					Return("", "error: image is in use by container", 1) // Image removal fails
			},
			images:     []string{"quay.io/example/app:v1.0"},
			wantCount:  0, // No removals succeeded
			wantErr:    true,
			wantErrMsg: "all image removals failed",
		},
		{
			name: "empty list - no removals",
			setupMocks: func(mockExec *executer.MockExecuter) {
			},
			images:    []string{},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			log := log.NewPrefixLogger("test")
			mockExec := executer.NewMockExecuter(ctrl)
			readWriter := fileio.NewReadWriter()
			mockSpecManager := spec.NewMockManager(ctrl)
			config := config.Pruning{Enabled: true}

			tc.setupMocks(mockExec)

			podmanClient := client.NewPodman(log, mockExec, readWriter, poll.Config{})
			m := NewManager(podmanClient, mockSpecManager, readWriter, log, config).(*manager)

			count, err := m.removeEligibleImages(context.Background(), tc.images)
			require.Equal(tc.wantCount, count)
			if tc.wantErr {
				require.Error(err)
				if tc.wantErrMsg != "" {
					require.Contains(err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestManager_PruneOnAlert(t *testing.T) {
	require := require.New(t)

	log := log.NewPrefixLogger("test")
	readWriter := fileio.NewReadWriter()

	testCases := []struct {
		name       string
		setupMocks func(*executer.MockExecuter, *spec.MockManager)
		config     config.Pruning
		wantErr    bool
	}{
		{
			name: "success - emergency pruning removes unused images",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager - current spec
				currentDevice := &v1beta1.Device{
					Spec: &v1beta1.DeviceSpec{
						Applications: lo.ToPtr([]v1beta1.ApplicationProviderSpec{
							{
								Name:    lo.ToPtr("app1"),
								AppType: v1beta1.AppTypeContainer,
							},
						}),
						Os: &v1beta1.DeviceOsSpec{
							Image: "quay.io/example/os:v1.0",
						},
					},
				}
				apps := lo.FromPtr(currentDevice.Spec.Applications)
				imageSpec := v1beta1.ImageApplicationProviderSpec{
					Image: "quay.io/example/app:v1.0",
				}
				require.NoError(apps[0].FromImageApplicationProviderSpec(imageSpec))

				// Mock spec reads - called multiple times by determineEligibleImages, validateRollbackCapability
				// Each calls getImageReferencesFromSpecs (Read Current + Rollback) and getOSImageReferences (Read Current + Rollback)
				// So 2 methods * 2 reads each = 4 reads of Current and 4 reads of Rollback
				mockSpec.EXPECT().Read(spec.Current).Return(currentDevice, nil).AnyTimes()
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).AnyTimes()

				// Mock nested target extraction FIRST - image doesn't exist locally (so extraction is skipped)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = image doesn't exist
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "inspect", "quay.io/example/app:v1.0"}).
					Return("", "", 1).AnyTimes() // exit code 1 = artifact doesn't exist

				// Mock Podman ListImages
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("quay.io/example/app:v1.0\nquay.io/example/unused:v1.0\n", "", 0)

				// Mock Podman ListArtifacts
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "", 0)

				// Mock ImageExists for validation (validateRollbackCapability)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/app:v1.0"}).
					Return("", "", 0).AnyTimes()
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/os:v1.0"}).
					Return("", "", 0).AnyTimes()

				// Mock removal of unused image - check exists first, then remove
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "exists", "quay.io/example/unused:v1.0"}).
					Return("", "", 0) // Image exists
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "rm", "quay.io/example/unused:v1.0"}).
					Return("", "", 0) // Image removal succeeds
			},
			config:  config.Pruning{Enabled: true},
			wantErr: false,
		},
		{
			name: "success - continues even if validation fails (emergency situation)",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// Mock spec manager - error reading current spec
				// determineEligibleImages handles all validation
				mockSpec.EXPECT().Read(spec.Current).Return(nil, errors.New("failed to read spec")).AnyTimes()
				mockSpec.EXPECT().Read(spec.Rollback).Return(nil, errors.New("rollback not found")).AnyTimes()

				// Mock Podman ListImages - called before determineEligibleImages fails
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"image", "ls", "--format", `{{if and .Repository (ne .Repository "<none>")}}{{.Repository}}:{{.Tag}}{{else}}{{.ID}}{{end}}`}).
					Return("", "", 0)

				// Mock Podman ListArtifacts - also called before determineEligibleImages fails
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"--version"}).
					Return("podman version 5.5.0", "", 0)
				mockExec.EXPECT().ExecuteWithContext(gomock.Any(), "podman", []string{"artifact", "ls", "--format", "{{.Name}}"}).
					Return("", "", 0)

				// Note: determineEligibleImages will fail when trying to read specs, so no removal happens
				// But the method still returns nil (fail-safe pattern)
			},
			config:  config.Pruning{Enabled: true},
			wantErr: false,
		},
		{
			name: "success - disabled pruning",
			setupMocks: func(mockExec *executer.MockExecuter, mockSpec *spec.MockManager) {
				// No mocks needed - pruning is disabled
			},
			config:  config.Pruning{Enabled: false},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create new controller for each test to avoid expectation leaks
			testCtrl := gomock.NewController(t)
			defer testCtrl.Finish()

			testMockExec := executer.NewMockExecuter(testCtrl)
			testMockSpecManager := spec.NewMockManager(testCtrl)

			tc.setupMocks(testMockExec, testMockSpecManager)

			podmanClient := client.NewPodman(log, testMockExec, readWriter, poll.Config{})
			m := NewManager(podmanClient, testMockSpecManager, readWriter, log, tc.config).(*manager)

			err := m.PruneOnAlert(context.Background())
			if tc.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
