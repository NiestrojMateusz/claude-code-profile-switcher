# Fix: Selector TUI Viewport & Cursor Visibility

## Problem

`tui/selector/selector.go` — `View()` renders the full item list unconditionally. `RunModel` uses `tea.NewProgram(m)` with no options (inline mode). On large skill lists (> terminal height), the initial cursor at row 0 scrolls off-screen. User must press ↓ many times before `> ` enters the visible region.

## Root Cause

Two compounding issues:
1. **No altscreen** — inline mode appends to the shell scroll buffer; content taller than the terminal overflows upward.
2. **No viewport clamping** — `View()` emits every item with no windowing or scroll offset.

## Tracer Bullet (do this first, get feedback)

Add `tea.WithAltScreen()` to `RunModel` and `RunWithSelected`. This alone fixes the overflow — altscreen takes over the terminal, so the full list is contained. Verify the picker works at all before implementing scroll.

---

## Phase 1 — Altscreen (tracer bullet)

**File:** `tui/selector/selector.go`

**Change:** `tea.NewProgram(m)` → `tea.NewProgram(m, tea.WithAltScreen())`  
Apply in both `RunModel` (line 181) and `RunWithSelected` (line 208).

**Test:** Manual — run profile creation, confirm picker fills terminal cleanly and exits without corrupting shell output.

---

## Phase 2 — Viewport model

**Goal:** Clamp rendering to terminal height so the cursor is always visible even without altscreen (resilient, correct-by-construction).

### 2a. Store terminal height on Model

```go
// Add to Model struct
Height int // 0 = unlimited
```

Handle `tea.WindowSizeMsg` in `Update()`:
```go
case tea.WindowSizeMsg:
    m.Height = msg.Height
```

### 2b. Compute scroll offset

Add helper `scrollOffset(visible []Item) int`:
- If `m.Height == 0`, return 0 (no clamping).
- Reserve 4 lines for header/footer (step header, filter line, hint line, blank).
- `viewH := m.Height - 4`
- Keep cursor in window: `offset = clamp(m.Cursor - viewH + 1, 0, max(0, len(visible)-viewH))`

### 2c. Clamp `View()` render loop

```go
visible := m.Visible()
offset := m.scrollOffset(visible)
end := min(offset+viewH, len(visible))
for i, item := range visible[offset:end] { ... }
```

Cursor check: `if (offset + i) == m.Cursor`.

---

## Phase 3 — Tests (RED → GREEN)

### Test 1 — viewport clamping
```
RED:  given Height=10, 50 items → View() produces ≤ 10 lines
GREEN: scrollOffset + clamped slice in View()
```

### Test 2 — cursor always in view
```
RED:  Cursor=49, Height=10, 50 items → View() contains "> "
GREEN: scrollOffset ensures cursor is in the rendered window
```

### Test 3 — no clamping when Height=0
```
RED:  Height=0, 50 items → View() contains all 50 items
GREEN: scrollOffset returns 0 when Height==0
```

---

## TDD Cycle Order

1. `TestViewClampsToTerminalHeight` → RED → implement `scrollOffset` → GREEN  
2. `TestCursorAlwaysVisibleInViewport` → RED → verify clamping logic → GREEN  
3. `TestNoClamingWhenHeightUnset` → RED → guard `Height==0` → GREEN  
4. REFACTOR: extract `viewportHeight()` helper, clean up constants  

---

## Acceptance Criteria

- [ ] Altscreen mode: picker fills terminal, no shell corruption on exit
- [ ] `> ` cursor visible immediately on open, regardless of list length
- [ ] Scrolling works: navigate through lists longer than terminal height
- [ ] `Height=0` path renders full list (backward compat for tests)
- [ ] All existing tests pass unmodified
- [ ] 3 new tests covering the behaviors above
