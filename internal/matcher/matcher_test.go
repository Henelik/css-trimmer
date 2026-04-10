package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindSubMatch(t *testing.T) {
	testCases := []struct {
		name    string
		prefix  string
		postfix string
		content string
		want    []string
	}{
		{
			name:    "emtpy content",
			prefix:  `class="`,
			postfix: `"`,
			content: "",
			want:    []string{},
		},
		{
			name:    "single match",
			prefix:  `class="`,
			postfix: `"`,
			content: `abcdefg class="meow_mix"`,
			want:    []string{"meow_mix"},
		},
		{
			name:    "multiple matches",
			prefix:  `class="`,
			postfix: `"`,
			content: `abc class="meow_mix" def class="doggy_dinner" ghi class="gopher_ganache"`,
			want:    []string{"meow_mix", "doggy_dinner", "gopher_ganache"},
		},
		{
			name:    "partial match",
			prefix:  `class="`,
			postfix: `"`,
			content: `class="meow_mix`,
			want:    []string{},
		},
		{
			name:    "empty match",
			prefix:  `class="`,
			postfix: `"`,
			content: `class=""`,
			want:    []string{""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FindSubMatches(tc.prefix, tc.postfix, tc.content)

			assert.Equal(t, tc.want, got)
		})
	}
}
