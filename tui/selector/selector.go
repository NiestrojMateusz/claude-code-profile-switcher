package selector

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Item is a single entry in the selector list.
type Item struct {
	Name     string
	Selected bool
}

// Model is the Bubbletea model for the multi-select list.
// All fields are exported so tests can set up state directly.
type Model struct {
	Items      []Item
	Cursor     int
	Filter     string
	Done       bool
	Confirmed  bool // true only when the user pressed Enter (not Esc/Ctrl+C)
	Step       int  // current step number (0 means no step header)
	TotalSteps int  // total number of steps
	Title      string
	Height     int // terminal height from WindowSizeMsg; 0 = unlimited
}

// New returns a Model pre-loaded with the given item names.
func New(names []string) Model {
	items := make([]Item, len(names))
	for i, n := range names {
		items[i] = Item{Name: n}
	}
	return Model{Items: items}
}

// Init satisfies tea.Model. No I/O on startup.
func (m Model) Init() tea.Cmd { return nil }

// Update handles key events and returns the next model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Height = msg.Height
		return m, nil
	case tea.KeyMsg:
		visible := m.Visible()

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Done = true
			return m, tea.Quit

		case tea.KeyEnter:
			m.Done = true
			m.Confirmed = true
			return m, tea.Quit

		case tea.KeySpace:
			// toggle selection on the item under the cursor
			if m.Cursor < len(visible) {
				name := visible[m.Cursor].Name
				for i := range m.Items {
					if m.Items[i].Name == name {
						m.Items[i].Selected = !m.Items[i].Selected
						break
					}
				}
			}

		case tea.KeyUp:
			if m.Cursor > 0 {
				m.Cursor--
			}

		case tea.KeyDown:
			if m.Cursor < len(visible)-1 {
				m.Cursor++
			}

		case tea.KeyBackspace:
			if len(m.Filter) > 0 {
				m.Filter = m.Filter[:len(m.Filter)-1]
				m.Cursor = 0
			}

		default:
			// rune input: treat as filter characters OR vim-style nav
			if len(msg.Runes) == 1 {
				switch string(msg.Runes) {
				case "j":
					if m.Cursor < len(visible)-1 {
						m.Cursor++
					}
				case "k":
					if m.Cursor > 0 {
						m.Cursor--
					}
				default:
					m.Filter += string(msg.Runes)
					m.Cursor = 0
				}
			}
		}
	}
	return m, nil
}

// NewWithStep returns a Model pre-loaded with the given item names and step header info.
func NewWithStep(names []string, step, totalSteps int, title string) Model {
	m := New(names)
	m.Step = step
	m.TotalSteps = totalSteps
	m.Title = title
	return m
}

// scrollOffset returns the index of the first visible item to render,
// keeping the cursor within the viewport window.
func (m Model) scrollOffset() int {
	if m.Height == 0 {
		return 0
	}
	// header (step + filter) + hint line + blank = up to 4 fixed lines
	fixed := 4
	viewH := m.Height - fixed
	if viewH <= 0 || m.Cursor < viewH {
		return 0
	}
	return m.Cursor - viewH + 1
}

// View renders the list with viewport clamping when Height is set.
func (m Model) View() string {
	var b strings.Builder
	if m.Step > 0 {
		b.WriteString(fmt.Sprintf("Step %d of %d: Select %s\n", m.Step, m.TotalSteps, m.Title))
	}
	if m.Filter != "" {
		b.WriteString("filter: " + m.Filter + "\n")
	}

	visible := m.Visible()
	offset := m.scrollOffset()

	end := len(visible)
	if m.Height > 0 {
		fixed := 4
		viewH := m.Height - fixed
		if viewH > 0 && offset+viewH < end {
			end = offset + viewH
		}
	}

	for i, item := range visible[offset:end] {
		cursor := "  "
		if offset+i == m.Cursor {
			cursor = "> "
		}
		check := "[ ]"
		if item.Selected {
			check = "[x]"
		}
		b.WriteString(cursor + check + " " + item.Name + "\n")
	}
	b.WriteString("\nspace: toggle  enter: confirm  esc: cancel\n")
	return b.String()
}

// Visible returns items whose Name contains the current filter string.
func (m Model) Visible() []Item {
	if m.Filter == "" {
		return m.Items
	}
	var out []Item
	for _, item := range m.Items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(m.Filter)) {
			out = append(out, item)
		}
	}
	return out
}

// Selected returns the names of all checked items.
func (m Model) Selected() []string {
	var out []string
	for _, item := range m.Items {
		if item.Selected {
			out = append(out, item.Name)
		}
	}
	return out
}

// NewWithSelected returns a Model with the given items, pre-checking those in preSelected.
func NewWithSelected(names, preSelected []string) Model {
	selected := make(map[string]bool, len(preSelected))
	for _, s := range preSelected {
		selected[s] = true
	}
	items := make([]Item, len(names))
	for i, n := range names {
		items[i] = Item{Name: n, Selected: selected[n]}
	}
	return Model{Items: items}
}

// RunModel starts the interactive TUI from an existing Model and returns selected item names.
// Returns nil, nil when the user cancels (Esc/Ctrl+C).
func RunModel(m Model) ([]string, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(Model)
	if !final.Confirmed {
		return nil, nil
	}
	return final.Selected(), nil
}

// Run starts the interactive TUI and returns the selected item names.
// Returns nil, nil when the user cancels (Esc/Ctrl+C).
func Run(items []string) ([]string, error) {
	return RunWithSelected(items, nil)
}

// RunWithSelected starts the TUI with preSelected items pre-checked.
// Returns nil, nil when the user cancels (Esc/Ctrl+C).
func RunWithSelected(items, preSelected []string) ([]string, error) {
	m := NewWithSelected(items, preSelected)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(Model)
	if !final.Confirmed {
		return nil, nil
	}
	return final.Selected(), nil
}
