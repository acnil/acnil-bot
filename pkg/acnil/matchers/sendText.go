package matchers

import (
	"fmt"
	"strings"
)

type Matcher interface {
	// Matches returns whether x is a match.
	Matches(x interface{}) bool

	// String describes what the matcher matches.
	String() string
}

func ContainsString(text string) *SendTextMatcher {
	return &SendTextMatcher{
		Expect: text,
	}
}

type SendTextMatcher struct {
	Expect string
}

func (m *SendTextMatcher) Matches(x interface{}) bool {
	v, ok := x.(string)
	if !ok {
		return false
	}
	return strings.Contains(v, m.Expect)
}

func (m *SendTextMatcher) String() string {
	return fmt.Sprintf("Contains \"%s\"", m.Expect)
}
