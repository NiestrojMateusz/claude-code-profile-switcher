# PRD: Unified 3-Step Picker for Skills, Commands, and Agents

## Problem Statement

When creating a new profile or running init, users can pick which skills to include via an interactive TUI picker. However, commands and agents — which are equally important parts of a profile — have no picker UI at all. Users cannot select or manage which commands and agents belong to a profile through the same interactive flow. Additionally, when editing an existing profile, users can only modify skills, not commands or agents.

## Solution

Extend the existing skills picker into a unified, sequential 3-step picker that covers skills, commands, and agents. Each step shows a clearly labeled header (`"Step X of 3: Select <Category>"`), allowing independent selection per category. This unified flow runs during init, profile creation, and profile editing.

## User Stories

1. As a developer running init, I want to pick skills in step 1 of 3, so that I know there are more steps to complete.
2. As a developer running init, I want to pick commands in step 2 of 3, so that I can include the right custom commands in my base profile.
3. As a developer running init, I want to pick agents in step 3 of 3, so that I can include the right agents in my base profile.
4. As a developer creating a new profile, I want to go through the same 3-step picker as init, so that I can configure skills, commands, and agents in one flow.
5. As a developer editing an existing profile, I want to edit skills, commands, and agents in the same 3-step picker, so that I don't have to re-create the profile to change commands or agents.
6. As a developer, I want to see a step indicator (`Step 2 of 3: Select Commands`) at the top of each picker, so that I know where I am in the selection flow.
7. As a developer, I want each step to be independently selectable, so that I can select 3 skills and 0 commands and 2 agents without issue.
8. As a developer, I want commands discovered from my backup directories, so that my existing command files are available for selection.
9. As a developer, I want agents discovered from my backup directories, so that my existing agent files are available for selection.
10. As a developer, I want previously selected commands to be pre-checked when editing a profile, so that I can see what is already included and make incremental changes.
11. As a developer, I want previously selected agents to be pre-checked when editing a profile, so that I can see what is already included and make incremental changes.
12. As a developer, I want to navigate each step with the same keyboard shortcuts as the skills picker (j/k, space, enter, filter by typing), so that the UX is consistent.
13. As a developer, I want to press enter on an empty step to proceed without selecting anything, so that I can skip commands or agents if I have none.

## Implementation Decisions

### Modules to Build or Modify

**Selector TUI (`tui/selector`)**
- Add `StepInfo{Current, Total int}` and `Title string` to the selector model
- Render `"Step X of Y: Select <Title>"` as a header above the list
- No changes to selection mechanics — space, enter, j/k, filter remain identical
- Interface stays generic: caller passes title and step info, selector has no knowledge of categories

**Discovery (`internal/skill` or co-located discovery package)**
- Add `discoverCommands(backupDirs []string) []string` mirroring `discoverSkills()`
- Add `discoverAgents(backupDirs []string) []string` mirroring `discoverSkills()`
- Both scan the same backup directories but look in `commands/` and `agents/` subdirs respectively
- Shared internal helper for scanning a subdir across multiple backup roots

**Init flow (`cmd/init.go`)**
- Replace single `tuiSkillPicker` call with sequential 3-step orchestration
- Order: skills (step 1) → commands (step 2) → agents (step 3)
- After all 3 steps, symlink selected skills, commands, and agents into the base profile

**Profile create flow (`cmd/profile.go`)**
- Replace single picker call with same 3-step orchestration as init
- Pass discovered items and empty pre-selection to each step

**Profile edit flow (`cmd/profile.go`)**
- Extend `runProfileEdit()` from skills-only to 3-step flow
- Pre-populate each step with currently active selections for that profile
- After completion, re-apply symlinks for all three categories

**Test Fixtures**
- Seed example `.md` files in `commands/` and `agents/` subdirs of temp/backup directories
- Used for manual testing and to verify discovery works end-to-end
- At least 2-3 example files per category with realistic names

### Architectural Decisions

- The 3-step orchestration is not a separate module — it lives inline in each command handler (`runInit`, `runProfileCreate`, `runProfileEdit`). The three calls are sequential and the shared pattern is extracted only if duplication becomes significant.
- Discovery functions follow the existing `discoverSkills()` pattern exactly. No abstraction layer added until all three discovery functions exist and duplication is visible.
- `SkillEditorPicker` interface (used in profile.go for swappable pickers) is extended or replaced to support 3-category selection, maintaining testability.

### Symlink Behavior

- After all 3 steps complete, symlinks are applied for all three categories in one pass using the existing `symlink.Switch()` / `symlink.Target` mechanism, which already has `Skills`, `Commands`, and `Agents` fields.

## Testing Decisions

**What makes a good test here:**
- Test external behavior only: given a set of discovered files and user selections, assert that the correct items are symlinked into the profile
- Do not test TUI rendering internals — test that the picker returns the right selected items given simulated input
- Do not mock the filesystem for symlink tests — use temp dirs

**Modules to test:**
- Discovery functions: given a temp dir structure with `commands/` and `agents/` subdirs containing example files, assert correct items are returned
- Selector step header: assert that `StepInfo{2, 3}` with title `"Commands"` renders the correct header string
- Init/profile orchestration: integration-style test that runs all 3 steps and asserts the resulting profile directory contains the correct symlinks

**Prior art:** existing skill discovery tests and symlink tests in the codebase follow the temp-dir pattern and serve as reference.

## Out of Scope

- Creating or editing command/agent file content from within the picker (open in `$EDITOR`)
- Adding new commands or agents inline during the picker flow
- Renaming items in the picker
- Filtering or grouping items within a single step by source backup directory
- Any UI beyond the existing Bubbletea/checkbox style (Lipgloss redesign is already planned as a separate phase)

## Further Notes

- The `symlink.Target` struct already has `Skills`, `Commands`, and `Agents` fields, so the symlink layer requires no schema changes.
- Profile directory structure already includes `commands/` and `agents/` subdirs — no filesystem layout changes needed.
- Test fixture files should be created in the existing temp-dir test setup, not committed as static fixtures, to keep the repo clean.
