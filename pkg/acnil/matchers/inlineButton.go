package matchers

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func ToOneDimension(in [][]tele.InlineButton) []tele.InlineButton {
	out := []tele.InlineButton{}
	for _, b := range in {
		out = append(out, b...)
	}
	return out
}

func WithButtonText(text string) *InlineKeyboardTextMatcher {
	return &InlineKeyboardTextMatcher{
		Expected: text,
	}

}

type InlineKeyboardTextMatcher struct {
	Expected string
	actual   string
}

func (m *InlineKeyboardTextMatcher) String() string {
	return fmt.Sprintf("A button with name %s", m.Expected)
}
func (m *InlineKeyboardTextMatcher) GoString() string {
	return fmt.Sprintf("A button with name %s", m.Expected)
}

func (m *InlineKeyboardTextMatcher) Match(actual interface{}) (success bool, err error) {
	actualButton, ok := actual.(tele.InlineButton)
	if !ok {
		return false, fmt.Errorf("actual value must be of type tele.InlineButton, but it was %#v", actual)
	}
	m.actual = actualButton.Text
	return m.actual == m.Expected, nil
}

func (m *InlineKeyboardTextMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected button text to be %s, but it was %s", m.Expected, m.actual)
}

func (m *InlineKeyboardTextMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected button text to NOT be %s, but it was exactly %s", m.Expected, m.actual)
}
