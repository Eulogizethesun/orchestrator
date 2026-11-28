package orch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/kuasar-sandbox/orchestrator/internal/types"
)

// envdInitFingerprintVersion must change whenever the /init payload semantics
// (defaults, field set, or token posture) change. A stale fingerprint is never
// reused: the conductor restart or a version bump simply causes /init to run.
const envdInitFingerprintVersion = "kuasar-envd-init-v1"

// These defaults are part of the envd /init contract and are shared by the
// request builder and the fingerprint so the two cannot drift.
const (
	envdDefaultUser    = "user"
	envdDefaultWorkdir = "/home/user"
)

// envdInitFingerprint covers exactly the values envd /init applies: env vars,
// the fixed default user/workdir, and — only in MMDS posture — the per-sandbox
// EnvdAccessToken. It deliberately excludes the timestamp (restore already sets
// the wall clock before ACK) and the MMDS session token (guest-minted, TTL,
// invalidated by pause/resume, and never an /init input).
func envdInitFingerprint(sb *types.Sandbox, mmdsEnabled bool) string {
	if sb == nil {
		return ""
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\nprofile=%s\ntemplate=%s\nuser=%s\nworkdir=%s\nmmds=%t\n",
		envdInitFingerprintVersion, sb.Profile, sb.TemplateID, envdDefaultUser, envdDefaultWorkdir, mmdsEnabled)
	keys := make([]string, 0, len(sb.Env))
	for key := range sb.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := sb.Env[key]
		// Length-prefix both sides so ("ab","c") cannot collide with ("a","bc").
		fmt.Fprintf(h, "env %d:%s %d:%s\n", len(key), key, len(value), value)
	}
	if mmdsEnabled {
		fmt.Fprintf(h, "envd-token %d:%s\n", len(sb.EnvdAccessToken), sb.EnvdAccessToken)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// envdInitWasApplied reports whether this process successfully applied the
// same /init payload to this sandbox. A missing entry (new identity, conductor
// restart, cache eviction) is a miss, which forces a conservative /init.
//
// The hint assumes envd itself has not restarted in the guest since that
// successful /init. The conductor has no envd-incarnation signal; an envd crash
// is an abnormal condition (the first exec fails) and a reset/recreate re-runs
// /init. It never authorizes skipping a create/fork, a changed env/token, or a
// changed MMDS posture.
func (o *Orchestrator) envdInitWasApplied(id, fingerprint string) bool {
	if o == nil || id == "" || fingerprint == "" {
		return false
	}
	o.envdInitMu.Lock()
	defer o.envdInitMu.Unlock()
	return o.envdInitFP[id] == fingerprint
}

// markEnvdInit records a successful /init. It must be called only after envd
// returned 204, so a failed init never authorizes a later skip.
func (o *Orchestrator) markEnvdInit(id, fingerprint string) {
	if o == nil || id == "" || fingerprint == "" {
		return
	}
	o.envdInitMu.Lock()
	defer o.envdInitMu.Unlock()
	if o.envdInitFP == nil {
		o.envdInitFP = map[string]string{}
	}
	o.envdInitFP[id] = fingerprint
}

// forgetEnvdInit drops the hint on delete/retention/cache eviction. The next
// launch then runs /init; losing the hint is always safe.
func (o *Orchestrator) forgetEnvdInit(id string) {
	if o == nil || id == "" {
		return
	}
	o.envdInitMu.Lock()
	defer o.envdInitMu.Unlock()
	delete(o.envdInitFP, id)
}
