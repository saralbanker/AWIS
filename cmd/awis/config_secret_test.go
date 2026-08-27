package main

import "testing"

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
		// The originally-recognised keys.
		"api_key", "anthropic_api_key",
		"API_KEY", "Anthropic_API_Key", // case must not matter
		// Near-misses that the exact-match list silently leaked.
		"anthropic_apikey", "apikey", "openai_api_key", "my_api_key_2",
		// Other credential shapes.
		"auth_token", "TOKEN", "client_secret", "password", "passwd",
		"aws_credential", "refresh_token",
	}
	for _, k := range secret {
		if !isSecretConfigKey(k) {
			t.Errorf("isSecretConfigKey(%q) = false; this key's value would be printed verbatim", k)
		}
	}

	// Keys that must stay readable — masking everything would make
	// `config show` useless for the settings an operator actually needs to
	// check.
	visible := []string{
		"namespace", "tick", "data_dir", "log_level", "intelligence",
		"plugins_dir", "timeout", "model",
	}
	for _, k := range visible {
		if isSecretConfigKey(k) {
			t.Errorf("isSecretConfigKey(%q) = true; a non-secret setting is being hidden from the operator", k)
		}
	}
}

// TestEveryRecognisedSecretKeyIsCoveredBySubstring keeps the two mechanisms
// consistent: every key on the explicit list must ALSO be caught by the
// substring rule, so the explicit list can never be the only thing standing
// between a credential and stdout.
func TestEveryRecognisedSecretKeyIsCoveredBySubstring(t *testing.T) {
	for k := range configSecretKeys {
		var covered bool
		for _, frag := range secretKeySubstrings {
			if len(frag) > 0 && contains(k, frag) {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("configSecretKeys has %q, which no entry in secretKeySubstrings matches", k)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
