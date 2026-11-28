// Package trustprepare implements the KUASAR_TRUST_PREPARE operating mode.
//
// KUASAR_TRUST_PREPARE=1 makes the runner append --trust-prepare to the
// sandbox-ctl argv for memory-snapshot restores whose artifact preparation
// succeeded in this same node process. The flag is a trust declaration: the
// prepare pass already validated the exact immutable artifacts with the same
// library, so the runtime skips only re-verification — the snapshot.cfg
// canonical re-encoding equality and the Sandbox E canonical re-encoding +
// EROFS payload size cross-check. Parsing and every file open are unchanged,
// and both re-verified properties still fail closed later (a non-canonical
// config re-encodes to the same document; a payload size mismatch surfaces
// when the EROFS superblock or block reader opens during preflight) — still
// ahead of any VM side effect.
//
// Off by default. It requires the paired sandbox-ctl change (--trust-prepare)
// to be deployed: a flag sent to an older sandbox-ctl is rejected as unknown
// and fails the launch. Production deployments must leave it unset until the
// trust policy is signed off.
package trustprepare

import "os"

// EnvVar is the environment variable that enables the mode.
const EnvVar = "KUASAR_TRUST_PREPARE"

// Enabled reports whether the trust-prepare handoff is active.
func Enabled() bool {
	return os.Getenv(EnvVar) == "1"
}
