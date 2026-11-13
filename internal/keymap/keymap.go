// Package keymap lets the user choose the keys for the app's actions.
//
// The app itself always works with its default keys. A Map sits in front of
// it and turns the key the user pressed into the default key of the action
// that key is bound to, so pages need not know keys can change.
package keymap

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Scope is the part of the app an action belongs to. A key only has to be
// unique among the actions that can be reached at the same time: those of
// one scope plus the global ones.
type Scope int

const (
	None   Scope = iota // A screen with no actions of its own
	Global              // Works everywhere
	Page                // The page view
	Rows                // The rows of a database
	Tree                // The sidebar, when it has focus
)

// Action is something the user can do with a key press.
type Action struct {
	Name    string // Name used under "keys:" in the config file
	Scope   Scope
	Default string // Key as Bubble Tea names it: "e", "ctrl+p", "tab"
}

// Actions lists every action whose key can be changed.
func Actions() []Action {
	return []Action{
		{"help", Global, "?"},
		{"palette", Global, "ctrl+p"},
		{"sidebar", Global, "ctrl+b"},
		{"focus", Global, "tab"},
		{"quit", Global, "q"},
		{"new_page", Global, "ctrl+n"},

		{"edit", Page, "e"},
		{"properties", Page, "p"},
		{"todos", Page, "t"},
		{"links", Page, "o"},
		{"back", Page, "["},
		{"forward", Page, "]"},
		{"find", Page, "/"},
		{"find_next", Page, "n"},
		{"find_previous", Page, "N"},
		{"archive", Page, "D"},
		{"reload", Page, "r"},

		{"row_new", Rows, "n"},
		{"row_sort", Rows, "s"},
		{"row_filter", Rows, "/"},
		{"row_reload", Rows, "r"},
		{"row_more", Rows, "m"},

		{"tree_reload", Tree, "r"},
	}
}

// reserved keys move around, confirm, cancel or quit everywhere in the app
// and cannot be given to an action.
var reserved = map[string]bool{
	"ctrl+c": true, "esc": true, "enter": true,
	"up": true, "down": true, "left": true, "right": true,
	"pgup": true, "pgdown": true, "home": true, "end": true,
	"j": true, "k": true, "h": true, "l": true, " ": true,
	"g": true, "G": true, "f": true, "b": true, "u": true, "d": true,
}

// A key is one visible character, or a named key the terminal can send.
// ctrl+i and ctrl+m are left out: terminals send them as tab and enter.
var keyPattern = regexp.MustCompile(`^(\S|ctrl\+[abd-hj-ln-z]|alt\+\S|f[1-9]|f1[0-2]|tab|shift\+tab|backspace|delete)$`)

// Map holds the user's key choices.
type Map struct {
	keys     map[string]string               // Action name -> key in effect
	incoming map[Scope]map[string]tea.KeyMsg // Chosen key -> default key press
	unbound  map[Scope]map[string]bool       // Default keys given up by their action
}

// Default returns a Map with every action on its default key.
func Default() *Map {
	m, err := New(nil)
	if err != nil {
		panic("keymap: default keys are invalid: " + err.Error())
	}
	return m
}

// New builds a Map from the "keys" section of the config file: action name
// to key. It fails on an unknown action, a key that cannot be used, and two
// actions that would share a key on the same screen.
func New(overrides map[string]string) (*Map, error) {
	actions := Actions()
	known := make(map[string]bool, len(actions))
	for _, action := range actions {
		known[action.Name] = true
	}

	names := make([]string, 0, len(overrides))
	for name := range overrides {
		names = append(names, name)
	}
	sort.Strings(names) // Errors must not depend on map order

	m := &Map{
		keys:     make(map[string]string, len(actions)),
		incoming: map[Scope]map[string]tea.KeyMsg{},
		unbound:  map[Scope]map[string]bool{},
	}
	for _, action := range actions {
		m.keys[action.Name] = action.Default
	}
	for _, name := range names {
		if _, ok := known[name]; !ok {
			return nil, fmt.Errorf("keys: unknown action %q (known: %s)", name, strings.Join(actionNames(actions), ", "))
		}
		key := overrides[name]
		if err := checkKey(key); err != nil {
			return nil, fmt.Errorf("keys: %s: %w", name, err)
		}
		m.keys[name] = key
	}

	if err := m.checkConflicts(actions); err != nil {
		return nil, err
	}

	for _, action := range actions {
		key := m.keys[action.Name]
		if key == action.Default {
			continue
		}
		if m.incoming[action.Scope] == nil {
			m.incoming[action.Scope] = map[string]tea.KeyMsg{}
			m.unbound[action.Scope] = map[string]bool{}
		}
		m.incoming[action.Scope][key] = keyPress(action.Default)
		m.unbound[action.Scope][action.Default] = true
	}
	return m, nil
}

func actionNames(actions []Action) []string {
	names := make([]string, len(actions))
	for i, action := range actions {
		names[i] = action.Name
	}
	return names
}

func checkKey(key string) error {
	switch {
	case key == "":
		return fmt.Errorf("no key given")
	case reserved[key]:
		return fmt.Errorf("%q is used for moving, confirming or leaving and cannot be reassigned", key)
	case !keyPattern.MatchString(key):
		return fmt.Errorf("%q is not a key (use one character, ctrl+<letter>, alt+<character>, f1-f12, tab, shift+tab, backspace or delete)", key)
	}
	return nil
}

// checkConflicts makes sure no key means two things on one screen.
func (m *Map) checkConflicts(actions []Action) error {
	for _, screen := range []Scope{Page, Rows, Tree} {
		owner := map[string]string{}
		for _, action := range actions {
			if action.Scope != Global && action.Scope != screen {
				continue
			}
			key := m.keys[action.Name]
			if other, taken := owner[key]; taken {
				return fmt.Errorf("keys: %q is bound to both %s and %s", key, other, action.Name)
			}
			owner[key] = action.Name
		}
	}
	return nil
}

// keyPress builds the key press Bubble Tea would report for a default key.
func keyPress(key string) tea.KeyMsg {
	switch key {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "ctrl+b":
		return tea.KeyMsg{Type: tea.KeyCtrlB}
	case "ctrl+n":
		return tea.KeyMsg{Type: tea.KeyCtrlN}
	case "ctrl+p":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// isText reports whether a key press would type a character.
func isText(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeySpace || (msg.Type == tea.KeyRunes && !msg.Alt)
}

// Translate turns the key the user pressed into the key the app expects.
// screen says which actions are reachable besides the global ones. While
// text is being typed (typing is true) characters are left alone and only
// global actions on non-character keys are translated.
//
// ok is false when the key should be ignored: it is the default key of an
// action the user moved elsewhere.
func (m *Map) Translate(msg tea.KeyMsg, screen Scope, typing bool) (translated tea.KeyMsg, ok bool) {
	if m == nil || len(m.incoming) == 0 || msg.Paste {
		return msg, true
	}
	scopes := []Scope{screen, Global}
	if typing {
		if isText(msg) {
			return msg, true
		}
		scopes = []Scope{Global}
	}

	key := msg.String()
	for _, scope := range scopes {
		if press, found := m.incoming[scope][key]; found {
			return press, true
		}
	}
	for _, scope := range scopes {
		if m.unbound[scope][key] {
			return msg, false
		}
	}
	return msg, true
}

// Key returns the key in effect for an action, or "" for an unknown action.
func (m *Map) Key(action string) string {
	if m == nil {
		return Default().Key(action)
	}
	return m.keys[action]
}

var placeholder = regexp.MustCompile(`\{([a-z_]+)\}`)

// Label fills "{action}" placeholders in help text with the keys in effect,
// so "{find}  {find_next}" reads "/  n" by default.
func (m *Map) Label(text string) string {
	return placeholder.ReplaceAllStringFunc(text, func(match string) string {
		if key := m.Key(match[1 : len(match)-1]); key != "" {
			return key
		}
		return match
	})
}
