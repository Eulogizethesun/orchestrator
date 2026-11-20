// Package childenv builds environments for short-lived Go helper processes
// that a long-lived service spawns (sandbox-ctl, checkpoint-cleanup, ...).
package childenv

import "strings"

// GOMAXPROCS caps a short-lived Go helper's scheduler width unless the parent
// environment already sets GOMAXPROCS.
//
// Go sizes its runtime before main: with the default GOMAXPROCS=NumCPU it
// initializes per-P state for every core, which costs ~15-20ms of process
// startup on a 384-core node. Every sandbox create/resume execs a fresh
// runtime, so that tax lands on the hot path. The helpers are I/O-bound (one
// VM control socket, one UFFD handler, one guest link), so a few Ps are
// plenty. Calling runtime.GOMAXPROCS in the child cannot recover the cost —
// it runs after that initialization — so the value must be in the child's
// environment. This keeps the cap scoped to the helper and leaves the
// conductor/runner defaults untouched.
func GOMAXPROCS(env []string, value string) []string {
	for _, kv := range env {
		if strings.HasPrefix(kv, "GOMAXPROCS=") {
			return env
		}
	}
	return append(env, "GOMAXPROCS="+value)
}
