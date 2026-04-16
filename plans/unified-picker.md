# Plan: Unified 3-Step Picker for Skills, Commands, and Agents

> Source PRD: docs/prd-unified-picker.md

## Architectural decisions

- **Picker order**: Skills → Commands → Agents (fixed, all three flows)
- **Step header format**: `"Step X of 3: Select <Category>"`
- **Discovery source**: backup directories only (same roots as skill discovery), scanning `commands/` and `agents/` subdirs
- **Symlink layer**: unchanged — `symlink.Target{Skills, Commands, Agents}` already models all three categories
- **Profile directory layout**: unchanged — `commands/` and `agents/` subdirs already exist per profile
- **Orchestration location**: inline in each command handler (`runInit`, `runProfileCreate`, `runProfileEdit`), not a separate module

---

## Phase 1: Step-aware selector

**User stories**: 6, 12

### What to build

Extend the existing generic selector TUI to accept a step header — a current step number, total step count, and category title. The selector renders this as a header line above the list. Selection mechanics (space, enter, j/k, filter by typing) are unchanged. No category knowledge leaks into the selector itself.

### Acceptance criteria

- [ ] Selector renders `"Step 2 of 3: Select Commands"` when given step 2 of 3 and title "Commands"
- [ ] Selector renders `"Step 1 of 3: Select Skills"` when given step 1 of 3 and title "Skills"
- [ ] Existing selection mechanics (space, j/k, enter, filter) unchanged
- [ ] Selector can be called without step info (backwards compatible, no header rendered)

---

## Phase 2: Command and agent discovery

**User stories**: 8, 9

### What to build

Two new discovery functions — one for commands, one for agents — that scan the same backup directories as skill discovery but look in `commands/` and `agents/` subdirs respectively. Seed example `.md` fixture files into these subdirs in temp/backup directories so discovery returns real results during manual testing and tests.

### Acceptance criteria

- [ ] `discoverCommands()` returns `.md` files found in `commands/` subdirs of backup dirs
- [ ] `discoverAgents()` returns `.md` files found in `agents/` subdirs of backup dirs
- [ ] Both functions return empty slice (not error) when no files found
- [ ] Example fixture files exist in temp dirs and are returned by discovery
- [ ] Discovery functions have unit tests using temp dir setup

---

## Phase 3: Init and profile create — 3-step flow

**User stories**: 1, 2, 3, 4, 7, 13

### What to build

Wire both the init flow and the profile create flow to run three sequential picker calls — skills, commands, agents — using the step-aware selector from Phase 1 and the discovery functions from Phase 2. After all three steps complete, symlink selected items from all three categories into the profile in one pass. Pressing enter on an empty list proceeds without selecting anything.

### Acceptance criteria

- [ ] Init runs skills picker (step 1 of 3), then commands picker (step 2 of 3), then agents picker (step 3 of 3)
- [ ] Profile create runs the same 3-step flow
- [ ] Selecting 0 items in any step proceeds to the next step without error
- [ ] After completion, selected skills, commands, and agents are all symlinked into the profile
- [ ] Unselected categories result in no symlinks for that category (not an error)
- [ ] Step header displays correctly in both flows

---

## Phase 4: Profile edit — 3-step flow with pre-selection

**User stories**: 5, 10, 11

### What to build

Extend profile edit to run the same 3-step picker flow, pre-populating each step with the currently active selections for that profile and category. After completion, re-apply symlinks for all three categories based on the new selections.

### Acceptance criteria

- [ ] Profile edit runs skills picker (step 1 of 3), commands picker (step 2 of 3), agents picker (step 3 of 3)
- [ ] Currently active skills are pre-checked in step 1
- [ ] Currently active commands are pre-checked in step 2
- [ ] Currently active agents are pre-checked in step 3
- [ ] Saving with unchanged selections produces the same symlink state
- [ ] Deselecting a previously selected item removes its symlink from the profile
- [ ] Adding a newly selected item creates its symlink in the profile
