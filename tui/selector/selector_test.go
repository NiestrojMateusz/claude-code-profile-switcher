package selector

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(k string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func specialKey(t tea.KeyType) tea.Msg {
	return tea.KeyMsg{Type: t}
}

func TestInitialModelListsAllItems(t *testing.T) {
	m := New([]string{"git-helper", "code-review", "test-runner"})
	if len(m.Items) != 3 {
		t.Errorf("want 3 items, got %d", len(m.Items))
	}
	for _, item := range m.Items {
		if item.Selected {
			t.Errorf("item %q should start unselected", item.Name)
		}
	}
	if m.Cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", m.Cursor)
	}
}

func TestCursorMovesDown(t *testing.T) {
	m := New([]string{"a", "b", "c"})
	next, _ := m.Update(key("j"))
	m = next.(Model)
	if m.Cursor != 1 {
		t.Errorf("cursor should be 1 after j, got %d", m.Cursor)
	}
}

func TestCursorDoesNotGoBelow(t *testing.T) {
	m := New([]string{"a", "b"})
	m.Cursor = 1
	next, _ := m.Update(key("j"))
	m = next.(Model)
	if m.Cursor != 1 {
		t.Errorf("cursor should stay at 1 (last item), got %d", m.Cursor)
	}
}

func TestCursorMovesUp(t *testing.T) {
	m := New([]string{"a", "b", "c"})
	m.Cursor = 2
	next, _ := m.Update(key("k"))
	m = next.(Model)
	if m.Cursor != 1 {
		t.Errorf("cursor should be 1 after k, got %d", m.Cursor)
	}
}

func TestCursorDoesNotGoAboveZero(t *testing.T) {
	m := New([]string{"a", "b"})
	m.Cursor = 0
	next, _ := m.Update(key("k"))
	m = next.(Model)
	if m.Cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.Cursor)
	}
}

func TestSpaceTogglesSelection(t *testing.T) {
	m := New([]string{"a", "b"})
	next, _ := m.Update(specialKey(tea.KeySpace))
	m = next.(Model)
	if !m.Items[0].Selected {
		t.Error("item 0 should be selected after space")
	}
	// toggle off
	next, _ = m.Update(specialKey(tea.KeySpace))
	m = next.(Model)
	if m.Items[0].Selected {
		t.Error("item 0 should be deselected after second space")
	}
}

func TestSelectedReturnsCheckedNames(t *testing.T) {
	m := New([]string{"a", "b", "c"})
	m.Items[0].Selected = true
	m.Items[2].Selected = true
	got := m.Selected()
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Errorf("Selected(): want [a c], got %v", got)
	}
}

func TestEnterSetsConfirmed(t *testing.T) {
	m := New([]string{"a", "b"})
	next, _ := m.Update(specialKey(tea.KeyEnter))
	m = next.(Model)
	if !m.Confirmed {
		t.Error("Enter should set Confirmed=true")
	}
	if !m.Done {
		t.Error("Enter should set Done=true")
	}
}

func TestEscDoesNotSetConfirmed(t *testing.T) {
	m := New([]string{"a", "b"})
	next, _ := m.Update(specialKey(tea.KeyEsc))
	m = next.(Model)
	if m.Confirmed {
		t.Error("Esc should not set Confirmed")
	}
	if !m.Done {
		t.Error("Esc should set Done=true")
	}
}

func TestNewWithSelectedPreChecksItems(t *testing.T) {
	m := NewWithSelected([]string{"a", "b", "c"}, []string{"a", "c"})
	if !m.Items[0].Selected {
		t.Error("item 'a' should be pre-selected")
	}
	if m.Items[1].Selected {
		t.Error("item 'b' should not be pre-selected")
	}
	if !m.Items[2].Selected {
		t.Error("item 'c' should be pre-selected")
	}
}

func TestViewRendersStepHeader(t *testing.T) {
	m := New([]string{"a", "b"})
	m.Step = 2
	m.TotalSteps = 3
	m.Title = "Commands"
	view := m.View()
	want := "Step 2 of 3: Select Commands"
	if !strings.Contains(view, want) {
		t.Errorf("View() should contain %q, got:\n%s", want, view)
	}
}

func TestViewOmitsStepHeaderWhenNotSet(t *testing.T) {
	m := New([]string{"a", "b"})
	view := m.View()
	if strings.Contains(view, "Step") {
		t.Errorf("View() should not contain 'Step' when no step info set, got:\n%s", view)
	}
}

func TestNewWithStepInfo(t *testing.T) {
	m := NewWithStep([]string{"skill-a", "skill-b"}, 1, 3, "Skills")
	if m.Step != 1 {
		t.Errorf("Step: want 1, got %d", m.Step)
	}
	if m.TotalSteps != 3 {
		t.Errorf("TotalSteps: want 3, got %d", m.TotalSteps)
	}
	if m.Title != "Skills" {
		t.Errorf("Title: want Skills, got %q", m.Title)
	}
	if len(m.Items) != 2 {
		t.Errorf("Items: want 2, got %d", len(m.Items))
	}
}

func TestFilterNarrowsList(t *testing.T) {
	m := New([]string{"git-helper", "git-blame", "test-runner"})
	// type "git" into filter
	for _, ch := range "git" {
		next, _ := m.Update(key(string(ch)))
		m = next.(Model)
	}
	visible := m.Visible()
	if len(visible) != 2 {
		t.Errorf("filter 'git' should show 2 items, got %d: %v", len(visible), visible)
	}
}
