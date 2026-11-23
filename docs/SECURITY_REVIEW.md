> Pre-adoption security review of the Relynk fork (Nov 2026). Round 1 audited upstream v1.11.0 (commit 575c6f0); Round 2 verified the fixes, the Initiatives feature and the release pipeline on this branch. Findings marked fixed are addressed in commits 3c8f788 and later.

# Security review: linear-cli (Relynk fork of joa23/linear-cli) at commit 575c6f0

## Verdict: SAFE WITH FIXES

I found no malicious code, obfuscation, telemetry, analytics or update checks. The binary only sends data to Linear (api.linear.app, linear.app OAuth, uploads.linear.app, and the signed upload URL Linear returns). Credentials are stored with 0600/0700 permissions and are never printed or logged.

The fork is identical to upstream: HEAD = origin/main = upstream joa23 HEAD = 575c6f0, with no Relynk-specific commits. The published upstream v1.11.0 Linux x86_64 binary was built from tag commit 70961d5 (`vcs.modified=false`), and its sha256 `ab6da7ba…46db` matches Formula/linear-cli.rb.

The problems are in the distribution path and a few hardening gaps, not in hostile behaviour. Fix items 1 to 4 below before rolling it out to every machine.

How I checked:
- `go build` and `go vet`: clean.
- `go mod verify`: all modules verified.
- `go test ./...`: all pass except `internal/config TestConfigManagerErrorHandling/permission_denied`. That test fails only because the sandbox runs as root, so it is not a product bug.
- govulncheck could not run: vuln.go.dev is blocked by this sandbox's egress proxy. Re-run it on a networked machine.
- I scanned the built binary's strings for URLs and checked `go version -m` on both the local build and the official release.

---

## Findings (ranked)

### HIGH

**H1. The Homebrew/release channel points at the upstream maintainer, not at this fork.** `Formula/linear-cli.rb:7,11,15,21,25` download from `https://github.com/joa23/linear-cli/releases/...`. `.goreleaser.yml:61-64` publishes to `owner: joa23`.
- Installing through the Formula or the tap trusts joa23's GitHub account and CI. Every future `brew upgrade` would pull unreviewed upstream code onto every developer machine, so this audit only covers v1.11.0.
- **Fix:** build from a pinned, reviewed commit of github.com/Relynk/linear-cli, using your own CI and a current Go toolchain. Distribute through an internal tap or artifact store, set goreleaser `release.github.owner: Relynk`, and update the Formula URLs. Review the upstream diff before every bump.

### MEDIUM

**M1. Release binaries are built with an outdated Go standard library.** `.github/workflows/release.yml:24-26` pins `go-version: '1.21'`, while `go.mod:3` says `go 1.24.2`. The toolchain therefore auto-switches to exactly go1.24.2.
- Confirmed: the official v1.11.0 binary reports `go1.24.2`.
- go1.24.2 is from April 2025 and lacks many later stdlib security fixes (net/http, net/url, crypto/x509, encoding/asn1/pem and others). This binary does HTTP and TLS against untrusted servers (for example `attachments download <any URL>`).
- Actions are tag-pinned, not SHA-pinned (`actions/checkout@v4`, `setup-go@v5`, `goreleaser-action@v6` with `version: '~> v2'`).
- `.goreleaser.yml:5-7` runs `go mod tidy` at release time, which can change go.mod/go.sum after review.
- **Fix:** build with the latest patched Go (set `go-version-file: go.mod` and bump `toolchain` to a current release). SHA-pin all actions and the goreleaser version. Drop `go mod tidy` from `before.hooks`. Run govulncheck in CI.

**M2. The OAuth callback server listens on all network interfaces.** `internal/oauth/oauth.go:168-171` and `:296-299` use `Addr: ":" + port`, default port 37412 (`internal/cli/auth.go:247`).
- During `auth login`, for up to 5 minutes, anyone on the LAN can hit `/oauth-callback`. The `state` check (`oauth.go:188,316`) blocks code injection, but a LAN peer can abort the login with `?error=x` (`oauth.go:266-270`).
- There is no PKCE. The authorize URL has no `code_challenge` (`oauth.go:86-111`), so the flow relies only on the per-user client secret.
- On state mismatch the function returns without `server.Close()` (`oauth.go:188-189,316-317`).
- `GenerateState` ignores the error from `rand.Read` (`oauth.go:397-400`). This is negligible on Go 1.24 or later.
- **Fix:** bind to `127.0.0.1:` + port. Add PKCE (S256). Close the server on every return path. Check the `rand.Read` error.

**M3. The token can be sent in cleartext to an `http://uploads.linear.app` URL.** `pkg/linear/attachments/client.go:652-658` (`isPrivateLinearURL`) matches only on hostname, not scheme. The bearer token is then attached at `client.go:313-320` and `:668-674`. The same host-only check exists in `internal/service/issueexport.go:139-145`.
- A URL taken from untrusted issue or comment content (`issues export` auto-downloads every uploads.linear.app link) would send the workspace-wide read/write token unencrypted to anyone on the network path.
- Cross-host redirects are safe: Go strips `Authorization` when a redirect leaves the domain.
- **Fix:** require `parsed.Scheme == "https"` before attaching the token, and reject non-https URLs for downloads.

**M4. The CLI's HTTP clients ignore `HTTPS_PROXY`/`HTTP_PROXY`, which matters for egress enforcement.** `pkg/linear/core/http_client.go:10-31,37` builds an `http.Transport` with no `Proxy` field. The API client, the OAuth token exchange (`auth.go:128` uses `core.GetSharedHTTPClient()`), downloads and uploads all use it.
- Verified: with `HTTPS_PROXY` set, the binary did a direct DNS lookup of api.linear.app via 8.8.8.8 instead of using the proxy.
- If company egress control relies on a proxy env var, this tool bypasses it, or fails where direct egress is blocked.
- **Fix:** add `Proxy: http.ProxyFromEnvironment` to the transport, and/or enforce the allowlist at the firewall or DNS layer.

**M5. A GitHub workflow lets anyone trigger an AI rewrite of skill files with write permission.** `.github/workflows/skill-apply-optimize.yml:2-17` runs on `issue_comment` from anyone whose comment contains `/apply-optimize` on any PR. It has `contents: write` and `pull-requests: write`, and there is no `author_association` or permission check. It runs the third-party action `tesslio/skill-review-and-optimize` (SHA-pinned) in apply mode.
- `skill-review.yml` sends `TESSL_API_TOKEN` to the same third party on `pull_request`.
- SKILL.md files are compiled into the binary (`internal/skills/embed.go:6`) and installed into agents by `linear skills install`. An outsider-triggered, AI-generated change to skill text is therefore a prompt-injection supply-chain path into every agent.
- **Fix:** delete both tessl workflows in the fork, or at minimum restrict them to `OWNER`/`MEMBER` and remove `contents: write`.

**M6. Agents get a workspace-wide token plus tools that read local files and upload them to Linear, and untrusted Linear content flows back to agents.** This is inherent to the design.
- Tokens are stored in plaintext JSON at `~/.config/linear/token` (`internal/token/storage.go:141-150,184-195`), mode 0600, with no OS keychain. The OAuth client secret is in plaintext at `~/.config/linear/config.yaml` (`internal/config/manager.go:51-56,124,135`; written by `auth.go:115-123`).
- Scopes are `read write` (user mode) or `app:assignable app:mentionable read write` (agent mode) (`oauth.go:92,105`). There is no admin scope.
- These commands read any local file the agent names and upload it to Linear with no path restriction: `attachments create --file` / `issues create|update|comment --attach` (`client.go:978-990,1085-1105`) and `documents create|update --content-file` (`internal/cli/documents.go:412`). A prompt-injected agent could upload ~/.ssh keys or .env files into Linear.
- Issue and comment text is written straight into Claude task files (`linear tasks export`, `internal/service/taskexport.go:192-216`, `internal/taskwriter/writer.go:33-56`) and into export markdown.
- **Fix:** issue tokens per developer from a dedicated Linear OAuth app with minimal scope, preferably agent mode so actions are attributable. Treat Linear content as untrusted in agent policy, and consider deny rules for uploading paths outside the repo. Optionally add a `--file` guard that refuses dotfiles and paths outside the CWD.

### LOW

**L1. The token can be written relative to the current directory.** `storage.go:146` and `config/manager.go:54`: if `os.UserHomeDir()` fails, the token and config go to `./.config/linear/` relative to the CWD, possibly inside a git repo. **Fix:** fail closed instead.

**L2. Token file permissions are not enforced on existing files.** The comment at `storage.go:62` claims WriteFile is atomic via rename, which is false. `os.WriteFile(…,0600)` does not tighten an existing file and follows symlinks. **Fix:** write to a temp file with 0600, `Chmod`, then `Rename`.

**L3. `linear init` changes agent instruction files without asking.** It appends a Linear section to `CLAUDE.md` and `AGENTS.md` in the CWD (`internal/cli/init.go:101-112,138-169`), and writes `.linear.yaml` with 0644 (`init.go:132`). The content is benign (team key from the API plus static text), but it modifies agent instructions unprompted. **Fix:** prompt first or add an opt-out flag. Document this for developers.

**L4. `skills install` writes into the current directory.** It writes 0644 files to `./.claude/skills/<name>/` relative to the CWD (`internal/cli/skills.go:80-170`). It never overwrites, and the paths come only from the embedded FS, so there is no traversal. I read all embedded skill content (`internal/skills/*/SKILL.md`, 1,462 lines): only `linear …` commands plus benign `cat`/`jq`/`gh issue view`/`git log` piping examples. There is no `allowed-tools` frontmatter, no network or exfiltration instructions, no hidden Unicode, and no auto-apply of bulk mutations (link-deps and triage "recommend" and present). **Fix:** none needed. Re-review skills on each upgrade (see M5).

**L5. `attachments download <url>` fetches any URL with any scheme Go supports.** `internal/cli/attachments.go:217-257` → `client.go:308-347`. Responses are capped at 100MB. The filename from Content-Disposition is sanitized for `/` and `\` (`client.go:353-369`, `issueexport.go:148-157`), so there is no path traversal. It acts as a generic fetcher (SSRF-style) for agents, but no token is sent to non-Linear hosts. **Fix:** restrict to https://uploads.linear.app.

**L6. Non-idempotent mutations are retried.** GraphQL mutations are retried up to 5 times on network EOF or reset errors and 429 (`pkg/linear/core/base_client.go:78-79,130-200`). This can create duplicate issues or comments. It is an integrity issue, not a security one.

**L7. Committed `.claude/settings.local.json` grants broad Claude Code permissions to anyone who opens this repo.** The allow list includes `rm:*`, `git push:*`, `go run:*`, `source:*`, `goreleaser release:*`, `gh release create:*` and `WebFetch(domain:github.com)`. It affects only people working in this repo, not users of the binary. **Fix:** remove the file from git and add it to `.gitignore`.

**L8. `make setup` / `install-deps` runs `npm install -g @anthropic-ai/claude-cli`** (`Makefile:21-28`). This is a global install of a package name that is not the official one (`@anthropic-ai/claude-code`). The scope is Anthropic-owned, so typosquatting risk is low. `make build` does not call it. **Fix:** remove the target.

### INFO
- **I1. No embedded secrets.** The OAuth client ID and secret are supplied by each user (prompted at `auth.go:228-277`, or from env). Personal API keys are sent without the Bearer prefix and OAuth tokens with it (`internal/token/sanitize.go:50-60`). Tokens are never printed. `auth login` prints the authorize URL (client_id and state only) and the token file path (`auth.go:143,166`). Errors include only a 100-char GraphQL query preview (`base_client.go:313-318`).
- **I2. `os/exec` is used only in `openBrowser`** (`internal/cli/auth.go:295-308`): `open`, `xdg-open` or `rundll32 url.dll,FileProtocolHandler` with the locally built authorize URL. No shell is invoked, and nothing else spawns processes.
- **I3. No suspicious init code, encoded blobs or background work.** There are no `init()` functions. No base64 or long encoded literals appear in non-test code. The only goroutines are the two OAuth callback servers. There are no hidden Cobra commands, no `unsafe`, `plugin` or `syscall` use, and no hidden Unicode in skills, docs or workflows.
- **I4. Destructive commands are all explicit, single-ID operations with no confirmation prompt:**
  - `labels delete <id>`: permanent issueLabelDelete, refused in agent mode (`internal/service/label.go:103-113`, `cli/labels.go:198-224`).
  - `attachments delete <id>` (`cli/attachments.go:264-280`).
  - `documents delete <ref>`: moves to trash (`cli/documents.go:376-395`).

  Other mutations: issueCreate/Update (including state changes; `--labels` replaces all labels at `cli/issues.go:653`), commentCreate, issueRelationCreate, projectCreate/Update, cycleCreate/Update, notificationUpdate, labelCreate/Update, attachmentCreate/Update, documentCreate/Update.

  There is no issue, project or comment delete or archive. The library has `BatchUpdateIssues` (`pkg/linear/issues/client.go:1147`) and `ArchiveCycle` (`pkg/linear/cycles/client.go:433`), but no CLI command exposes them, and there are no hidden bulk operations.
- **I5. Supply chain looks clean.** Only 5 third-party modules are linked into the binary: spf13/cobra v1.10.2, spf13/pflag v1.0.10, gopkg.in/yaml.v3 v3.0.1, golang.org/x/image v0.35.0 and dominikbraun/graph v0.23.0.
  - `x/image/draw` resizes images in the legacy attachment/MCP path (`client.go:801`).
  - `dominikbraun/graph` provides dependency-graph and cycle detection for `deps`/`search` (Apache-2.0; low maintenance since 2023, but small).
  - `getoutreach/goql` v1.13.5 is imported only by `pkg/linear/testutil/mock_server.go`, a test helper. It is not in the binary (checked with `go list -deps ./cmd/linear`). It pulls gobox, otel and blackfriday into the module graph only.
  - There are no `replace` directives, go.sum is complete, and `go mod verify` passes.
- **I6. Git history shows no suspicious authors or changes.** Authors are joa23 (26 commits) and ordinary upstream PR contributors (techjoec, Johannes Zillmann, Felix Lisczyk, and others). Recent changes to auth and networking code (token refresh fix #42, API-key standardization #41, attachment filename change 64c1136, export 593f094) are consistent with their descriptions.
- **I7. Other unsafe items are not part of the binary.** `scripts/vrataski-loop.sh` runs an unattended loop: it pulls issues assigned to "me", feeds them to Claude as tasks, and marks them Done. That is a prompt-injection-heavy pattern, so do not use it. `test-release.sh` creates and deletes data in the `TEST` team (`TEAM` is overridable). `.claude/worktrees/` is an untracked leftover from an earlier agent session.
- **I8. No egress during auth status checks beyond Linear.** Auth checks make no network calls besides Linear. There is no version or update check (`--version` is local, set by ldflags).

---

## Complete egress allowlist (binary)

| Host | Purpose | Code |
|---|---|---|
| `api.linear.app` (443) | GraphQL API `https://api.linear.app/graphql`; OAuth token exchange and refresh `https://api.linear.app/oauth/token` | `pkg/linear/core/base_client.go:19,263`; `internal/oauth/oauth.go:18,226,353` |
| `linear.app` (443) | OAuth authorize page, opened in the user's browser; the CLI itself makes no request here | `oauth.go:17,95,110`; `auth.go:146` |
| `uploads.linear.app` (443) | Downloads of private uploads (token attached) | `attachments/client.go:652-674`, `service/issueexport.go:139` |
| Signed upload URL returned by Linear's `fileUpload` mutation (in practice a Google Cloud Storage signed URL, e.g. `storage.googleapis.com`; confirm empirically) | PUT of uploaded file bytes; Linear-supplied headers; no Linear token sent | `attachments/client.go:991-1080` |
| `localhost`/all interfaces :37412 (inbound, during `auth login` only) | OAuth callback listener | `oauth.go:168-171,296-299` |
| *Any user- or agent-supplied URL* | Only via an explicit `linear attachments download <url>`; no token is sent unless the host is uploads.linear.app | `cli/attachments.go:217-257` |

Other URL literals are help or doc text only and never fetched: github.com/joa23/linear-cli/issues, github.com/org/repo/pull/42, figma.com, and linear.app/acme/document/…. No telemetry, analytics, update check, GitHub API call or DNS call exists outside the HTTP client.

## Environment variables read

Binary (runtime):
- `LINEAR_API_KEY`: API token. Used only if no token file exists (`internal/token/storage.go:170`; `pkg/linear/client.go:132-139`).
- `LINEAR_CLIENT_ID` and `LINEAR_CLIENT_SECRET`: OAuth app credentials (`internal/config/manager.go:182-186`; `internal/cli/auth.go:101,104`; `pkg/linear/client.go:113,116`).
- `LOG_LEVEL`, `LOG_FILE` and `POLLING_INTERVAL`: loaded into config (`config/manager.go:167-178`) but never used anywhere else (legacy). In particular, `LOG_FILE` writes nothing.
- Implicit: `HOME`/`USERPROFILE` via `os.UserHomeDir`, and `TMPDIR` via `os.TempDir` (download default dir). Browser launchers (`xdg-open` and others) read their own env. `HTTP(S)_PROXY`/`NO_PROXY` are **not** honored by the CLI's clients (see M4).

Non-binary:
- `test-release.sh`: `TEAM`, `BINARY`.
- `scripts/vrataski-loop.sh`: `CLAUDE_TASKS_DIR`, `CLAUDE_SESSION_ID`, `HOME`.
- Makefile: `VERSION`.
- Workflows: `secrets.GITHUB_TOKEN`, `secrets.TESSL_API_TOKEN`.
- Tests also set `LINEAR_API_TOKEN` (a legacy name the binary no longer reads).

## Fixes required before adoption
1. Build and distribute from a pinned Relynk commit with your own CI and a current Go version. Do not use joa23's tap or releases (H1, M1).
2. Bind the OAuth callback to 127.0.0.1 and add PKCE (M2).
3. Require https before attaching the token to downloads (M3).
4. Delete the tessl workflows in the fork (M5). Remove `.claude/settings.local.json` and the `install-deps` target (L7, L8).
5. Enforce egress by firewall or DNS allowlist, or add `ProxyFromEnvironment` (M4).
6. Agent policy: treat Linear content as untrusted, and use a minimal-scope agent-mode OAuth app per developer (M6).

---

# Round 2: branch claude/loving-ritchie-5qkzuq (HEAD c361242)

**Verdict: the fixes are correct. No Critical, High or Medium issues remain in the code. What is left is Low/Info hardening, plus two settings that must be made in the GitHub repo, outside the code.**

Checks run on HEAD:
- `go build ./...` and `go vet ./...`: clean.
- `go test ./...`: all pass except the root-only `TestConfigManagerErrorHandling`, the same sandbox artifact as in round 1.
- I rebuilt the binary and confirmed two behaviours. With `HTTPS_PROXY` set, API calls now go through the proxy: the request reached it, and the sandbox proxy refused the host. `attachments download http://uploads.linear.app/a` is refused.
- I verified with `git ls-remote` that every action SHA in the workflows is the commit its tag comment names: checkout v7.0.1 `3d3c42e5`, setup-go v7.0.0 `b7ad1dad`, goreleaser-action v7.3.0 `a4f02ab5`, attest-build-provenance v4.2.2 `4d101475`.
- go1.27.4 exists (`GOTOOLCHAIN=go1.27.4 go version` downloaded and ran it).
- The setup-go v7.0.0 source (`src/installer.ts:655-657`) and its README confirm that `go-version-file` uses the go.mod `toolchain` line, so CI and releases really build with go1.27.4 even with `GOTOOLCHAIN=local`.

## (a) Security fix commit 3c8f788

| Round-1 item | Status | Evidence |
|---|---|---|
| M2 callback bind | Fixed | `internal/oauth/oauth.go:204,337` bind to `127.0.0.1:`+port; `defer server.Close()` at `:218,351` |
| M2 PKCE | Fixed | S256 verifier and challenge at `oauth.go:40-60`, created per handler at `:93,106`; `code_verifier` sent at `:262`; covered by the `TestPKCEParameters` test |
| M3 https-only token attach | Fixed | `pkg/linear/attachments/client.go:657-658`, `internal/service/issueexport.go:144-145` |
| L5 download host restriction | Fixed (CLI only) | `internal/cli/attachments.go:243-245,295-303`. The library `DownloadToFile`/`DownloadBytes` still accept any URL, which is acceptable because they only attach the token to https://uploads.linear.app |
| M4 proxy | Fixed | `pkg/linear/core/http_client.go:12` (verified at runtime) |
| L1 CWD fallback | Fixed, with a weaker fallback (R2-L2) | `internal/token/storage.go:144-149`, `internal/config/manager.go:53-57` |
| M5 tessl workflows | Removed | Only `ci.yml` and `release.yml` remain |
| L7 settings.local.json | Removed and git-ignored | |
| H1 / M1 distribution and toolchain | Addressed in the pipeline (part c) | |
| L2 non-atomic token write, L3 `init` edits CLAUDE.md/AGENTS.md, M6 arbitrary-file upload | **Still open** (not in scope of the fix) | Unchanged |

Remaining items from (a):
- **R2-L1 (Low). PKCE is silently skipped if the verifier can't be generated.** `oauth.go:40-46` returns `""` when `rand.Read` fails, and `addPKCE` then just leaves PKCE out. `GenerateState` (`oauth.go:439-443`) still ignores the `rand.Read` error.
  - **Fix:** return an error and abort the login in both places. In practice `rand.Read` doesn't fail on Go 1.24 or later.
- **R2-L2 (Low). The new temp-dir fallback can be hijacked on shared machines.** The fallback path is `os.TempDir()/linear-<uid>/token` (`storage.go:149`, `manager.go:57`). On a multi-user host another user can create that directory first. `MkdirAll(…,0700)` (`storage.go:56,187`) accepts an existing directory without checking who owns it, so the attacker could swap in their own token file or plant a symlink.
  - On Windows `os.Getuid()` is -1.
  - It only applies when neither HOME nor XDG_CONFIG_HOME can be resolved.
  - **Fix:** fail closed with "cannot determine home directory". Otherwise, `Lstat` the directory and check it is owned by the current user with mode 0700.
- **R2-L3 (Low, functional).** The redirect URI is still `http://localhost:<port>` (`internal/cli/auth.go:133,256`) while the server listens only on IPv4 127.0.0.1. Browsers that resolve localhost to ::1 first normally fall back to 127.0.0.1, so this should work. RFC 8252 recommends the literal `127.0.0.1`, but changing it requires updating each OAuth app's registered redirect. The error message at `auth.go:152` mentions a `--port` flag that doesn't exist.
  - **Fix:** document this, or move to `http://127.0.0.1:<port>`.

## (b) Initiatives feature (a669a51..HEAD)

- **New network destinations: none.** Every query goes through `ExecuteRequest` to api.linear.app. The new `https://linear.app/...` and `github.com/Relynk` strings appear only in tests, docs or help text and are only parsed (`internal/service/initiative.go:435-449`), never fetched. No new `os/exec` calls, environment variable reads or file writes.
- **GraphQL injection: none.** Every query and mutation in `pkg/linear/initiatives/client.go` (`:170,251,283,331,403,441,470,499,531,616,690`) and in `pkg/linear/projects/client.go` (`ListByInitiative`) is a constant string with typed `$variables`. Filters are built as Go maps (`buildInitiativeProjectFilter`). No `Sprintf` or string concatenation goes into query text.
- **File and stdin input:** `--content-file`/`--body-file`/`-` go through the shared `readTextFlags` (`internal/cli/documents.go:409-431`). It does `os.ReadFile` on any path, which is the same accepted-by-design behaviour as round-1 M6 and introduces nothing new. The two flags are mutually exclusive.
- **Destructive behaviour:**
  - The new commands are `initiatives archive|unarchive|delete <ref>` (`internal/cli/initiatives.go:381-409`) and `remove-project` (`:434-455`). Each acts on one explicit ref with no bulk path.
  - `initiativeDelete` moves the initiative to trash, and the output says linked projects are kept (`internal/service/initiative.go:278-288`).
  - Name refs must match exactly one initiative; ambiguous names are refused (`initiative.go:503-520`).
  - Edge case: a *name* ending in `-<12 hex>` is treated as a slug first (`initiative.go:451-453`, `:463-470`), so it could resolve to a different initiative whose slug matches. This is negligible.
  - There is no confirmation prompt, consistent with the rest of the CLI.
- **R2-L4 (Low). The skill text gives agents `delete` alongside routine commands with no caution.** `internal/skills/linear/SKILL.md` (new lines) lists `linear initiatives archive|unarchive|delete <ref>`. The CLI help says "Prefer 'archive'", but the skill text doesn't.
  - **Fix:** add a line to the skill: "Prefer `archive`. Only run `delete` or `remove-project` when the user explicitly asks for it."
- **Info.** `test-release.sh` (new section 10c) creates, links, archives and deletes a test initiative in the live workspace. It temporarily links the first project of `$TEAM` (default TEST) and unlinks it again. Run it only against a test team.

## (c) Release pipeline (01404a8, a669a51)

These look good:
- Every action is SHA-pinned (verified).
- GoReleaser is pinned to v2.19.0.
- Top-level `permissions: {}`, with write permissions only in the release job (`contents`, `id-token`, `attestations`).
- `persist-credentials: false`.
- No build cache in the release job.
- Tests run in a separate read-only job.
- `GOTOOLCHAIN=local` and `-mod=readonly`.
- `go mod verify` runs, and `go mod tidy -diff` is a check only.
- The `go mod tidy` release hook was removed.
- Builds use `-trimpath` with `mod_timestamp` and fixed archive metadata for reproducibility.
- Releases are published to Relynk as drafts and only made public after build provenance is attested.
- `workflow_dispatch` was removed.
- CI runs on `pull_request` with read-only permissions and no secrets, and includes govulncheck pinned to v1.9.0.
- The Makefile's global `npm install` was removed.
- `go.mod` has `toolchain go1.27.4` (`go.mod:5`).

Remaining items:
- **R2-M1 (Medium, repo settings, not code). Anyone with write access can still cut a release.** `release.yml` uses `environment: release`, but that only protects anything once the environment is configured, as the workflow's own comment says. Until then, anyone who can push a `v*.*.*` tag publishes a release that will be installed everywhere.
  - **Fix:** in Relynk/linear-cli settings, create the `release` environment with required reviewers. Add a tag ruleset restricting who can create `v*` tags. Enable branch protection on main and immutable releases.
- **R2-L5 (Low). The release job doesn't check which Go version it used.** "Show toolchain" (`release.yml`) only prints the version.
  - **Fix:** fail the job if `go version` doesn't report the go.mod toolchain (for example `go version | grep -q "go$(awk '/^toolchain/{sub("go","",$2);print $2}' go.mod) "`). This guards against a future setup-go change.
- **R2-L6 (Low). Toolchain freshness needs an owner.** go1.27.4 is current today, but nothing bumps the toolchain or the action SHAs automatically.
  - **Fix:** enable Dependabot or Renovate for `gomod` (including the toolchain line) and `github-actions`, and keep govulncheck in CI as the tripwire.
- **Info.** `Formula/linear-cli.rb:18-39` now points to Relynk URLs but still holds the upstream v1.11.0 checksums (the file flags this). Until the first Relynk release, `brew install` fails safely (404 or checksum mismatch). Update the version and sha256 values after the first Relynk release.
- **Info.** The module path and ldflags still use `github.com/joa23/linear-cli` (`go.mod:1`, `.goreleaser.yml` ldflags, Makefile `MODULE`). This is cosmetic and has no security impact.

## Round 2 summary

| Sev | ID | Location | Fix |
|---|---|---|---|
| Medium | R2-M1 | GitHub repo settings / `release.yml` `environment: release` | Configure the `release` environment with reviewers, protect `v*` tags, enable immutable releases |
| Low | R2-L1 | `internal/oauth/oauth.go:40-46,439-443` | Abort the login if `rand.Read` fails, instead of silently dropping PKCE or state |
| Low | R2-L2 | `internal/token/storage.go:149`, `internal/config/manager.go:57` | Fail closed, or check owner and mode of the `/tmp/linear-<uid>` directory |
| Low | R2-L3 | `internal/cli/auth.go:133,152,256` | Use 127.0.0.1 in the redirect, or document it; fix the `--port` hint |
| Low | R2-L4 | `internal/skills/linear/SKILL.md` (initiatives block) | Add "prefer archive; delete/remove-project only on explicit user request" |
| Low | R2-L5 | `.github/workflows/release.yml` "Show toolchain" | Assert that `go version` equals the go.mod toolchain |
| Low | R2-L6 | repo config | Dependabot or Renovate for gomod, the toolchain line and actions |
| Open from R1 | L2, L3, M6 | `storage.go:56-63,184-195`; `cli/init.go:101-169`; `--file/--attach/--content-file/--body-file` | Atomic 0600 writes; ask before editing CLAUDE.md/AGENTS.md; agent policy and path guard for uploads |
