package plugin

// isolation.go — dedicated-UID plugin isolation (D-11, PRD §32 rows 38/53,
// founder ruling DEC-9).
//
// A plugin subprocess previously ran as the engine's own UID, so
// env-scrubbing (NFR-S-02) alone could not stop a same-UID process from
// simply opening .awis/runtime.db directly. ResolvePluginUser turns the
// optional `plugins_user` config setting (a username, or a numeric
// uid[:gid]) into a concrete (uid, gid) pair that the spawn site
// (transport.go spawnPlugin) sets via SysProcAttr.Credential.

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"
)

// ResolvePluginUser resolves spec — a username (e.g. "awis-plugin") or a
// numeric uid, optionally followed by ":<gid>" (e.g. "1000" or "1000:1000")
// — to a concrete (uid, gid) pair.
//
// A username resolves to its passwd-entry uid and primary gid via
// os/user.Lookup. A bare numeric uid resolves its primary gid via
// os/user.LookupId. A "uid:gid" pair is taken literally: neither side is
// looked up. An empty spec, or a spec that does not resolve to a known
// user/uid, is an error — callers must not fall back to unisolated spawn on
// a resolution failure (D-11 failure-mode requirement).
func ResolvePluginUser(spec string) (uid, gid uint32, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: empty plugin user spec")
	}

	if idx := strings.IndexByte(spec, ':'); idx >= 0 {
		uidStr, gidStr := spec[:idx], spec[idx+1:]
		uidN, uerr := strconv.ParseUint(uidStr, 10, 32)
		if uerr != nil {
			return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: invalid uid %q in plugin user spec %q: %w", uidStr, spec, uerr)
		}
		gidN, gerr := strconv.ParseUint(gidStr, 10, 32)
		if gerr != nil {
			return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: invalid gid %q in plugin user spec %q: %w", gidStr, spec, gerr)
		}
		return uint32(uidN), uint32(gidN), nil
	}

	if uidN, uerr := strconv.ParseUint(spec, 10, 32); uerr == nil {
		u, lerr := user.LookupId(spec)
		if lerr != nil {
			return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: uid %d has no passwd entry, cannot resolve its gid (use uid:gid form instead): %w", uidN, lerr)
		}
		gidN, gerr := strconv.ParseUint(u.Gid, 10, 32)
		if gerr != nil {
			return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: uid %d resolved to non-numeric gid %q: %w", uidN, u.Gid, gerr)
		}
		return uint32(uidN), uint32(gidN), nil
	}

	u, lerr := user.Lookup(spec)
	if lerr != nil {
		return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: unknown plugin user %q: %w", spec, lerr)
	}
	uidN, uerr := strconv.ParseUint(u.Uid, 10, 32)
	if uerr != nil {
		return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: user %q resolved to non-numeric uid %q: %w", spec, u.Uid, uerr)
	}
	gidN, gerr := strconv.ParseUint(u.Gid, 10, 32)
	if gerr != nil {
		return 0, 0, fmt.Errorf("plugin: ResolvePluginUser: user %q resolved to non-numeric gid %q: %w", spec, u.Gid, gerr)
	}
	return uint32(uidN), uint32(gidN), nil
}
