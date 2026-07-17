package listview

import (
	"errors"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func TestItemFilterValueUsesLabelAndRetainsValue(t *testing.T) {
	option := Option{
		Label: "service:web abc123",
		Value: "arn:.../abc123",
	}
	i := item{Option: option}

	if got := i.FilterValue(); got != option.Label {
		t.Fatalf("FilterValue() = %q, want %q", got, option.Label)
	}
	if i.Value != option.Value {
		t.Fatalf("item.Value = %q, want %q", i.Value, option.Value)
	}
}

func TestRenderOptionsReturnsNoItemsError(t *testing.T) {
	const title = "Select a service"

	_, _, err := RenderOptions(title, nil)
	if err == nil {
		t.Fatal("RenderOptions() error = nil, want *NoItemsError")
	}

	var noItemsErr *NoItemsError
	if !errors.As(err, &noItemsErr) {
		t.Fatalf("RenderOptions() error type = %T, want *NoItemsError", err)
	}
	if noItemsErr.Title != title {
		t.Fatalf("NoItemsError.Title = %q, want %q", noItemsErr.Title, title)
	}
}

func TestModelEnterSelectsValueWhenLabelsAreDuplicated(t *testing.T) {
	const label = "service:web abc123"
	items := []list.Item{
		item{Option: Option{Label: label, Value: "arn:aws:ecs:us-east-1:123456789012:service/first/abc123"}},
		item{Option: Option{Label: label, Value: "arn:aws:ecs:us-east-1:123456789012:service/second/abc123"}},
	}
	listModel := list.New(items, itemDelegate{}, listWidth, listHeight)
	listModel.Select(1)

	var enter tea.KeyMsg = tea.KeyPressMsg{Code: tea.KeyEnter}
	updated, cmd := (model{list: listModel}).Update(enter)
	got, ok := updated.(model)
	if !ok {
		t.Fatalf("Update() model type = %T, want listview.model", updated)
	}
	if got.choice != items[1].(item).Value {
		t.Fatalf("Update() choice = %q, want %q", got.choice, items[1].(item).Value)
	}
	if cmd == nil {
		t.Fatal("Update() command = nil, want tea.Quit")
	}
	cmdMsg := cmd()
	if _, ok := cmdMsg.(tea.QuitMsg); !ok {
		t.Fatalf("Update() command message type = %T, want tea.QuitMsg", cmdMsg)
	}
}
