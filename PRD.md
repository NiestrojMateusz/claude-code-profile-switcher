# PRD: ccp — Claude Code Profile Manager

## Problem Statement

When testing multiple Claude Code harnesses, skills, commands, and agents accumulate in `~/.claude/`. Because Claude Code injects all installed skills and commands into the system context at the start of every session, having many harness-specific tools installed simultaneously wastes significant context window tokens before any conversation begins. There is no built-in way to switch between focused sets of tools depending on the task at hand.

## Solution

A Go CLI tool (`ccp`) that manages named profiles — each profile is a directory containing its own `skills/`, `commands/`, and `agents/` folders. Activating a profile atomically replaces `~/.claude/skills`, `~/.claude/commands`, and `~/.claude/agents` with symlinks pointing to the profile's directories. A base profile owns the canonical skill directories; other profiles symlink from base (with per-profile overrides possible). On every invocation, `ccp` checks `~/.agents/skills` for newly available agent skills and prompts the user to assign them to profiles.

## User Stories

1. As a Claude Code power user, I want to switch between named profiles, so that only the skills relevant to my current task consume context window tokens.
2. As a user, I want `ccp init` to guide me through an interactive TUI setup, so that I can select which existing skills form my base profile without manual file operations.
3. As a user, I want `ccp init` to let me choose which skills from `~/.agents/skills` are included in the base profile, so that agent-specific tools don't pollute all sessions.
4. As a user, I want my original `~/.claude/skills`, `~/.claude/commands`, and `~/.claude/agents` directories renamed to `skills_backup_<timestamp>` before any symlinks are created, so that I can recover the original state if needed.
5. As a user, I want `ccp switch <profile>` to atomically replace the three symlinks, so that a profile switch is instantaneous and never leaves a partial state.
6. As a user, I want `ccp switch` to warn me if any `claude` processes are currently running, so that I can decide whether to proceed knowing active sessions may be affected.
7. As a user, I want `ccp profile create <name>` to automatically symlink all base skills into the new profile's skills directory, so that new profiles inherit a complete baseline without manual setup.
8. As a user, I want an interactive TUI editor (`ccp profile edit <name>`) that lets me deselect individual base skills for a specific profile, so that I can have a lean, focused skill set per profile.
9. As a user, I want `ccp skill add <path-or-git-url> [--profile <name>]` to add a skill from a local path or git repository to a specific profile, so that I can extend profiles without manual filesystem operations.
10. As a user, I want `ccp profile list` to show all profiles and which one is currently active, so that I know my current context at a glance.
11. As a user, I want `ccp status` to show the active profile and the symlink targets for all three directories, so that I can verify the current state.
12. As a user, I want `ccp` (with no arguments) to detect new skills in `~/.agents/skills` on every invocation and ask me whether to add them to an existing or new profile, so that I don't miss newly installed agent tools.
13. As a user, I want the base profile to be activatable like any other profile, so that I can run sessions with only the canonical base skill set.
14. As a user, I want `ccp profile delete <name>` to remove a profile and its symlinks, so that I can clean up unused profiles.
15. As a user, I want `ccp init` to be skipped gracefully if a base profile already exists, so that re-running the command doesn't destroy existing configuration.
16. As a user, I want the tool to work when no profile is active (original real directories intact), so that `ccp` can be installed without immediately forcing a migration.
17. As a user, I want `ccp skill list [--profile <name>]` to show which skills are active in a profile (distinguishing base-inherited vs. profile-specific), so that I can audit a profile's composition.

## Implementation Decisions

### Modules

**`config`**
- Manages `~/.claude-profiles/config.json` — stores active profile name, list of known profiles, and timestamp of last `~/.agents/skills` scan.
- Interface: `Load()`, `Save()`, `ActiveProfile()`, `SetActive(name)`, `KnownSkills() []string`.
- This is the source of truth for state; all other modules read from it.

**`profile`**
- CRUD for profiles in `~/.claude-profiles/<name>/`.
- On `Create(name)`: creates `skills/`, `commands/`, `agents/` subdirs; symlinks all base skills into `skills/`.
- On `Delete(name)`: removes the profile directory.
- Knows the distinction between base-inherited symlinks and profile-owned real dirs within a profile.

**`symlink`**
- Handles all symlink operations on `~/.claude/skills`, `~/.claude/commands`, `~/.claude/agents`.
- Atomic switch: `os.Remove(link)` then `os.Symlink(target, link)`.
- `IsSymlink(path)`, `CurrentTarget(path)`, `Switch(profile)`.
- Never touches anything outside these three paths.

**`backup`**
- Before the first symlink operation, renames original real directories to `<dir>_backup_<timestamp>`.
- Idempotent: if backup already exists, skips.

**`process`**
- Detects running `claude` processes by scanning `/proc` (Linux) or using `ps` output (macOS).
- Returns list of PIDs; caller decides whether to warn or abort.

**`skill`**
- `AddLocal(profileName, srcPath)`: copies or symlinks a local skill dir into profile.
- `AddGit(profileName, repoURL)`: clones repo into a temp dir, then calls `AddLocal`.
- `Remove(profileName, skillName)`: removes skill symlink from profile's skills dir.

**`monitor`**
- Compares current contents of `~/.agents/skills` against `config.KnownSkills()`.
- Returns list of new, previously-unseen skills.
- Called at startup; presents TUI prompt if new skills found.

**`tui/init`**
- First-run wizard: multi-select list of skills from `~/.claude/skills_backup_*` and `~/.agents/skills`.
- Confirms selections, then triggers `backup` + base profile creation.

**`tui/editor`**
- Profile editor: shows all base skills with checkboxes; unchecked = symlink removed from this profile.
- Writes changes back via `profile` module.

**`tui/selector`**
- Reusable Bubbletea component: multi-select list with search/filter.
- Used by both `tui/init` and `tui/editor`.

**`tui/monitor`**
- Prompt shown when new agent skills are detected: list with options "add to profile X / create new profile / skip / never ask again for this skill".

### Directory Layout

```
~/.claude-profiles/
  config.json
  base/
    skills/       ← real dirs (moved from ~/.claude/skills_backup_*)
                    + symlinks to ~/.agents/skills/* (selected at init)
    commands/
    agents/
  <profile>/
    skills/       ← symlinks to base/skills/* (minus deselected)
                    + profile-owned real dirs (added via ccp skill add)
    commands/
    agents/
```

`~/.claude/skills` → `~/.claude-profiles/<active>/skills/`
`~/.claude/commands` → `~/.claude-profiles/<active>/commands/`
`~/.claude/agents` → `~/.claude-profiles/<active>/agents/`

### Tech Stack

- Language: Go
- CLI framework: Cobra (`github.com/spf13/cobra`)
- TUI: Bubbletea + Bubbles + Lipgloss (`github.com/charmbracelet/*`)
- Install: `go install github.com/matis/ccp@latest`
- Binary name: `ccp`

### Symlink Strategy

- `~/.claude/skills`, `~/.claude/commands`, `~/.claude/agents` become dir-level symlinks (not contents-as-symlinks).
- Within `base/skills/`: real directories for skills that originated in `~/.claude/skills`; symlinks for skills from `~/.agents/skills`.
- Within `<profile>/skills/`: symlinks to `base/skills/<name>` for inherited skills; real dirs for profile-added skills.

## Testing Decisions

**What makes a good test:** test external behavior through the public interface of each module. Do not assert on internal file structure unless it is the module's explicit contract. Tests should be runnable without a real `~/.claude` directory by accepting configurable root paths.

**Modules to test:**

- `symlink` — core correctness: given a temp dir structure, verify Switch replaces symlinks atomically, verify IsSymlink and CurrentTarget return correct values, verify behavior when target does not exist.
- `backup` — verify rename happens with correct timestamp format, verify idempotency (second call does nothing).
- `config` — verify JSON round-trip, verify SetActive/ActiveProfile consistency.
- `profile` — verify Create produces correct directory structure, verify that inherited base symlinks resolve correctly, verify Delete cleans up completely.
- `monitor` — verify new skill detection against a known-skills list.
- `skill` — verify AddLocal copies correctly, verify AddGit (integration, can be skipped in CI).

**Approach:** all modules accept root path as a parameter (default `~/.claude-profiles`) so tests can use `t.TempDir()`.

## Out of Scope

- Per-profile `settings.json` and `CLAUDE.md` (planned for roadmap).
- Per-profile plugin management (Claude Code handles plugin enable/disable natively).
- Background daemon / filesystem watcher for `~/.agents/skills` (check-on-invocation is sufficient for v1).
- GUI or web interface.
- Multi-user / shared profile repositories.
- Profile versioning or rollback beyond the initial backup.

## Further Notes

- The `~/.agents/` directory is an existing convention for a separate agent harness; `ccp` treats it as read-only and never writes into it.
- The base profile is special: it cannot be deleted while other profiles exist that depend on its skills.
- When no profile is active, `~/.claude/skills` remains a real directory; `ccp status` should make this state explicit.
- `ccp switch` should be safe to call from a shell alias or Claude Code hook once the tool is stable.
