package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kuasar-sandbox/orchestrator/internal/configsock"
	"github.com/kuasar-sandbox/orchestrator/internal/taskartifact"
	"github.com/kuasar-sandbox/orchestrator/internal/trustprepare"
	"github.com/kuasar-sandbox/orchestrator/internal/types"
)

// runTrustPrepareLaunch drives launchTaskWith through the two-stage prepare
// path and returns the exact sandbox-ctl argv handed to startChild.
func runTrustPrepareLaunch(t *testing.T) string {
	t.Helper()
	vmm, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer vmm.Close()
	startErr := errors.New("child start intercepted")
	var argv string
	err = launchTaskWith(context.Background(), func() {}, "/config.sock", "sid", "run-1", nil, vmm, taskLaunchOps{
		fetchBootstrap: func(context.Context, string, string, string) (*configsock.SandboxTaskSpec, error) {
			return &configsock.SandboxTaskSpec{
				SandboxID: "sid", RunID: "run-1", Workdir: "/task-work",
				Env: map[string]string{"MANIFEST_KEY": "authoritative-key"},
				Prepare: &configsock.ArtifactPrepareSpec{RunID: "run-1",
					RootRef: "manifest://root", AbsoluteDeadlineUnixNano: time.Now().Add(time.Minute).UnixNano(),
				},
			}, nil
		},
		setenv: os.Setenv,
		prepareArtifact: func(context.Context, configsock.ArtifactPrepareSpec) (*taskartifact.Result, error) {
			return &taskartifact.Result{
				PreparedSource: types.ResumeSource{Kind: types.ResumeSourceSnapshot, Ref: "manifest://root"},
				Summary: configsock.ArtifactPrepareSummary{
					SchemaVersion: configsock.ArtifactPrepareSchemaVersion, PreparedSourceKind: string(types.ResumeSourceSnapshot),
					ResolutionDigest: strings.Repeat("1", 64), RequiredRefCount: 1,
				},
				RefLocationURIs: map[string]string{"a-location": "file:///a"},
			}, nil
		},
		completePrepare: func(context.Context, string, string, string, configsock.ArtifactPrepareSummary) (*configsock.LaunchSpec, error) {
			return &configsock.LaunchSpec{Exec: "/bin/sandbox-ctl", Args: []string{"run"}}, nil
		},
		chdir: func(string) error { return nil },
		startChild: func(_ string, childArgv []string, _ []string, _, _ *os.File) error {
			argv = strings.Join(childArgv, " ")
			return startErr
		},
	})
	if !errors.Is(err, startErr) {
		t.Fatalf("launchTaskWith error = %v", err)
	}
	return argv
}

func TestLaunchTaskTrustPrepareAppendsFlagWhenEnabled(t *testing.T) {
	t.Setenv(trustprepare.EnvVar, "1")
	argv := runTrustPrepareLaunch(t)
	if want := "--restore manifest://root --trust-prepare"; !strings.Contains(argv, want) {
		t.Fatalf("argv = %q, want it to contain %q directly after --restore", argv, want)
	}
}

func TestLaunchTaskOmitsTrustPrepareByDefault(t *testing.T) {
	t.Setenv(trustprepare.EnvVar, "")
	argv := runTrustPrepareLaunch(t)
	if strings.Contains(argv, "--trust-prepare") {
		t.Fatalf("argv = %q, want no --trust-prepare without %s=1", argv, trustprepare.EnvVar)
	}
}
