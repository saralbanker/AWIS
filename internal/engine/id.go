package engine

import (
	"crypto/rand"
	"encoding/hex"
)

// newUUIDv4 returns a random RFC 4122 version-4 UUID string using crypto/rand
// (stdlib only — no third-party dependency). It is the engine's default id
// source for instance ids and event ids; tests inject a deterministic source via
// the engine's newID field (IMP §3 determinism rule: ids via injectable sources).
func newUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand.Read never returns an error on supported platforms; if it
		// somehow does, panicking is correct — a workflow engine cannot mint
		// non-unique ids silently.
		panic("engine: crypto/rand failed: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}
