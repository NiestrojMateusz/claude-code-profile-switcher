# Plan: ccp — Claude Code Profile Manager

> Source PRD: PRD.md

## Session log

### 2026-04-15 — Phase 1 + Phase 2 (TDD, outside-in)

**Interface design**: chose Design A (minimal — `ConfigStore{Load, Save}`, `SymlinkInspector{Inspect([]string)}`). Adapted to pure functions (no interfaces) after review — `config.Load(root)`, `symlink.Inspect(claudeDir, target)`.

**Implemented packages** (21 tests, all GREEN):
- `internal/config` — `Load`, `Save`, `ActiveProfile`, `SetActive`
- `internal/symlink` — `Inspect`, `InspectAll`, `Kind` (Absent/Real/Symlink)
- `internal/backup` — `Backup` (rename + idempotent no-op)
- `internal/profile` — `Create` (subdirs + config only, **no symlinks** — symlinks are caller's responsibility)
- `tui/selector` — Bubbletea `Model` with cursor, space-toggle, filter; `Run()` for production
- `cmd/status` — `runStatus(profilesRoot, claudeDir, w)`
- `cmd/init` — `runInit(profilesRoot, claudeDir, picker, w)` with injected `SkillPicker`

**Key decisions**:
- `profile.Create` does NOT create symlinks — only `runInit` and future `ccp switch` do
- `SkillPicker` injected as `func([]string)([]string,error)` — tests stub it, prod uses TUI
- `discoverSkills` is a stub returning nil — skill discovery wired in Phase 5
- Backup lands in same dir as original: `~/.claude/skills_backup_<unix-ts>`

**Next session starts at**: Phase 4 — `ccp profile` subcommands

### 2026-04-15 — Phase 3 (TDD, outside-in)

**Implemented packages** (4 new tests, all GREEN):
- `internal/symlink` — added `Switch(claudeDir, profileRoot)`: validates all subdirs first (fail-fast), then remove+create per symlink
- `internal/process` — `Scan()` uses `ps -eo pid,comm`; internal `scanEntries` is injectable for tests
- `cmd/switch` — `runSwitch(profilesRoot, claudeDir, name, scanProcesses, confirm, w)` with injected process scanner + confirm callback

**Key decisions**:
- Two-pass in `Switch`: validate all subdirs → then mutate. Prevents partial state.
- `confirm func([]int)(bool,error)` injected alongside scanner — same pattern as `SkillPicker` in `cmd/init`
- `slices.Contains` used for profile lookup (Go 1.25)

---

## Architectural decisions

- **Binary name**: `ccp`
- **Module path**: `github.com/matis/ccp`
- **Config location**: `~/.claude-profiles/config.json`
- **Profile root**: `~/.claude-profiles/<name>/` (each with `skills/`, `commands/`, `agents/` subdirs)
- **Managed symlinks**: `~/.claude/skills`, `~/.claude/commands`, `~/.claude/agents` — dir-level symlinks only
- **Base profile name**: `base` — special, cannot be deleted while dependents exist
- **Backup naming**: `<original-dir>_backup_<unix-timestamp>`
- **Agent skills source**: `~/.agents/skills/` — read-only, never written by `ccp`
- **Key data models**: `Config{ActiveProfile, Profiles []string, KnownSkills []string}`, `Profile{Name, RootPath}`
- **CLI framework**: Cobra; **TUI**: Bubbletea + Bubbles + Lipgloss
- **Testability**: all modules accept root path as parameter (default `~/.claude-profiles`) so tests use `t.TempDir()`

---

## Phase 1: Skeleton + `ccp status`

**User stories**: 11, 16

### What to build

Wire up the Cobra root command and a `status` subcommand. The `config` module loads (or initialises an empty) `config.json`. The `symlink` module inspects `~/.claude/skills`, `~/.claude/commands`, and `~/.claude/agents` to determine whether each is a real directory, a symlink (and to what target), or absent. `ccp status` prints active profile name and the state of each of the three managed paths. When no profile is active (real directories intact), it says so explicitly.

This phase delivers a compilable, installable binary that gives useful output from day one and validates the full stack: Cobra command → config module → symlink module → terminal output.

### Acceptance criteria

- [x] `go build ./...` produces a `ccp` binary
- [x] `ccp status` prints active profile (or "no active profile") and symlink targets for all three managed paths
- [x] `ccp status` works correctly when `~/.claude-profiles/config.json` does not yet exist (treats as uninitialized)
- [x] `symlink.Inspect` tested for real dir, symlink, and absent cases via `t.TempDir()` fixture
- [x] `config.Load` / `config.Save` / `config.ActiveProfile` / `config.SetActive` round-trip correctly in unit tests

---

## Phase 2: `ccp init` with TUI wizard

**User stories**: 2, 3, 4, 15

### What to build

Build the `backup` module and the `tui/selector` reusable component, then wire them into `ccp init`.

`backup` renames `~/.claude/skills`, `~/.claude/commands`, and `~/.claude/agents` to `<dir>_backup_<timestamp>` before any symlink is created. It is idempotent: if a backup already exists it skips.

`tui/selector` is a Bubbletea multi-select list with search/filter. It is generic enough to be reused by the profile editor and monitor prompt in later phases.

`ccp init` runs a wizard that:
1. Detects skills in the backed-up `~/.claude/skills_backup_*` and in `~/.agents/skills/`
2. Presents them in `tui/selector` so the user picks which go into the base profile
3. Creates `~/.claude-profiles/base/skills|commands|agents/`, moves/symlinks selected skills in
4. Replaces `~/.claude/skills|commands|agents` with symlinks to `base/`
5. Writes `config.json` with `base` as active profile

If a base profile already exists, `ccp init` prints a message and exits without modifying anything.

### Acceptance criteria

- [x] `ccp init` backs up original directories with correct timestamp suffix before touching symlinks
- [x] Running `ccp init` a second time exits gracefully without modifying existing config or directories
- [x] `tui/selector` renders a filterable multi-select list and returns the selected items (model logic unit-tested; `Run()` wired for production)
- [ ] Selected skills from `~/.claude/skills_backup_*` appear as real dirs in `base/skills/` — `discoverSkills` is a stub; wired in Phase 5
- [ ] Selected skills from `~/.agents/skills/` appear as symlinks in `base/skills/` — wired in Phase 5
- [x] `~/.claude/skills`, `~/.claude/commands`, `~/.claude/agents` are symlinks pointing to `base/` after init
- [x] `config.json` records `base` as active profile
- [x] `backup` module is idempotent (tested in unit tests with temp dir)

---

## Phase 3: `ccp switch`

**User stories**: 5, 6, 13

### What to build

`ccp switch <profile>` atomically replaces the three managed symlinks to point at the named profile. "Atomic" here means: remove old symlink, create new one — done per-link in sequence (true atomic swap is not possible for symlinks without a tmp+rename trick, which is not needed given single-user scope).

The `process` module scans for running `claude` processes (via `ps` on macOS, `/proc` on Linux). If any are found, `ccp switch` prints a warning listing their PIDs and asks for confirmation before proceeding.

The base profile is switchable like any other profile.

### Acceptance criteria

- [x] `ccp switch base` updates all three symlinks to point at `~/.claude-profiles/base/`
- [x] `ccp switch <name>` fails with a clear error if the named profile does not exist
- [x] Running `ccp status` after a switch reflects the new active profile
- [x] `process` module detects a mock process list and returns correct PIDs (unit tested)
- [x] A warning with PIDs is printed when `claude` processes are running; user is prompted to confirm
- [x] `symlink.Switch` is tested end-to-end in a temp dir: correct target after switch, no partial state on failure

---

## Phase 4: `ccp profile` subcommands

**User stories**: 7, 10, 14

### What to build

Three subcommands under `ccp profile`:

- `profile list`: prints all known profiles, marking the active one
- `profile create <name>`: creates `~/.claude-profiles/<name>/skills|commands|agents/`, then symlinks every entry in `base/skills/` into `<name>/skills/` (inherited baseline)
- `profile delete <name>`: removes the profile directory; refuses if the profile is currently active or if it is `base` and other profiles exist

### Acceptance criteria

- [ ] `ccp profile list` shows all profiles; active profile is visually distinguished
- [ ] `ccp profile create <name>` produces correct directory structure with symlinks to all base skills
- [ ] Newly created profile's skill symlinks resolve correctly to base skill directories
- [ ] `ccp profile delete <name>` removes the profile directory and its entry from `config.json`
- [ ] `profile delete` refuses to delete the active profile and prints a clear error
- [ ] `profile delete base` is refused when other profiles still exist
- [ ] `profile` module CRUD is tested with `t.TempDir()`

---

## Phase 5: `ccp skill` subcommands

**User stories**: 9, 17

### What to build

Two subcommands under `ccp skill`:

- `skill add <path-or-git-url> [--profile <name>]`: adds a skill to a profile. For a local path, copies or symlinks the directory into the profile's `skills/`. For a git URL, clones into a temp dir then delegates to the local-path path. Defaults to the active profile if `--profile` is omitted.
- `skill list [--profile <name>]`: lists skills in a profile, distinguishing base-inherited symlinks from profile-owned real directories.

### Acceptance criteria

- [ ] `ccp skill add ./my-skill` copies the skill into the active profile's `skills/`
- [ ] `ccp skill add https://github.com/...` clones and installs the skill
- [ ] `ccp skill list` shows all skills for the active profile with inherited/owned labels
- [ ] `ccp skill list --profile <name>` works for non-active profiles
- [ ] `skill.AddLocal` is unit tested with a temp dir fixture
- [ ] `skill.AddGit` integration test is present but skipped in CI (build tag or env var guard)

---

## Phase 6: TUI profile editor

**User story**: 8

### What to build

`ccp profile edit <name>` opens a Bubbletea editor that shows all base skills as a checkbox list (checked = inherited, unchecked = removed from this profile). On save, checked skills are present as symlinks in `<name>/skills/`; unchecked ones are removed. Uses `tui/selector` from Phase 2.

### Acceptance criteria

- [ ] `ccp profile edit <name>` renders a checkbox list of all base skills
- [ ] Unchecking a skill and saving removes its symlink from the profile's `skills/`
- [ ] Re-checking a skill and saving restores the symlink
- [ ] Changes are not applied until the user confirms (no partial writes on cancel)
- [ ] The editor reuses `tui/selector` without duplicating multi-select logic

---

## Phase 7: Monitor + new skill detection

**User story**: 12

### What to build

The `monitor` module compares the current contents of `~/.agents/skills/` against `config.KnownSkills()` and returns a list of previously-unseen skills.

On every `ccp` invocation (root command `PersistentPreRun`), if new skills are detected, a `tui/monitor` prompt appears: a list of new skills each with options — "add to profile X", "create new profile", "skip", "never ask again for this skill". Selections are applied immediately and `config.KnownSkills` is updated so the skill is not surfaced again.

### Acceptance criteria

- [ ] `monitor` returns only skills not present in `config.KnownSkills()` (unit tested)
- [ ] New skills in `~/.agents/skills/` trigger the TUI prompt on the next `ccp` invocation
- [ ] "Never ask again" marks the skill in `config.KnownSkills` so it is not surfaced in future runs
- [ ] "Add to profile X" installs the skill symlink into the named profile's `skills/`
- [ ] "Skip" dismisses without modifying config; skill reappears on next invocation
- [ ] No prompt appears when no new skills are detected
