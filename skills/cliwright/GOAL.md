# GOAL — Build a complete, production-grade CLI for any API

> Paste this whole file after `/goal` (or invoke the `cliwright` skill). It is a
> fully self-contained, model-agnostic brief that turns any HTTP API into a polished,
> distributable command-line tool built to a single production-grade standard — defined
> in full **in this document**. It depends on no other repository, codebase, or file:
> every pattern is inlined here, and the §8 skeletons are the source of truth.
> Fill in the **TARGET API** block, then execute the phases in order.

---

## TARGET API (fill this in — the only thing that changes per project)

```yaml
api_name:        # e.g. "Stripe", "Notion", "AdGuard Home"
vendor_url:      # marketing/site URL
docs_url:        # API reference root
docs_llms_txt:   # llms.txt / OpenAPI / Postman collection URL if one exists (huge time-saver)
base_url:        # e.g. https://api.example.com/v1   (+ sandbox/staging URL if any)
auth:            # one OR MORE of: api-key-header | basic(email:token) | bearer-token | oauth2-pkce
                 #   if the API offers several (e.g. a pasted token OR OAuth2, like Canvas),
                 #   ship each as a selectable method — see §1 "auth providers"
rate_limit:      # known limits + which response headers expose quota (e.g. X-RateLimit-Remaining)
pagination:      # one of: offset(start/limit) | page/per_page | cursor | Link-header
resources:       # priority-ordered list of the resources to ship first, with key fields each
                 #   e.g. customers(id,name,email,created), invoices(id,number,total,status)
special_ops:     # long-running jobs, file up/download, fiscal/e-invoice, webhooks, bulk import — if any
profile_flag:    # NAME of the multi-profile selector flag, so it reads naturally per API:
                 #   --bot (Telegram: a profile IS a bot), --instance (n8n), --account (accounting).
                 #   Default: profile. --profile is ALWAYS kept as a hidden alias for back-compat.
# Identity / distribution
binary_name:     # e.g. "stripe" (the command users type)
module_path:     # e.g. github.com/<owner>/<repo>
repo_owner:      # GitHub owner/org
homebrew_tap:    # e.g. <owner>/homebrew-<repo>
license:         # default MIT
distribution_scope: # HOW FAR to go — one of: local-build | +commit | +push | +release (default: local-build)
                 #   the agent NEVER creates a remote repo, pushes, or publishes a release
                 #   beyond this scope. Going further is the USER's decision, not the agent's.
release_flow:    # only if scope includes +release: local-goreleaser | ci-tag (default: ci-tag)
fix_priority:    # default: fix HIGH+MED, defer LOW with a tracked note
doc_scope:       # public-docs detail: minimal | standard | full (default: standard)
```

**Research first, ask second.** Everything about the *API itself* (auth, base URL,
pagination, rate-limit headers, resources, fields, special ops, JSON quirks) is your
homework — fetch the docs/OpenAPI and fill it in yourself. Do **not** ask me what the
docs already answer. Only ask me what the web cannot know: my environment, my
preferences, and my distribution identity (see §0). If the TARGET block + docs give you
everything, skip questions and start building.

---

## 0. Requirements: research first, ask only what the web can't answer

**Step 1 — Auto-resolve from the API's own docs (don't ask me these).** Fetch `docs_url` /
`docs_llms_txt` / OpenAPI / Postman collection and fill in every field you can. These are
*facts about the API*, not preferences — determine them yourself:

- **Auth model(s)** → token storage + the `auth` UX. One **or more** of: `api-key-header` /
  `basic` (email:token, Alegra) / `bearer` / `oauth2-pkce` (refresh + local/OOB, Canvas).
  If the API offers several (Canvas takes a pasted personal token **or** OAuth2), implement
  each as a selectable method behind one `Authenticator` interface — see §1 "auth providers".
- **Base URL** pattern + sandbox/staging URL.
- **Pagination model** → drives the `--all` walker: offset(start/limit) / page(per_page) /
  cursor(nextCursor) / Link-header, plus the exact param names.
- **Rate-limit headers** → if they exist, adaptive limiter; if not, fixed RPS.
- **Resources + key fields**, and which are read-only or enterprise/plan-gated.
- **Special ops** → custom actions, file up/download, long-running jobs, webhooks.
- **JSON quirks** → IDs as string *and* number? money fields? `{id,name}` refs? fields
  that are sometimes an object, sometimes an array? These drive the flexible JSON types.
- **Binary-name collisions** on PATH → propose an alternative (e.g. `<name>ctl`); don't ask.
- **Existing CLI?** → search npm / GitHub / the vendor docs for a first-party or popular CLI
  for this API (e.g. `@n8n/cli`). If one exists, summarize its scope and the build-vs-adopt
  trade-off — but **do not decide for me and do not refuse to build**: surface it; the choice
  to build anyway (for coverage, fleet consistency, a missing feature) is **mine** (§0 Step 2).

If the docs are ambiguous or contradictory, **state the assumption you're making** and
proceed — don't turn it into a question.

**Step 1b — Enumerate the COMPLETE method/endpoint set BEFORE authoring the manifest
(non-negotiable).** The manifest (§11) must be *derived from an enumerated list of every method
the API exposes* — never from memory. Recall under-captures **silently**: a hand-curated manifest
looks perfectly consistent (every command it ships maps to a declared resource) while wrapping a
fraction of the API, and `spec-check` can't see the gap because it only proves CLI ⊆ manifest,
never manifest == full API. *(This is the tgctl failure — ~⅓ of the Telegram Bot API wrapped, the
shortfall invisible because the manifest was authored from the model's memory, not an enumeration.)*
So enumerate from a source, in order of preference:

1. **A machine spec exists** — OpenAPI/Swagger, a Postman collection, or `llms.txt`: parse it and
   take its operation list as ground truth (e.g. count `jq '.paths | to_entries[] | .value | keys[]' openapi.json`).
2. **No machine spec (RPC-style / non-OpenAPI APIs)** — scrape the docs' **full method index**:
   every method anchor/heading on the API reference page (Telegram lists ~one `<h4>` per method —
   `sendMessage`, `getUpdates`, …). Count them. Better still, adopt a **community machine spec**
   when one exists: for Telegram, `ark0f/tg-bot-api` publishes a generated, machine-readable Bot
   API spec — use it as the enumeration source rather than hand-listing methods from memory.

Record the enumerated total **and its source** in the manifest (`api_method_total` +
`api_method_source`, §11). `make spec-completeness` then **fails the build** when the manifest
covers materially less than ~90% of that total without a recorded waiver in `DECISIONS.md` — so
under-capture is a loud gate failure, not an invisible default.

**Step 2 — Ask me only what the web cannot know** (batch into ≤4 questions, skip any
already answered in the TARGET block):

- **My environment** — which instance/host/deployment I use and its base URL
  (e.g. self-hosted vs Cloud), since that's not a public fact.
- **My preferences/scope** — which resources to ship first, read-only vs full CRUD,
  multi-instance/profiles vs single-profile; **how far to distribute** (`distribution_scope`:
  local-build → +commit → +push → +release); and, **if an official/competing CLI exists**,
  whether to build anyway or adopt it — my call, not yours.
- **My distribution identity** — repo owner, module path, Homebrew tap name, license,
  and whether to package a Claude Code skill/plugin.

If everything needed is already present, **skip the questions and start building.**

Keep a checked-in API manifest **derived from the enumerated method list** (§11), so the CLI
surface can be diffed against it for both **consistency** (`make spec-check` — CLI ⊆ manifest)
and **completeness** (`make spec-completeness` — manifest covers ≥ ~90% of the enumerated API).

---

## 1. The standard (non-negotiable practices)

Default stack: **Go + Cobra + GoReleaser** (the practices below assume it). **No Viper** —
resolve config with a plain manual precedence helper (`firstNonEmpty(flag, env, file, default)`),
the house pattern across the reference CLIs; don't pull a config framework. The principles
transfer to other languages, but unless I say otherwise, build it in Go.

**Every CLI you produce MUST have:**

- **A generic core, thin resources.** Adding a resource is ~3 small files and
  **zero edits to shared code**. The CRUD/pagination/retry/output logic is written
  once, generically, and reused for every resource.
- **Output formats: table (default), json, yaml, csv** (+ `-o id` for one id/line, pipeable to
  `xargs`, and a global `--jq` gojq escape hatch). One renderer for all resources, driven by
  JSON normalization. Global `-o/--output`. Table is colored only on a TTY; honor `NO_COLOR`
  and `--no-color` **wired in the renderer** (isatty/`NO_COLOR`), not just claimed. Column
  order is **deterministic** — a preferred-key list then alphabetical, never map-iteration
  order. `--columns` selects fields; cap auto-columns (~10) with a stderr note; rune-aware
  truncate wide cells with `…` + a stderr hint to use `-o json`. `--quiet` suppresses chatter.
  **CSV cells MUST be sanitized against formula injection (CWE-1236):** neutralize a leading
  `= + @` (and a leading `-` that isn't a real negative) so a crafted value can't execute in
  Excel/Sheets. Keep notes/warnings on **stderr** so stdout stays pipe-clean.
- **Config precedence: flag > env > config file > default**, resolved with a manual
  `firstNonEmpty(...)` per field (no Viper). Config at `~/.<binary>-cli/config.yaml` (or
  `$XDG_CONFIG_HOME`) — **dir `0700`, file `0600`, written atomically** (temp-in-same-dir +
  rename, never a torn write). Env overrides are namespaced `<BINARY>_*`, but **pick the real
  names per service** — the secret var is *not* uniformly `<BIN>_TOKEN` (it's `N8NCTL_API_KEY`,
  `ADGUARD_PASSWORD`, `ALEGRA_TOKEN`, …). Named **profiles**/instances for multi-account, with the
  selector flag **named per API** so it reads naturally (the manifest's `profile_flag`/`profile_noun`,
  default `--profile`; see §3) — `--bot` for Telegram, `--instance` for n8n, `--account` for
  accounting — and `--profile` kept as a hidden alias for back-compat;
  **validate** profile names (reject `/ \ : * ? " < > |` — traversal) and base URLs (require
  `http|https` + host, and **reject plain `http://` for non-loopback hosts** — cleartext leak).
- **Secrets never in plaintext.** Tokens live in the OS keyring
  (`zalando/go-keyring`: macOS Keychain / Linux Secret Service / Windows Cred Mgr),
  with an encrypted-file fallback. Never write a token to config-in-repo, code, or
  commit messages. Redact `Authorization` in dry-run unless `--show-token`.
- **Auth providers (when the API supports more than one method).** Don't hardcode a single
  scheme. `internal/auth` exposes an **`Authenticator` interface** — `Apply(req)` (+
  `Refresh(ctx)`/`Validate(ctx)` for OAuth) — with **one implementation per method**
  (`api-key-header`, `basic`, `bearer`/static-token, `oauth2-pkce` with `--mode auto|local|oob`).
  The **profile records which method** (non-secret, in `config.yaml`); the credential lives in
  the keyring (OAuth also stores a refresh token + expiry). `auth login` selects the method (a
  `--method` flag, or per-method subcommands like `auth token set` plus an OAuth path); the
  client resolves the active `Authenticator` from the profile and applies it per request.
  A single-method API uses exactly one implementation — this scales down to the simple case.
- **Resilient client.** Exponential backoff with **full jitter** (`random(0, base·2^n)` — a
  deliberate design, not a bug); **honor `Retry-After`** (delta-seconds *and* HTTP-date) before
  computing backoff; retry 429 + 5xx + transient network errors; **only auto-retry idempotent
  methods** (GET/HEAD/PUT/DELETE/**OPTIONS**) — never silently retry POST/PATCH. Rate limiting
  has **two branches**: if the API exposes quota headers, read them and slow as the budget
  depletes; if it does **not**, use a fixed RPS with **halve-on-429 + gradual restore**.
- **Actionable errors.** A typed `APIError{StatusCode, Code, Message, Details, Body}`
  whose `Error()` appends a **hint** keyed by status (401→"run `<bin> auth login`",
  403→"check permissions", 404→"verify id with `list`", 429→"rate limited — slow down",
  5xx→"server error, usually transient"). Map any domain-specific error codes to remedies.
- **`--dry-run`** prints the **equivalent `curl`** (shell-escaped, header-redacted) and
  performs no request. Indispensable for debugging and for teaching.
- **Cancellable on Ctrl-C.** `main` builds a `signal.NotifyContext(ctx, SIGINT, SIGTERM)`
  and calls `rootCmd.ExecuteContext(ctx)`; **every** call site uses `cmd.Context()`, never
  `context.Background()`. Then a Ctrl-C cancels in-flight `--all` pagination, retry backoff,
  and multi-step loops mid-request. (Easy to forget — a fresh cobra app defaults to
  `context.Background()` everywhere, so this is a deliberate wiring step, not free.)
- **AI-agent-ready by construction.** Every resource command carries MCP tool annotations
  (read-only / write / destructive) set **once** in the generic builder, **as each resource is
  added (Phase E)** — don't defer to a later pass; retrofitting onto dozens of commands is
  tedious. The annotations are ophis's exported constants (`AnnotationReadOnly` /
  `AnnotationDestructive` / `AnnotationOpenWorld` / `AnnotationIdempotent`), which map to the
  **singular** MCP keys (`readOnlyHint`, …); **there is no "write" key** — a write = set
  `openWorldHint` with read-only/destructive absent. An `mcp` server exposes the commands as
  MCP tools and an `agent guard` turns the same annotations into host safety rules — see §3b.
- **File-reading features are path-confined.** If the CLI ever reads a local file named by
  *data* (imports, externalized code, a `$ref` inside a record), confine the read to its
  base directory: reject absolute paths and `..` escapes **and** resolve symlinks before
  reading. Otherwise a crafted input file is an arbitrary-local-file-read — and, on a
  command that then uploads, an exfiltration primitive. Namespace any sentinel you embed in
  user data (e.g. `$<binary>_file`, not bare `$ref`) so it can't collide with a real field.
- **Standard meta-command set** (every CLI ships these — see Phase 3).
- **Comments explain WHY, not WHAT.** `gofmt -s` clean; passes `golangci-lint` and
  `go vet`; security-clean under `gosec` + `govulncheck` (CI-only — these track the *Go
  toolchain* version; a local run can fail on stdlib CVEs unrelated to your code).
- **≥80% test coverage as a ratchet**, enforced in CI — new code ships with its tests **in the
  same commit**, and every feature batch re-runs the gate before committing (the line breaks on
  every batch otherwise). `httptest` mock servers; table-driven tests; fuzz the flexible JSON
  decoders; test failure paths, not just happy paths. **Portable tests (Windows CI runs the
  matrix too):** build every path with `filepath.Join` and write scratch files under
  `t.TempDir()` — **never hardcode `/` separators or Unix paths** (a literal `/tmp/...`,
  `$HOME/.config`, or `"a/b"` fails on Windows; this broke `TestDirAndPath_XDG`,
  `TestConfig_SaveLoadRoundTrip`, `TestFileStore_RoundTrip` in practice). **Capture command
  output via cobra, not the real stdout:** prefer `cmd.SetOut(&buf)` / `SetErr(&buf)` with a
  `bytes.Buffer` (deterministic, race-free). If a test genuinely must hijack `os.Stdout` (e.g. a
  command that writes to `os.Stdout` directly, like `completion`), it **MUST drain the
  `os.Pipe()` reader concurrently in a goroutine** *before* invoking the code — a
  read-after-write helper deadlocks once the program writes more than the OS pipe buffer (the
  writer blocks with no reader; the buffer is ~64KB on Linux/macOS but far smaller on Windows,
  so `completion bash` hung Windows CI to the timeout). And **reset the cobra command tree
  between tests** (subcommand flags persist on a shared root and leak across cases).

---

## 2. Architecture (the blueprint)

```
cmd/<binary>/main.go        entry point; alias expansion BEFORE cobra.Execute(); ldflags version vars
commands/
  root.go                   global persistent flags, getAPIClient(), render(), PersistentPreRun (color/update)
  generic.go                generic CRUD + custom-action command builders (registerResource)
  auth.go config.go init.go doctor.go completion.go alias.go api.go version.go
  mcp.go                    registers the `mcp` subtree via ophis.Command (§3b)
  agent.go agent_hosts.go   `agent guard` — classify the tree + emit host safety config (§3b)
  <resource>.go             one per resource; self-registers via init()
  internal/options/         (optional, Pattern B style) option structs with Validate()
  internal/logging/         (optional) structured command logging
internal/
  api/
    client.go               auth, retries, adaptive rate limit, dry-run, GetJSON/Do
    resource.go             generic Resource[T]: List/ListAll/Get/Create/Update/Delete/Action
    pagination.go           the API's pagination model; decodeList normalizes data/results/rows wrappers
    types.go                flexible JSON types: ID, Int, Money, Ref/Refs, StringOrSlice
    errors.go               APIError + hint mapping
    retry.go ratelimit.go   backoff + adaptive limiter
    <resource>.go           one per resource: struct(s) + Client accessor (e.g. c.Widgets())
  auth/                     keyring + encrypted-file token storage (+ OAuth2/PKCE if needed)
  config/                   profiles + env overrides + manual precedence (no Viper)
  output/                   table/json/yaml/csv formatter
  version/                  build metadata (set via ldflags)
tools/gendocs/              generates docs/commands/*.md from the cobra tree
```

**Two resource patterns. Pick one by the decision rule in §11 (Determinism) — not by taste —
and stay consistent across the whole CLI:**

- **Pattern A — Generic-core (default; for uniform REST CRUD APIs).**
  `internal/api/resource.go` exposes `Resource[T any]` with
  `List(ctx, ListParams) / ListAll / Get(ctx, id) / Create / Update / Delete /
  Action(ctx, id, action, body, out)`. `commands/generic.go` builds the
  list/get/create/update/delete subcommands from a `resourceSpec[T]{Use, Aliases,
  Short, New, Columns, OrderFields, ListFilters, NoCreate/NoUpdate/NoDelete,
  UpdateMethod, Extra}`. `UpdateMethod` (PUT default; PATCH where the API requires it) is a
  generic-core **knob**, not a per-resource override — set it in the spec, never copy-paste CRUD.
  A new resource = a type + a `Client` accessor + one `registerResource(...)` in `init()`.

- **Pattern B — Service-layer (only for irregular APIs** with per-resource includes,
  masquerade, or special non-CRUD endpoints). Each resource gets a
  `XxxService struct{ client *Client }` with explicit methods + `ListXxxOptions`, plus a
  command file using an option struct with `Validate()` and structured logging.

**Flexible JSON types (`internal/api/types.go`)** — these prevent the most common
real-world API breakages, so include them by default:
- `ID` — unmarshals from string **or** number, always marshals as string (no precision
  loss above 2^53; consistent table rendering).
- `Int` — accepts a number **or** a numeric string; decode `Int64` before `Float64` to avoid
  >2^53 precision loss; reject `NaN`/`Inf` and validate against the JSON-number grammar.
- `Bool` — accepts a real bool **or** `"true"`/`"1"`/`"yes"`.
- `Money` — stored/emitted as exact decimal **text**, never `float64` (no rounding); same
  `NaN`/`Inf` rejection.
- `Ref`/`Refs` — `{id,name}` nested objects; `Refs` accepts a single object **or** an array.
- `StringOrSlice` — accepts `"x"` or `["x","y"]`.
- Unknown JSON fields are ignored, so structs need not be exhaustive.

**Adapt to the API — these are defaults, not dogma (decide by rule, not vibe).** The standard
above is *n8n-cli's design generalized*; a different API shape can justify dropping pieces — but
make it a **rule**, recorded in `DECISIONS.md`, so it stays deterministic (§11):
- **Flexible types** by default — *unless* the API ships a strict typed OpenAPI → then plain
  typed structs (`int64`/`time.Time`/`float64`) are cleaner (exactly Pattern B's case).
- **Profiles/multi-instance** by default — *unless* the API is a single fixed instance/appliance
  → then one config, no profiles (e.g. a self-hosted box with one host).
- **CSV** by default — *unless* the data isn't tabular; omit it rather than fake it.

---

## 3. Standard meta-command set (build these regardless of API)

| Command | UX |
|---|---|
| `auth login` | Interactive (or flag-driven) credential capture; **verify against a `whoami`-style endpoint**; store token in keyring, non-secret bits (incl. the chosen **method**) in config. If the API has **>1 auth method**, select with `--method` (or per-method subcommands like `auth token set` + an OAuth path); OAuth2+PKCE with `--mode auto\|local\|oob`. See §1 "auth providers". |
| `auth logout` | Remove stored token(s) for the active profile/instance. |
| `auth status` (alias `whoami`) | Show profile/instance, base URL, identity, auth validity. |
| `config path / view / set / use <profile> / list-profiles` | Inspect & edit config; **redact secrets** in `view`. |
| `init` / `setup` | First-run interactive wizard: pick base URL, capture creds, write config + keyring, smoke-test. |
| `doctor [--json]` | Diagnostics: config present, creds resolvable, connectivity, auth works, clock, version. Exit non-zero on failure. |
| `completion [bash\|zsh\|fish\|powershell]` | Shell completion; ship completions in archives/packages. Add dynamic completion for `--columns`/resource names. |
| `alias set/list/remove` | User-defined command aliases, expanded **before** cobra parses; never shadow built-ins. |
| `api <METHOD> <PATH> [-d body] [-q k=v]` | Raw authenticated request escape hatch (honors `--dry-run`, `--output`). |
| `version [--json] [--check]` | Print version/commit/date (from ldflags); `--check` compares against latest GitHub release. |
| `mcp [start\|stream\|tools\|claude\|cursor\|vscode]` | Run the CLI as an **MCP server** so AI agents drive the API; auto-exposes commands as annotated tools and installs host config (see §3b). |
| `agent guard --host <claude-code\|codex\|opencode>` | Generate **agent-safety** config that blocks destructive ops for an agent driving the CLI (see §3b). |
| `update [--check]` *(optional)* | Self-update from GitHub releases. |
| `repl` / `shell` *(optional)* | Interactive session with history, context vars, completion. |

Global persistent flags: `-o/--output`, the **multi-profile selector** (named from the manifest's
`profile_flag`; see below), `--base-url`, `--dry-run`, `--show-token`, `-v/--verbose`, `--no-color`,
`--columns`, `--quiet`, plus list flags `--all`, `--limit`, `--sort`, `--filter`.

**Name the multi-profile flag per API.** A generic `--profile` reads wrong for most services — a
profile *is* a bot for Telegram, an instance for n8n, an account for accounting. Take the flag name
(and help noun) from the manifest's `profile_flag`/`profile_noun` (default `"profile"` when nothing
fits better), and **keep `--profile` as a HIDDEN alias** so muscle memory and existing scripts never
break. Wire it once in `root.go` (and stamp the same noun into the generic command builder's help):

```go
// profileFlag/profileNoun come from the manifest (default "profile"). Both flags target the
// same var, so --bot and the legacy --profile are interchangeable; --profile is hidden.
rootCmd.PersistentFlags().StringVar(&profile, profileFlag, "", "named "+profileNoun+" to use")
if profileFlag != "profile" {
    rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "alias for --"+profileFlag)
    _ = rootCmd.PersistentFlags().MarkHidden("profile")
}
```

---

## 3b. Expose the CLI to AI agents (MCP server + agent guard)

Modern reference CLIs ship an **MCP server** and an **agent guard**. Both are nearly free
because they read the command tree you already built — *if* you annotated it (§1).

**MCP server (`commands/mcp.go`).** Use `github.com/njayp/ophis` (a Cobra→MCP bridge over
the official `modelcontextprotocol/go-sdk`). One `init()` registers the whole `mcp` subtree:

```go
rootCmd.AddCommand(ophis.Command(&ophis.Config{
    ToolNamePrefix: "<short>",   // tools become <short>_<resource>_<verb>
    Selectors: []ophis.Selector{{
        CmdSelector:           ophis.ExcludeCmdsContaining("agent","auth","config","alias","init","skills","doctor"),
        // Exclude the profile selector under BOTH its configured name (profileFlag, e.g. "bot")
        // and the hidden "profile" alias, so an agent can't switch instances either way.
        InheritedFlagSelector: ophis.ExcludeFlags("show-token", profileFlag, "profile", "api-key", "base-url"),
    }},
}))
```

- ophis walks the tree, auto-derives a tool per runnable leaf, and **replays the cobra
  command** on invocation — so every tool reuses the same client, keyring, profiles, and
  `--dry-run`. No separate handler layer.
- **Never expose secret/instance flags** (`--api-key`, `--show-token`, the profile selector
  under both its configured `profile_flag` name and the `--profile` alias, `--base-url`): the
  server uses whatever profile is active at startup, so an agent can't switch instances or read
  the key.
- Exclude setup/meta commands (`auth`, `config`, `init`, `alias`, `skills`, `agent`); a
  registry test (`TestMCPExcludesSetupCommands`) locks the surface.
- In the generic builder, tag each subcommand with ophis hint annotations so hosts gate
  writes: `readOnlyHints` (list/get/search/diff/schema/…), `writeHints` (create/update/…),
  `destructiveHints` (delete). Mark **unannotated custom verbs destructive by default**;
  read-only customs opt back in by calling `readOnlyHints` in their resource file.

**Agent guard (`commands/agent.go` + `agent_hosts.go`).** `agent guard --host
<claude-code|codex|opencode> [--all-writes] [--write]` classifies every API command (using
the same `openWorldHint`/`readOnlyHint` annotations) into read / write / irreversible, then
emits host safety config: hard-block irreversible verbs (delete, plus any domain-specific
irreversible ops like void/emit), make ordinary writes require approval, leave reads free.
For Claude Code it writes a `PreToolUse` hook + `settings.json` deny/ask rules; for Codex a
read-only sandbox `config.toml`; for OpenCode `opencode.json` permissions. It derives from
the **live tree**, so it stays correct across upgrades. Exclude the guard from the MCP
surface so an agent can't disable its own rails. Note the binary name (Bash patterns) may
differ from the MCP tool prefix (tool patterns) — thread both. MCP-only operation is the
hard guarantee; the Bash hook is best-effort (defeats quote/backslash tricks, not variable
indirection).

**Agent-guard hardening (MANDATORY — every one of these was a real, verified bypass or
dead-config bug found in a fleet-wide audit of 6 generated CLIs; do not ship without them):**

1. **The PreToolUse hook is required, not optional.** Permission deny rules alone are
   literal prefixes — `./bin/<bin> …`, `/usr/local/bin/<bin> …`, `env X=1 <bin> …`,
   `de""lete` quote-splits, and `;`/`|`/`&&` chaining all sidestep them. The hook is the
   enforcement layer; the rules are belt-and-suspenders.
2. **Anchored matching ERE (all four parts, exactly):** per blocked subcommand path `<P>`,
   match `(^|[;&|([:space:]]+)([^[:space:]]*/)?<bin>[[:space:]]+<P>([[:space:];&|)]|$)` on
   a cleaned string. The optional `([^[:space:]]*/)?` catches path-invoked binaries; the
   leading class anchors the command position; the **trailing class must accept separators**
   (`reset;true` glues `;` to the verb — space-or-EOL alone is a bypass for no-arg
   destructive commands). Clean first: strip `\042 \047 \134`, collapse newlines. Verify
   `my<bin>` (different binary with the name as a suffix) does NOT match.
3. **No-jq fallback must flatten JSON punctuation** (`tr '\n{}:,' '     '`) before matching:
   the compact payload glues the command to its key (`"command":"<bin> …"`), so without
   flattening the anchor can never match and the branch is silently **fail-open**. The
   no-jq test must build a strict PATH (symlink dir with only cat/tr/grep/sed) — merely
   prepending an empty dir leaves jq reachable and the branch untested (this exact test
   flaw masked the fail-open bug in two repos).
4. **Nothing that mutates remote state may classify as read or local.** Two real escapes:
   (a) verb-name collision — a leaf sharing its name with a read verb ("sync assignments"
   vs "analytics assignments") → keep a full-path write-override map checked before the
   read allowlist; (b) annotation gap — a hand-built command added without annotations
   falling through as "local/utility" (`packages import`, `company update`) → add a
   `TestEveryAPICommandIsAnnotated` walk with an explicit local-groups allowlist so
   unannotated commands fail the build, and default unannotated/Extra commands to
   write-or-destructive, never allowed.
5. **Enumerate cobra aliases.** `msg delete`, `wf delete`, `exec prune` bypass rules and
   hook that only list canonical paths — emit the full group-alias × verb-alias
   cross-product everywhere (rules, hook array, opencode). If the CLI has user-defined
   alias support (`<bin> alias set`), gate `alias set` itself.
6. **Raw-api escape:** verify the emitted method patterns match the command's REAL syntax
   (positional `api <METHOD> <PATH>` vs flag `-X` vs RPC method names — for RPC-style APIs
   allowlist `get*` at the method position instead). Block DELETE/PUT/POST/PATCH
   case-insensitively at the method position only, so a GET whose path contains "delete"
   stays allowed.
7. **Claude permission rules are literal prefixes, NOT regex.** `mcp__.*<bin>.*_(delete)`
   matches nothing — dead config that reads as coverage. Emit exact tool names. (Hook
   *matchers* in settings.json ARE regex; the two syntaxes differ.)
8. **Emit real host schemas.** Codex: top-level `sandbox_mode` / `approval_policy`
   (an invented `[sandbox]` table is silently ignored). OpenCode: `permission` (singular)
   with a `bash` sub-map. And `--write` must actually write the files (never overwriting).
9. **Ship the execution battery as tests** (`commands/agent_hook_test.go`): run the real
   generated hook under bash with jq payloads, asserting at minimum — blocked cmd,
   path-prefixed blocked cmd, glued-separator (`…;true`), quote/backslash/newline
   obfuscation, chained (`;`/`|`/`&&`), env-prefixed → all DENY; read cmd, blocked verb
   inside an argument, `cat <resource>_delete.go`, api GET with verb in path, `my<bin>`
   → all ALLOW; MCP exact blocked tool DENY, read tool + near-miss (`…_delete2`) ALLOW;
   plus the strict no-jq variants. Accepted, documented limitations: variable indirection,
   shell aliases/eval (MCP-only mode is the hard guarantee), and the conservative denial of
   a quoted blocked command inside an argument (`rg "<bin> orders refund" src/`) — the
   quote-stripping that defeats `de""lete` makes these indistinguishable; deny is the safe
   direction.

---

## 3c. Beyond the API (value-adds that differentiate)

Once the CRUD surface is complete, the highest-leverage additions go *past* the raw API —
they were the biggest differentiators in practice:

- **Declarative GitOps** for the central resource: `apply --dir <dir>` reconciles a
  directory of files (create / update / skip-unchanged via a canonical compare / `--prune`
  drift) with a `--dry-run` plan; `lint` (rules grounded in the API's own schema, not
  invented; exits non-zero as a CI gate), `diff`, and `convert` (JSON↔YAML, with long code
  fields **externalized** to sibling files for clean review). **Match by a stable handle if the
  API exposes one; if it doesn't (many don't — n8n keys by name), match by name and make
  duplicate-detect-and-skip mandatory** — never act on an arbitrary one of two same-named
  resources (that's the `--prune`-deletes-the-wrong-thing bug). Detect "unchanged" with a
  **field whitelist** (compare only the writable fields: name/body/settings/…), not a strip-list
  of volatile fields — whitelisting auto-drops `id`/`updatedAt`/server-managed noise.
- **Backup / restore** the instance to a git-friendly directory (YAML + externalized code);
  a partial backup must record failures and **exit non-zero**, never masquerade as success.
- **Cross-instance promote/sync** between profiles (`sync <id> --to <profile>`) — the
  multi-instance payoff; the official/competing single-instance tools can't do it.
- **Search across resources by content** (by node/field/credential), which the vendor UI
  usually can't.

These are optional, but a CLI that only mirrors the API is commoditized; these are what make
it worth installing over `curl` or the first-party tool.

---

## 4. Build procedure (execute in order; commit per phase)

**Phase A — Scaffold.** `go mod init <module_path>`; create the directory tree;
add `Makefile` (with the `verify` target), `.golangci.yml` (v2), `.githooks/pre-commit`,
`.gitignore`, `LICENSE`, `AGENTS.md` with `CLAUDE.md`→`AGENTS.md` symlink, the gate scripts
(`scripts/{spec-check,spec-completeness,cover-check,dod-check,judge}.sh`), and the checked-in
`api-manifest.json` — **enumeration-derived** with `api_method_total`/`api_method_source` (§0
Step 1b, §11). Wire version vars + ldflags. **Scaffold CI now so the linters track
the toolchain: build golangci-lint, gosec, and govulncheck FROM SOURCE with the job's Go
(`go install …@latest` in the workflow) — never pin a scanner version and never lean on the
prebuilt `golangci-lint-action` / `securego/gosec` Action. A pinned/prebuilt scanner lags the
Go toolchain and can't analyze a module already on the next Go release (e.g. a go1.25 module
under a Go-1.24 linter → "configuration contains invalid elements"); this cost real redo cycles
in practice. Pin `go-version` consistently across every job (use `stable`, or the module's
exact Go) so lint/security/test/build all run the same toolchain as `go.mod`.**

**Phase B — Client core.** `internal/api/{client,resource,pagination,types,errors,retry,ratelimit}.go`.
Implement auth, dry-run curl, adaptive rate limit, idempotent retry, the generic
`Resource[T]` (or service base), and the flexible JSON types. Unit-test the core
**once, thoroughly**, including fuzz tests for the JSON types.

**Phase C — Cross-cutting UX.** `internal/{config,auth,output,version}` and
`commands/root.go` (global flags, `getAPIClient()`, `render()`, `PersistentPreRun`).

**Phase D — Meta commands.** Phase 3 table: `auth`, `config`, `init`, `doctor`,
`completion`, `alias`, `api`, `version`.

**Phase E — Resource loop.** For each resource in `resources` (priority order), add the
**3 files** (api type + command registration + tests). The resource **set** is derived from the
spec/manifest (§11), not hand-picked; `resources` only sets priority. Test only what is *unique*
to the resource. **Annotate read-only/write/destructive as you add each resource — do not defer
to E2** (the retrofit is real pain). The generic builder stamps the standard verbs; set
`readOnlyHints` on any read-only custom verb in its resource file. Re-run `make verify` per batch
(coverage is a ratchet — new code ships with its tests in the same commit).

**Phase E2 — Agent surface.** Add `commands/mcp.go` (ophis) and `commands/agent.go` +
`agent_hosts.go` (§3b). Lock the tool surface and the read/write/irreversible classification
with tests (`TestMCPExcludesSetupCommands`, `TestClassifyAPICommands`, per-host guard tests).

**Phase E3 — Beyond the API (optional, high-leverage).** Add the differentiators in §3c that
fit this API: declarative `apply`/`lint`/`diff`/`convert`, backup/restore, cross-instance
`sync`, content search.

**Phase F — Tests & gates.** `httptest` helper `newTestClient(t, handler)`; table-driven;
`require` for fatal/setup, `assert` otherwise; fuzz the decoders. The gate is **`make accept`**
(§12) green — `make check` alone is **not** the gate (it skips spec-check, dod-check, coverage,
and the judge); run the full `make verify` for any change that touches the surface, not just at
the end.

**Phase F2 — Live smoke test (OPTIONAL — and potentially destructive).** Only if the user opts
in *and* provides a real instance/credentials: exercise reads, then a full write lifecycle on
**disposable, uniquely-named** resources (create → verify → update → delete). **Never** run a
non-dry-run write/prune against data you didn't just create, and **never** against production
unless the user explicitly says so — live writes are irreversible. If no instance is provided,
skip it and say so ("live-test skipped — mocks only"). It is never a hard gate.

**Phase G — Docs.** `tools/gendocs/main.go` generates `docs/commands/*.md` from the cobra
tree (`make docs-gen`); MkDocs Material site; `README.md` with install + quickstart.

**Phase H — Distribution & CI/CD.** Phases 5 & 6 below — **but only as far as
`distribution_scope` allows.** Default is `local-build`: build + local commits, **no remote
repo, no push, no release.** Creating the GitHub repo, pushing, and tagging a release are
**explicit, user-authorized** steps — do them only when the scope includes them, then stop.

**Phase I — Packaging.** Claude Code plugin + skill (Phase 7 below) if distributing there.

Make small, conventional commits (`feat:`, `fix:`, `docs:`, `chore:`). Default branch
model: `feature/*`/`fix/*` → `develop` → **release from `develop`, fast-forward `main` to the
tag, skip pre-releases** (`if: !contains(ref_name,'-')`). Commit locally regardless of
`distribution_scope`; only push/tag when the user authorized it.

---

## 5. Distribution (`.goreleaser.yaml` + Makefile)

**GoReleaser** must produce:
- **Builds**: `CGO_ENABLED=0`; GOOS `linux,darwin,windows` × GOARCH `amd64,arm64`
  (ignore windows/arm64). ldflags inject version/commit/date.
- **Archives**: tar.gz (unix) / zip (windows); include `README`, `LICENSE`, and
  generated shell completions.
- **Homebrew tap** (`homebrew_casks:`, **not** the deprecated `brews:`): auto-commit the
  cask to `homebrew_tap` on release (needs `HOMEBREW_TAP_TOKEN` secret, repo scope).
  When the cask name (`<binary>-cli`) differs from the binary inside the archive
  (`<binary>`), set `binaries: [<binary>]` (the **plural list** — `binary:` is deprecated
  and fails `goreleaser check`) or the symlink points at the wrong name and `brew install`
  can't find the binary. On macOS/arm64 an unsigned binary is **SIGKILL'd by Gatekeeper**;
  add a cask `postflight`/`hooks.post.install` that runs
  `xattr -dr com.apple.quarantine "#{staged_path}/<binary>"`.
- **Scoop** (`scoops:`): the target bucket repo (`scoop-<repo>`) **must already exist** —
  GoReleaser pushes to it but won't create it.
- **Linux packages** (`nfpms:`): deb/rpm/apk → `/usr/bin` + completions to standard paths.
- **Docker** (optional): push `ghcr.io/<owner>/<bin>:{version,major,latest}` with OCI labels.
- **Supply chain**: `checksums.txt`; **cosign** keyless signing (pin cosign v2.6.3 for v2
  bundle format; keyless needs CI OIDC — `permissions: id-token: write`); **syft** SBOMs.
- **Changelog**: group by `^feat`/`^fix`/`^perf`; exclude docs/test/chore/merge.
- **Run `goreleaser check` in CI** (a step in `ci.yml`, not just the snapshot job) so a
  deprecated/invalid config breaks on PR, not at tag time when the release is half-done.

**Makefile** (standard targets): `build install uninstall run dev check fmt vet lint
tidy test test-race test-coverage cover-check docs-gen docs-serve docs-build snapshot
setup-hooks clean`. `make check` = fmt + vet + lint + test (the local gate).

> **Install the pre-commit hook, don't just ship it.** Shipping `.githooks/pre-commit`
> + a `make setup-hooks` target is not enough — until someone runs it (sets
> `core.hooksPath=.githooks`) the hook catches nothing and bad code reaches CI.
> Run `make setup-hooks` once at repo setup. Make the hook fast: gofmt/vet/lint/
> `go test -short -race`, and **skip the Go gate when no `.go`/`go.mod`/`go.sum` is
> staged** so docs-only commits stay instant. CI remains the source of truth; the
> hook is just a local pre-flight that blocks the obvious breakage.

**Pre-commit hook**: gofmt (staged), go vet, golangci-lint (skip gracefully if absent),
`go test -short -race`.

---

## 6. CI/CD (`.github/workflows/`)

- **`ci.yml`** (push to main/develop + PR): `lint` (gofmt, vet, golangci-lint, **+ a
  docs-gen drift check**: run `make docs-gen` and `git diff --exit-code -- docs/commands`),
  `security` (govulncheck **blocking**; gosec **gating on high-severity** with
  `-severity high -confidence medium ./...`). **Build all three scanners FROM SOURCE with the
  job's Go** — `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`,
  `go install github.com/securego/gosec/v2/cmd/gosec@latest`, and
  `go install golang.org/x/vuln/cmd/govulncheck@latest` — then run the binaries directly.
  **Do NOT pin a scanner version and do NOT use the prebuilt `golangci-lint-action` /
  `securego/gosec` Action**: a pinned/prebuilt scanner lags the Go toolchain and can't compile
  or analyze a module already on a newer Go (the `x/tools` `invalid array length -delta*delta`
  failure, or golangci's "configuration contains invalid elements"). `@latest` compiled in the
  job always matches the module's Go, and govulncheck's vuln DB is fetched live so the binary
  version barely affects determinism. Pin `go-version` the same way in every job (`stable` or
  the module's exact Go) so lint/security/test/build share one toolchain. Standalone gosec
  honors `#nosec G404` (not golangci's `//nolint:gosec`), and for a CLI it false-positives on
  **G404** (non-crypto jitter RNG) and **G703/G304** (writing to a user's own `--out`/input
  path) — suppress those with a justified `#nosec`. `test` matrix (ubuntu/macos/windows;
  `-race`; coverage gate ≥80% on ubuntu; codecov), `build` (`goreleaser check` then GoReleaser
  snapshot, skip sign/sbom/docker).
- **`release.yml`** (tag `v*`): GoReleaser full pipeline (Buildx, GHCR login, cosign, syft,
  `release --clean`).
- **`docs.yml`** (push to main touching docs/commands): regenerate refs, `mkdocs gh-deploy --force`.
- **`dependabot.yml`**: weekly gomod + github-actions, grouped.
- Repo hygiene: `CHANGELOG.md` (Keep a Changelog), `SECURITY.md` (supported versions +
  private reporting + token-handling policy), `CODE_OF_CONDUCT.md`, `.github/ISSUE_TEMPLATE/*`,
  `pull_request_template.md`, `codecov.yml`.
- Optional: `claude-code-review.yml` + `claude.yml` for automated PR review / `@claude` mentions.

---

## 7. Package as a Claude Code plugin + skill (optional but recommended)

`.claude-plugin/plugin.json`:
```json
{
  "$schema": "https://json.schemastore.org/claude-code-plugin-manifest.json",
  "name": "<binary>-cli",
  "version": "<semver, synced to releases>",
  "description": "<one-line capability summary>",
  "author": { "name": "<you>", "url": "https://github.com/<owner>" },
  "homepage": "https://github.com/<owner>/<repo>",
  "repository": "https://github.com/<owner>/<repo>",
  "license": "MIT",
  "skills": "./skills",
  "keywords": ["<api>", "cli", "ai-agent"]
}
```

`.claude-plugin/marketplace.json`: `{ "$schema": ".../marketplace.json", "name": "<short>",
"owner": {...}, "plugins": [ { "name": "<binary>-cli", "source": "./", "version": "...",
"description": "...", "homepage": "...", "keywords": [...] } ] }`.

`skills/<binary>-cli/SKILL.md` — frontmatter then a usage guide:
```yaml
---
name: <binary>-cli
description: <long "use this when…" trigger description — what the CLI does and when to reach for it>
version: <semver>
homepage: https://github.com/<owner>/<repo>
license: MIT
allowed-tools: Bash(<binary>:*)
metadata: {"openclaw":{"category":"<domain>","emoji":"<emoji>","requires":{"bins":["<binary>"],"env":["<ENV_TOKENS>"]},"install":[{"kind":"brew","formula":"<owner>/<repo>/<binary>-cli","bins":["<binary>"]},{"kind":"go","package":"<module_path>/cmd/<binary>@latest","bins":["<binary>"]}]}}
---
```
SKILL.md body: Prerequisites/install → "prefer the CLI over raw curl" rationale →
auth/config setup → golden rules → workflow (auth → discover → act → verify) → command
cheatsheet → troubleshooting. Add `skills/<binary>-cli/references/*.md` deep-dives
(`auth-and-config.md`, `<api>-commands.md`, `output-and-filtering.md`).
If the repo keeps skills under `.agents/skills/`, symlink `.claude/skills → ../.agents/skills`.

---

## 8. Reference skeletons (generic-core style)

`internal/api/widgets.go`:
```go
package api

// Widget — see <docs_url>/widgets
type Widget struct {
    ID     ID     `json:"id,omitempty"`
    Name   string `json:"name,omitempty"`
    Status string `json:"status,omitempty"`
}

// Widgets returns a typed handle to the /widgets resource.
func (c *Client) Widgets() *Resource[Widget] { return NewResource[Widget](c, "widgets") }
```

`commands/widgets.go`:
```go
package commands

import "<module_path>/internal/api"

func init() {
    registerResource(resourceSpec[api.Widget]{
        Use: "widgets", Aliases: []string{"widget"}, Short: "Manage widgets",
        New:         func(c *api.Client) *api.Resource[api.Widget] { return c.Widgets() },
        Columns:     []string{"id", "name", "status"},
        OrderFields: []string{"id", "name"},
        ListFilters: []listFilter{{Flag: "status", Query: "status", Usage: "open,closed,draft"}},
        // NoCreate/NoUpdate/NoDelete for read-only resources; Extra for custom actions.
    })
}
```

`internal/api/widgets_test.go`:
```go
func TestWidgets_List(t *testing.T) {
    c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, http.MethodGet, r.Method)
        assert.Equal(t, "/widgets", r.URL.Path)
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`[{"id":"1","name":"Widget A","status":"open"}]`))
    })
    got, err := c.Widgets().List(context.Background(), ListParams{})
    require.NoError(t, err)
    require.Len(t, got, 1)
    assert.Equal(t, ID("1"), got[0].ID)
}
```

---

## 9. Definition of Done (acceptance checklist)

> This checklist is the **gate**, not a vibe check. §12 wires every item below into one
> `make accept` gate (a deterministic check per atomic item + a judge rubric for the few
> genuinely subjective ones; the judge lives in `accept`, not the routine `verify`). The
> loop's completion promise may fire **only** when `make accept` exits `0`. Split compound
> items (e.g. the output-formats line) into atomic
> checks — one per format, one per meta-command.

- [ ] `make check` green: gofmt-clean, vet-clean, golangci-lint pass, gosec+govulncheck clean, tests pass.
- [ ] Coverage ≥ 80%, enforced in CI. Flexible JSON types are fuzz-tested.
- [ ] All priority `resources` shipped with list/get/create/update/delete (or read-only) + per-resource tests.
- [ ] Manifest is **enumeration-derived** (`api_method_total` + `api_method_source` recorded) and
      covers ≥ ~90% of the enumerated API — `make spec-completeness` green (or a `coverage-waiver` in `DECISIONS.md`).
- [ ] Output works in table/json/yaml/csv; `--columns`, `--filter`, `--sort`, `--all`, `--limit` work.
- [ ] Auth flow works end-to-end; tokens in keyring; `auth status`/`doctor` verify against the live API.
- [ ] `--dry-run` prints a correct, copy-pasteable curl with the secret redacted.
- [ ] Retry honors idempotency; rate limiter reads quota headers; 429 handled gracefully.
- [ ] Ctrl-C cancels in-flight work (`signal.NotifyContext` + `ExecuteContext`; `cmd.Context()`
      everywhere, no stray `context.Background()`).
- [ ] Any file-reading-from-data feature is path-confined (no `..`/absolute/symlink escape).
- [ ] Meta commands present: auth, config, init/setup, doctor, completion, alias, api, version.
- [ ] **MCP server** (`mcp`) exposes commands as annotated tools (read-only/write/destructive),
      excludes setup/secret commands, and reuses the same client/keyring/dry-run.
- [ ] **Agent guard** (`agent guard --host …`) generates host safety config from the live tree.
- [ ] `goreleaser check` is clean and a snapshot build produces all targets. **A real tagged
      release (push, cosign + SBOM, Homebrew cask `homebrew_casks` + `binaries:`) is required
      ONLY if `distribution_scope` includes `+release` — it is not part of "done" for a local
      build.** When in scope, the published release must `brew`/`go install` and run
      (Gatekeeper-clean on arm64).
- [ ] CI (`ci.yml`), release (`release.yml`), docs (`docs.yml`), dependabot configured.
- [ ] Docs: generated command reference + README quickstart; AGENTS.md present.
- [ ] (If distributing via Claude Code) plugin.json + marketplace.json + SKILL.md + references shipped.
- [ ] Hygiene: CHANGELOG, SECURITY, LICENSE, issue/PR templates. No secret ever committed.

---

## 10. Process & verification (how to actually ship it well)

Building the surface is half the job; these steps caught the real bugs and raised quality:

- **Live-test the surface against a real instance — OPTIONAL, user-gated, potentially
  destructive.** Mocks miss real-API behavior (this caught a dry-run that short-circuited its
  own read and an apply that mis-detected "unchanged"), so it's high value — but run it **only
  if the user opts in and supplies an instance/credentials.** Create only disposable,
  uniquely-named resources, never touch existing data, and **never** run a non-dry-run
  destructive/prune against production. Live writes are irreversible — if unsure, stay on mocks
  and say "live-test skipped". Never a hard gate (§4 Phase F2).
- **Adversarial multi-agent review + rating** before (and after) a release: fan out
  reviewers by dimension (correctness, security, resilience, API-fidelity, tests), then
  **verify each finding against the code _and its comments_** before acting — **expect ~half to
  be false positives** (a "high-severity" finding may be inverting an intentional design, e.g.
  AWS full-jitter; refuting it with cited rationale is a valid outcome, recorded in
  `DECISIONS.md`). Don't let a finding's _rank_ drive fix _order_. A dual-lens rating (strict
  auditor + pragmatic maintainer) surfaces gaps a single pass misses. **Cascade caution:** when
  you tighten a shared validator/lint rule, pre-sweep the tests for throwaway fixtures (e.g.
  `rg '"nodes":\s*\[\]' -g '*_test.go'`) and fix them in the same change — give the fixture real
  data, don't loosen the rule, and don't discover them one `make verify` at a time.
- **Demo GIF via VHS** (`.demo/demo.tape`): record the real binary against a **local mock**
  with seeded fake data (not a real/private instance), so the public GIF is clean and
  reproducible. Do all setup (env + mock) **outside** the tape (or `Hide` leaks frame 0);
  start the recording with `Ctrl+L`.
- **Honest comparison docs** vs the first-party / competing tools: a "where the official
  tool is genuinely better" section builds more trust than a one-sided pitch.
- **Ship as completely as the user asked — no further.** When `distribution_scope` includes
  `+release`: doc site (MkDocs), generated command reference, Homebrew/Scoop + deb/rpm/apk,
  cosign + SBOM, the Claude skill/plugin, and a tagged release that `brew upgrade` actually
  serves, verified end-to-end. **At `local-build` (default), stop after a clean `make verify` —
  do not create a repo, push, or release.** How far to publish is the user's call (§0).
- **Public-docs hygiene.** Never put internal strategy, competitor put-downs, or unverified
  benchmark claims in shipped docs (README/MkDocs/skill). Honest comparison ≠ editorializing —
  keep that in `DECISIONS.md`/memory, not the public surface.

---

## 11. Determinism rules (same API in → same CLI out)

A generator's value is reproducibility: two runs on the same spec must converge on the same
surface. Remove the free choices a loop would otherwise amplify into drift.

- **Pattern choice is a rule, not a judgment.** Use **Pattern A (generic-core) by default.**
  Switch to **Pattern B (service-layer)** *only* when the spec shows a documented trigger:
  per-resource `include`/expansion params, user impersonation/masquerade, or endpoints that
  aren't CRUD-on-a-resource. Record the trigger that forced Pattern B in `DECISIONS.md`.
- **Derive the resource set and order from an ENUMERATED spec, not from memory or taste.**
  First enumerate the complete method/endpoint set from a source (§0 Step 1b) — OpenAPI/Postman/
  `llms.txt`, else the docs' full method index or a community machine spec (Telegram →
  `ark0f/tg-bot-api`). Resources = every collection in that enumeration; order them
  deterministically (by the spec's tag order, then alphabetical). Flag read-only ones from the
  spec (no documented write verbs). Don't hand-pick "the important ones" and don't list from
  recall — ship the full priority surface; the human's TARGET `resources` only reorders priority,
  it does not change membership. Recall-authored manifests under-capture invisibly (the tgctl
  ~⅓-coverage failure); the enumeration is what makes the resource *set* reproducible.
- **Pin every assumption.** When the docs are ambiguous, write the assumption to a
  checked-in `DECISIONS.md` (one line each: question → decision → why) and read it back on
  every iteration. Never silently re-decide; the loop must see the same decisions each pass.
- **The spec-derived manifest is a hard gate — in BOTH directions.** Generate a checked-in
  `api-manifest.json` from the enumeration, listing resources, fields, and verbs, **plus the
  enumerated `api_method_total` and its `api_method_source`** (and, for RPC-style APIs, a flat
  `methods` array). Two gates anchor it:
  - **Consistency** — `make spec-check` fails when the built CLI surface diverges from the
    manifest (CLI ⊆ manifest: every command maps to a declared resource/verb).
  - **Completeness** — `make spec-completeness` fails when the manifest covers materially less
    than ~90% of `api_method_total` (manifest ≈ full API). Covered = `resources[].verbs` +
    `methods[]`; the denominator is the enumerated total. A deliberate shortfall (e.g. "read
    surface first, writes in v2") is allowed **only** with an explicit `coverage-waiver` line
    recorded in `DECISIONS.md`, so the loop sees the same decision every pass.

  spec-check alone never noticed under-capture (it only checks the direction that's already
  consistent); the completeness gate is what keeps "done" meaning the *same, full* surface
  every run.
- **Stable output.** Canonical field order (struct/JSON-tag order), stable default
  `--columns`, deterministic table/json/yaml/csv rendering — never rely on map-iteration order.

---

## 12. Acceptance gate (`make accept`) — the loop's exit condition

The build is finished only when one command proves it. Wire §9 into TWO gates: a
deterministic `verify` (what CI and routine `make` runs use), and `accept` (= `verify` +
the LLM `judge`) that the build loop binds to. The judge spends tokens and needs an agent,
so it must NOT run on every CI/dev `make verify`:

```make
verify: check spec-check spec-completeness   ## DETERMINISTIC gate (CI/dev); exit 0 == green
	go test ./... -coverprofile=coverage.out
	./scripts/cover-check.sh 80          # coverage ≥ 80% or fail
	./scripts/dod-check.sh               # one concrete check per atomic §9 item
judge: ; ./scripts/judge.sh             # LLM rubric for the few subjective items (build-time only)
accept: verify judge                    ## full build-acceptance; the /goal loop binds to THIS
```

- **`make check`** — fmt + vet + golangci-lint + gosec + govulncheck + tests (the §5/§6 gate).
- **`spec-check`** — built surface ⊆ `api-manifest.json`: every command maps to a declared
  resource/verb (consistency, §11).
- **`spec-completeness`** — `api-manifest.json` covers ≥ ~90% of the enumerated `api_method_total`
  (completeness, §0/§11). Fails on under-capture unless a `coverage-waiver` is recorded in
  `DECISIONS.md`. This is the gate that catches a memory-authored manifest wrapping a fraction of
  the API while every shipped command still looks consistent.
- **`dod-check.sh`** — a deterministic check per atomic §9 item: greps/smoke-tests that
  `mcp.go`, `agent.go`, each of the four output formats, the `--dry-run` curl,
  `signal.NotifyContext`, keyring storage, and every meta-command exist and behave. Exits
  non-zero on any miss.
- **`judge.sh`** — for criteria a grep can't prove (*errors carry actionable hints*,
  *comments explain WHY*, *help text has examples*), an LLM rubric scores them and fails
  below threshold. Keep this set small; prefer a deterministic check whenever one exists.

**Completion-promise binding (the anti-cheat).** When driven by `/goal` (or any Ralph-style
loop), the completion promise (e.g. `<promise>CLI COMPLETE</promise>`) may be emitted **only
after `make accept` exits `0`**. Do **not** emit a false promise to escape the loop — not
when stuck, not when "close enough," not for any reason. If the gate fails, read the failure,
fix the smallest thing, and iterate. `make accept` is the single source of truth for "done
and high."

---

## Guardrails

- **Auth** and **pagination** are structural — *determine* them from the docs/OpenAPI,
  don't invent them and don't ask me. Only ask if the docs genuinely don't specify.
- Read the API's OpenAPI/llms.txt for field names; never fabricate fields.
- Write the generic core once; never copy-paste CRUD per resource.
- Keep resource files thin; if you're editing shared code to add a resource, the
  abstraction is wrong — fix the abstraction. A custom verb that needs an extra flag must
  **extend** the generic command (the `resourceSpec` `Extra`/customizer), never
  `RemoveCommand()` + re-implement (that forks the implementation and drifts). Every command
  that talks to the API goes through the typed `Client`; a raw-HTTP escape hatch like `proxy`
  is the one documented exception — call it out as such.
- Never print or commit a real token; redact by default in dry-run and `config view`.
- Never expose secret/instance flags (`--api-key`, `--show-token`, the profile selector — under
  both its configured `profile_flag` name and the `--profile` alias — and `--base-url`) to the MCP
  tool surface, and exclude the `agent guard` command from it — an agent must not read the key,
  switch instances, or disable its own rails.
- Wire `cmd.Context()` from `ExecuteContext` through every call; a stray `context.Background()`
  silently breaks Ctrl-C cancellation. Annotate every resource command (read-only/write/
  destructive) in the generic builder, not per-command later.
- **Same API in → same CLI out:** choose the resource pattern by §11's rule, derive the
  resource set/order from the spec, and pin every assumption in `DECISIONS.md` — never
  re-decide per iteration.
- **Done is what `make accept` proves, not what you assert.** Emit a loop completion promise
  only after the gate exits `0`; never a false one. The gate is **`make accept`** (`verify`
  plus the LLM judge), not `make check` — for **every** change that touches the surface or a
  documented behavior, not
  just at first build.
- **The agent informs; the user decides how far.** Default `distribution_scope` is
  `local-build`: build + local commits only. **Never** create a remote repo, push, or publish a
  release on your own — those are explicit, user-authorized steps. Go exactly as far as the
  scope says, then stop.
- **An existing official/competing CLI is not a veto.** Detect and surface it with the
  build-vs-adopt trade-off, then build if the user wants to (coverage, fleet consistency, a
  missing feature). Never refuse to build or silently pivot to "adopt theirs".
- Comments explain WHY. Match the surrounding code's style.
