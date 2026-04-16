package monitor

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ActionKind describes what to do with a detected skill.
type ActionKind int

const (
	ActionSkip    ActionKind = iota // dismiss; will reappear next run
	ActionNever                     // mark as known; never surface again
	ActionAdd                       // add to the named profile
)

// Choice is the resolved decision for one skill.
type Choice struct {
	Skill   string
	Action  ActionKind
	Profile string // only meaningful when Action == ActionAdd
}

// row holds the per-skill UI state.
type row struct {
	skill   string
	action  ActionKind
	profile string // set when action == ActionAdd
}

// Model is the Bubbletea model for the new-skill action prompt.
type Model struct {
	rows     []row
	cursor   int
	profiles []string // available profiles to add to
	quitting bool     // true when user pressed q without confirming
	done     bool     // true when user pressed enter to confirm
}

// New returns a model for the given list of new skill names and available profiles.
func New(newSkills, profiles []string) Model {
	rows := make([]row, len(newSkills))
	for i, s := range newSkills {
		rows[i] = row{skill: s, action: ActionSkip}
	}
	return Model{rows: rows, profiles: profiles}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "n":
			m.rows[m.cursor].action = ActionNever
		case "s":
			m.rows[m.cursor].action = ActionSkip
		case "a":
			// Cycle through available profiles for ActionAdd.
			if len(m.profiles) > 0 {
				r := &m.rows[m.cursor]
				if r.action != ActionAdd {
					r.action = ActionAdd
					r.profile = m.profiles[0]
				} else {
					// Already ActionAdd — advance to next profile, wrapping around.
					idx := 0
					for i, p := range m.profiles {
						if p == r.profile {
							idx = i
							break
						}
					}
					r.profile = m.profiles[(idx+1)%len(m.profiles)]
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.rows) == 0 {
		return ""
	}
	var s string
	s += "New skills detected — choose an action for each:\n\n"
	for i, r := range m.rows {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		action := actionLabel(r)
		s += cursor + r.skill + "  [" + action + "]\n"
	}
	s += "\nKeys: ↑/↓ navigate  a=add  s=skip  n=never ask  enter=confirm  q=quit\n"
	return s
}

func actionLabel(r row) string {
	switch r.action {
	case ActionNever:
		return "never ask"
	case ActionAdd:
		if r.profile != "" {
			return "add → " + r.profile
		}
		return "add"
	default:
		return "skip"
	}
}

// Results returns the list of choices made by the user.
// Returns nil when the user quit without confirming (q key) or when the model
// was never run to completion.
func (m Model) Results() []Choice {
	if !m.done {
		return nil
	}
	choices := make([]Choice, len(m.rows))
	for i, r := range m.rows {
		choices[i] = Choice{Skill: r.skill, Action: r.action, Profile: r.profile}
	}
	return choices
}

// Run starts the interactive TUI for the given new skills and available profiles.
// Returns nil, nil when the user quits without confirming.
func Run(newSkills, profiles []string) ([]Choice, error) {
	m := New(newSkills, profiles)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	return result.(Model).Results(), nil
}
