package monitor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// sendKey simulates a key press and returns the updated model.
func sendKey(m Model, key string) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(Model)
}

func sendSpecialKey(m Model, key tea.KeyType) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	return updated.(Model)
}

// --- Initial state ---

func TestInitialActionIsSkip(t *testing.T) {
	m := New([]string{"skill-a", "skill-b"}, []string{"base"})
	for _, r := range m.rows {
		if r.action != ActionSkip {
			t.Errorf("row %q: want ActionSkip, got %v", r.skill, r.action)
		}
	}
}

func TestInitialCursorAtZero(t *testing.T) {
	m := New([]string{"skill-a"}, nil)
	if m.cursor != 0 {
		t.Errorf("cursor: want 0, got %d", m.cursor)
	}
}

// --- Navigation ---

func TestDownMoveCursor(t *testing.T) {
	m := New([]string{"skill-a", "skill-b"}, nil)
	m = sendKey(m, "j")
	if m.cursor != 1 {
		t.Errorf("cursor after down: want 1, got %d", m.cursor)
	}
}

func TestUpDoesNotGoBelowZero(t *testing.T) {
	m := New([]string{"skill-a"}, nil)
	m = sendKey(m, "k")
	if m.cursor != 0 {
		t.Errorf("cursor after up at 0: want 0, got %d", m.cursor)
	}
}

// --- Action selection ---

func TestNeverKeySetsActionNever(t *testing.T) {
	m := New([]string{"skill-a"}, nil)
	m = sendKey(m, "n")
	if m.rows[0].action != ActionNever {
		t.Errorf("want ActionNever, got %v", m.rows[0].action)
	}
}

func TestSkipKeyResetsAction(t *testing.T) {
	m := New([]string{"skill-a"}, nil)
	m = sendKey(m, "n") // set to never
	m = sendKey(m, "s") // back to skip
	if m.rows[0].action != ActionSkip {
		t.Errorf("want ActionSkip, got %v", m.rows[0].action)
	}
}

func TestAddKeyWithProfileSetsActionAdd(t *testing.T) {
	m := New([]string{"skill-a"}, []string{"base"})
	m = sendKey(m, "a")
	if m.rows[0].action != ActionAdd {
		t.Errorf("want ActionAdd, got %v", m.rows[0].action)
	}
	if m.rows[0].profile != "base" {
		t.Errorf("want profile 'base', got %q", m.rows[0].profile)
	}
}

func TestAddKeyWithNoProfilesDoesNothing(t *testing.T) {
	m := New([]string{"skill-a"}, nil)
	m = sendKey(m, "a")
	if m.rows[0].action != ActionSkip {
		t.Errorf("want ActionSkip when no profiles, got %v", m.rows[0].action)
	}
}

func TestAddKeyCyclesThroughProfiles(t *testing.T) {
	m := New([]string{"skill-a"}, []string{"base", "work", "personal"})

	m = sendKey(m, "a") // first press → base
	if m.rows[0].profile != "base" {
		t.Errorf("1st press: want 'base', got %q", m.rows[0].profile)
	}

	m = sendKey(m, "a") // second press → work
	if m.rows[0].profile != "work" {
		t.Errorf("2nd press: want 'work', got %q", m.rows[0].profile)
	}

	m = sendKey(m, "a") // third press → personal
	if m.rows[0].profile != "personal" {
		t.Errorf("3rd press: want 'personal', got %q", m.rows[0].profile)
	}

	m = sendKey(m, "a") // fourth press wraps → base
	if m.rows[0].profile != "base" {
		t.Errorf("4th press (wrap): want 'base', got %q", m.rows[0].profile)
	}
}

func TestActionOnRowDoesNotAffectOtherRows(t *testing.T) {
	m := New([]string{"skill-a", "skill-b"}, nil)
	m = sendKey(m, "n") // set skill-a to never
	if m.rows[1].action != ActionSkip {
		t.Errorf("skill-b should still be ActionSkip, got %v", m.rows[1].action)
	}
}

// --- Results ---

func TestResultsNilOnQuit(t *testing.T) {
	m := New([]string{"skill-a"}, nil)
	m = sendKey(m, "q")
	if m.Results() != nil {
		t.Error("Results should be nil when user quits")
	}
}

func TestResultsReflectsChoices(t *testing.T) {
	m := New([]string{"skill-a", "skill-b"}, []string{"base"})
	m = sendKey(m, "n")         // skill-a → never
	m = sendKey(m, "j")         // move to skill-b
	m = sendKey(m, "a")         // skill-b → add:base
	m = sendSpecialKey(m, tea.KeyEnter) // confirm

	results := m.Results()
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}

	if results[0].Skill != "skill-a" || results[0].Action != ActionNever {
		t.Errorf("results[0]: want {skill-a, ActionNever}, got %+v", results[0])
	}
	if results[1].Skill != "skill-b" || results[1].Action != ActionAdd || results[1].Profile != "base" {
		t.Errorf("results[1]: want {skill-b, ActionAdd, base}, got %+v", results[1])
	}
}
