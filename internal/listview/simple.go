package listview

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	listHeight        = 14
	listWidth         = 20
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

type Option struct {
	Label string
	Value string
}

type NoItemsError struct {
	Title string
}

func (e *NoItemsError) Error() string {
	return fmt.Sprintf("%s: no selectable items", e.Title)
}

func RenderOptions(title string, options []Option) (string, bool, error) {
	if len(options) == 0 {
		return "", false, &NoItemsError{Title: title}
	}

	items := make([]list.Item, 0, len(options))
	for _, option := range options {
		items = append(items, item{Option: option})
	}

	listModel := list.New(items, itemDelegate{}, listWidth, listHeight)
	listModel.Title = title
	m := model{list: listModel}

	p := tea.NewProgram(m)

	mi, err := p.Run()

	if err != nil {
		return "", false, err
	}

	m, ok := mi.(model)
	if !ok {
		return "", false, fmt.Errorf("unexpected model type %T", mi)
	}

	return m.choice, m.quitting, nil
}

type item struct {
	Option
}

func (i item) FilterValue() string { return i.Label }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Label)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + s[0])
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			if m.list.FilterState() == list.Filtering {
				break
			}
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.choice = i.Value
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}
