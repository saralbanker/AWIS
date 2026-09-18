package main

import (
	"strings"
	"testing"
)

// TestIsSecretConfigKeyFailsClosed is a security guard, not a style test.
//
// Masking used to be an exact-match lookup against the two key names V1
// recognises. Any other credential-shaped key — a future provider's, a
// hand-edited config.yaml, a near-miss typo — had its VALUE printed verbatim
// by `awis config show` and written verbatim into the append-only audit
// table. A credential printed once cannot be retracted from terminal
// scrollback or CI logs, so this must fail closed.
func TestIsSecretConfigKeyFailsClosed(t *testing.T) {
	secret := []string{
		// The originally-recognised credential keys.
		"api_key", "anthropic_api_key",
		"API_KEY", "Anthropic_API_Key", // case must not matter
		// Near-misses the old exact-match list silently leaked.
		"anthropic_apikey", "apikey", "openai_api_key", "my_api_key_2",
		// The shape that defeated the substring denylist: an HTTP
		// Authorization / Bearer value matches none of
		// key/token/secret/password/credential.
		"authorization", "Authorization", "auth", "bearer_token", "pat",
		// Other credential shapes.
		"auth_token", "TOKEN", "client_secret", "password", "passwd",
		"aws_credential", "refresh_token", "private_key", "signing_key",
		// And anything at all that is simply not on the visible list.
		"something_nobody_anticipated",
	}
	for _, k := range secret {
		if !isSecretConfigKey(k) {
			t.Errorf("isSecretConfigKey(%q) = false; this key's value would be printed verbatim", k)
		}
	}

	// The settings an operator genuinely needs to read back must stay visible,
	// or `config show` becomes useless.
	visible := []string{
		"namespace", "tick", "data_dir", "log_level", "intelligence",
		"plugins_dir", "model", "timeout",
		"NAMESPACE", " tick ", // case and surrounding space must not matter
	}
	for _, k := range visible {
		if isSecretConfigKey(k) {
			t.Errorf("isSecretConfigKey(%q) = true; a non-secret setting is hidden from the operator", k)
		}
	}
}

// TestVisibleKeysHoldNoCredential guards the allowlist itself: if someone adds
// a credential-shaped name to configVisibleKeys, that key's value becomes
// printable. This is the one direction the inversion cannot protect against,
// so it is asserted directly.
func TestVisibleKeysHoldNoCredential(t *testing.T) {
	banned := []string{"key", "token", "secret", "password", "passwd", "credential", "auth"}
	for k := range configVisibleKeys {
		for _, frag := range banned {
			if strings.Contains(strings.ToLower(k), frag) {
				t.Errorf("configVisibleKeys contains %q, which looks credential-shaped (%q); "+
					"its value would be printed verbatim", k, frag)
			}
		}
	}
}
