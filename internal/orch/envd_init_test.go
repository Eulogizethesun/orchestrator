package orch

import (
	"testing"

	"github.com/kuasar-sandbox/orchestrator/internal/types"
)

func TestEnvdInitFingerprintCoversPayloadOnly(t *testing.T) {
	base := &types.Sandbox{
		Profile: types.ProfileE2B,
		Env:     map[string]string{"A": "1", "B": "2"},
		EnvdAccessToken: "token-a",
		CreatedUnix:     100,
	}
	if got := envdInitFingerprint(nil, true); got != "" {
		t.Fatalf("nil sandbox fingerprint = %q, want empty", got)
	}

	// Map iteration order must not matter.
	reordered := &types.Sandbox{
		Profile: types.ProfileE2B,
		Env:     map[string]string{"B": "2", "A": "1"},
		EnvdAccessToken: "token-a",
		CreatedUnix:     999,
	}
	if a, b := envdInitFingerprint(base, true), envdInitFingerprint(reordered, true); a != b {
		t.Fatalf("fingerprint depends on map order or excluded fields: %q != %q", a, b)
	}

	// Every payload-affecting value changes the fingerprint.
	cases := []struct {
		name string
		sb   *types.Sandbox
		mmds bool
	}{
		{"env value", &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"A": "1", "B": "3"}, EnvdAccessToken: "token-a"}, true},
		{"env key", &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"A": "1", "C": "2"}, EnvdAccessToken: "token-a"}, true},
		{"token", &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"A": "1", "B": "2"}, EnvdAccessToken: "token-b"}, true},
		{"template", &types.Sandbox{Profile: types.ProfileE2B, TemplateID: "other-template", Env: map[string]string{"A": "1", "B": "2"}, EnvdAccessToken: "token-a"}, true},
		{"mmds posture", &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"A": "1", "B": "2"}, EnvdAccessToken: "token-a"}, false},
	}
	for _, tc := range cases {
		if got, want := envdInitFingerprint(tc.sb, tc.mmds), envdInitFingerprint(base, true); got == want {
			t.Fatalf("%s did not change the fingerprint", tc.name)
		}
	}
}

func TestEnvdInitFingerprintTokenOnlyInMMDSPosture(t *testing.T) {
	withToken := &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"A": "1"}, EnvdAccessToken: "token-a"}
	otherToken := &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"A": "1"}, EnvdAccessToken: "token-b"}
	if a, b := envdInitFingerprint(withToken, false), envdInitFingerprint(otherToken, false); a != b {
		t.Fatal("token must not affect /init fingerprint when MMDS is disabled")
	}
	if a, b := envdInitFingerprint(withToken, true), envdInitFingerprint(otherToken, true); a == b {
		t.Fatal("token must affect /init fingerprint when MMDS is enabled")
	}
}

func TestEnvdInitFingerprintRejectsAmbiguousEnv(t *testing.T) {
	a := &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"a": "bc"}}
	b := &types.Sandbox{Profile: types.ProfileE2B, Env: map[string]string{"ab": "c"}}
	if envdInitFingerprint(a, true) == envdInitFingerprint(b, true) {
		t.Fatal("length-unaware env encoding collided")
	}
}

func TestEnvdInitHintLifecycle(t *testing.T) {
	o := &Orchestrator{reg: map[string]*types.Sandbox{}}
	if o.envdInitWasApplied("sid", "fp") {
		t.Fatal("empty hint must miss")
	}
	o.markEnvdInit("sid", "fp")
	if !o.envdInitWasApplied("sid", "fp") {
		t.Fatal("marked hint did not match")
	}
	if o.envdInitWasApplied("sid", "other") {
		t.Fatal("different fingerprint matched")
	}
	o.uncache("sid")
	if o.envdInitWasApplied("sid", "fp") {
		t.Fatal("uncache did not drop the hint")
	}
}

func TestEnvdInitHintIgnoresEmptyValues(t *testing.T) {
	o := &Orchestrator{}
	o.markEnvdInit("", "fp")
	o.markEnvdInit("sid", "")
	if o.envdInitWasApplied("sid", "") {
		t.Fatal("empty fingerprint matched")
	}
}
