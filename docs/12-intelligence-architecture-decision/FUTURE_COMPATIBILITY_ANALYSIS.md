# FUTURE_COMPATIBILITY_ANALYSIS

Nine providers evaluated against Options A, B, and C.
Wire-protocol facts are Tier-3 (industry) evidence; everything about **AWIS's** ability to host them is Tier-1 (code).

---

## 1. Protocol grouping

| Provider | Wire protocol | Auth | Endpoint configurable? | Model identity required? |
|---|---|---|---|---|
| **Anthropic** | Messages API (`/v1/messages`) | `x-api-key` + `anthropic-version` | Needed for proxies | Yes |
| **OpenAI** | Chat Completions (`/v1/chat/completions`) | `Authorization: Bearer` | Rarely | Yes |
| **OpenRouter** | OpenAI-compatible | Bearer + optional `HTTP-Referer`, `X-Title` | **Mandatory** | **Yes — namespaced** (`anthropic/claude-…`) |
| **Together** | OpenAI-compatible | Bearer | **Mandatory** | Yes |
| **Groq** | OpenAI-compatible | Bearer | **Mandatory** | Yes |
| **LM Studio** | OpenAI-compatible | **None** | **Mandatory** | Yes |
| **vLLM** | OpenAI-compatible | Optional | **Mandatory** | Yes (single served model) |
| **Ollama** | Native `/api/chat` **+** partial OpenAI-compat `/v1` | **None** | **Mandatory** | Yes (tag, e.g. `llama3.1:8b`) |
| **Gemini** | `/v1beta/models/{model}:generateContent` | `x-goog-api-key` / query param | Rarely | **Structural — model is in the URL path** |

**Grouping: 6 providers are unambiguously one protocol** (OpenAI, OpenRouter, Together, Groq, LM Studio, vLLM). Ollama is a seventh if its `/v1` surface suffices, an eighth driver if not. Anthropic and Gemini are genuinely distinct.

---

## 2. Per-provider hosting analysis

Each row states what AWIS must have before the provider can run, and which option supplies it.

### Anthropic — *works today, is the only one that does*
Already implemented (`adapters/anthropic/`). Under **A** it stays one package. Under **B** it becomes the `anthropic-messages` driver with one instance. **No migration risk in either.** Note its `Config` already exposes `BaseURL`, `MaxAttempts`, `ModelFast`, `ModelQuality` — it is structurally a driver already.

### OpenAI — *the cheapest second provider under both options*
Needs Bearer auth (current adapter hardcodes `x-api-key` + `anthropic-version`, `anthropic.go:264-266`) and a chat-completions body. **A:** new package. **B:** new driver — identical work. *For this provider alone, A and B cost the same.* The divergence begins at the third.

### OpenRouter — *the first provider where A and B diverge sharply*
Same wire protocol as OpenAI. Adds: mandatory `base_url` (**B-4**), optional custom headers, and **namespaced model IDs** that make `model_hint`'s three values structurally insufficient (**B-5**).
**A:** a second near-duplicate of the OpenAI package — duplicating retry, usage parsing, error mapping, structured output.
**B:** a config entry.

### Together — *identical to OpenRouter*
**A:** third near-duplicate. **B:** config entry.

### Groq — *identical again*
**A:** fourth near-duplicate. **B:** config entry.
Groq and Together are the clearest evidence that the long tail is real rather than hypothetical: neither appeared in the original six-provider brief, both were added later, and both are pure config entries under B.

### LM Studio — *the provider that breaks the availability model*
Keyless. `IsAvailable()` returns `cfg.APIKey != ""` (`anthropic.go:124`), and the router skips any adapter returning false (`router.go:134`). Any implementation following the established repository pattern would be **configured, healthy, and never routed to** (**B-7**).
This is an *interface-level* gap and **neither A nor B fixes it.** It requires a real liveness probe, which the interface currently forbids (`IsAvailable()` is documented as making no round-trip). **Both options must address B-7 independently.**

### vLLM — *the provider that breaks the identity model*
Keyless or optionally keyed, mandatory `base_url`, serves one model.
The decisive case: **a realistic deployment runs two vLLM servers (prod and dev, or two GPU pools).** Under a registry keyed on adapter type, both register as `"vllm"` and `router.go:85-87` silently overwrites — `byName[r.Adapter.ProviderName()] = r`, plain map assignment, last-write-wins, no error.
**A survives this only if it adopts instance identity — i.e. only if it adopts half of B.**

### Ollama — *the provider that tests B's degradation path*
Keyless, mandatory `base_url`, model tags, frequently **not running** (so B-7 is acute, not theoretical).
Its OpenAI-compatible `/v1` surface is **partial** — parameter support and streaming semantics differ from OpenAI's.
- If `/v1` suffices → seventh instance of `openai-chat`.
- If not → its own `ollama-native` driver.
**This is the honest test of B's dominance claim, and B passes: the fallback is "add a driver", which is exactly what A would have done anyway.** B is never worse here.

### Gemini — *the provider that proves OpenAI-compatibility is not universal*
Model in the URL path, `contents`/`parts` body, different auth, `responseSchema` for structured output.
**A:** new package. **B:** new driver — identical work.
Gemini is the direct refutation of any "just standardise on OpenAI-compatible" instinct, and it is why the recommendation adopts OpenAI-compat as *one driver* rather than *the seam*.

---

## 3. Compatibility matrix

✅ hosts cleanly · ⚠️ hosts with cost or caveat · ❌ requires architecture change

| Provider | **A** Registry Only | **B** Registry+Driver+Instance | **C** B+Policy |
|---|:--:|:--:|:--:|
| Anthropic | ✅ | ✅ | ✅ |
| OpenAI | ✅ | ✅ | ✅ |
| OpenRouter | ⚠️ duplicate package | ✅ config | ✅ |
| Together | ⚠️ duplicate package | ✅ config | ✅ |
| Groq | ⚠️ duplicate package | ✅ config | ✅ |
| LM Studio | ⚠️ duplicate + **B-7** | ⚠️ **B-7** | ⚠️ **B-7** |
| vLLM (single) | ⚠️ duplicate + **B-7** | ⚠️ **B-7** | ⚠️ **B-7** |
| **vLLM (two endpoints)** | ❌ **B-6 collision** | ✅ | ✅ |
| **Two OpenAI accounts** | ❌ **B-6 collision** | ✅ | ✅ |
| Ollama | ⚠️ duplicate + **B-7** | ⚠️ **B-7** | ⚠️ **B-7** |
| Gemini | ✅ | ✅ | ✅ |
| *10th OpenAI-compatible vendor* | ❌ code + release | ✅ config | ✅ config |
| *Novel protocol provider* | ✅ | ✅ | ✅ |

**Package count for the nine named providers: A = 9, B = 3 (4 if Ollama needs a native driver).**

---

## 4. What the matrix shows

**4.1 · A does not fail on any single named provider.** It hosts all nine. The honest verdict on A is *expensive, not broken* — and any decision package claiming A "cannot support" these providers would be overstating.

**4.2 · A fails on two things that are not single providers.**
- **Two endpoints of one vendor** — an ordinary deployment shape (prod/dev, two GPU pools, two accounts for rate-limit spreading), and a hard collision at `router.go:85-87`.
- **The tenth vendor.** Groq and Together were not in the original brief and arrived later; under A each is a package and a release, under B each is a config line.

**4.3 · B-7 is option-independent.** Keyless local providers (Ollama, LM Studio, vLLM) are blocked by `IsAvailable()`'s credential-check semantics under **all three options**. This must not be used as an argument for or against any of them — it is a prerequisite for all.

**4.4 · Protocol fragmentation does not favour A.** Gemini and Ollama-native are the two most plausible fragmentation cases, and in both, B does exactly what A does: adds a driver. **The forbidden assumption "future providers will behave like current providers" is not load-bearing for B** — B's benefit is opportunistic, and its downside is bounded by A.

---

## 5. Three-to-five-year projection

| Scenario | A | B | C |
|---|---|---|---|
| **Provider count stays at 2–3** | Fine. Marginally simpler. | Fine. Equal cost. | Overbuilt. |
| **Reaches the 9 named** | 9 packages, 9 retry paths, 9 prompt renderers | 3 drivers, 1 retry path, 1 renderer | Same as B + unused policy layer |
| **Long tail continues** (Cerebras, DeepSeek, Fireworks, Mistral — all OpenAI-compatible) | Linear package growth; each a release | **Constant** — config entries | Constant |
| **A major provider fragments** | New package | New driver | New driver |
| **Cost control becomes a requirement** | Needs C anyway | Needs C anyway — seam already built | Already built |
| **Multi-tenant / multi-account** | ❌ blocked by B-6 | ✅ | ✅ |

Over a 3–5 year horizon the differentiator is not the nine named providers — **all three options host them.** It is the tenth through twentieth, and the deployment shapes (two endpoints, two accounts) that the identity model either permits or forbids.

Both differentiators favour B, and neither depends on assuming protocol convergence continues: they depend only on it having *already happened* for six of nine named providers, which is a present fact rather than a forecast.
