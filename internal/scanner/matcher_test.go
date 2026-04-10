package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindClassMatch(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "emtpy content",
			content: "",
			want:    nil,
		},
		{
			name:    "single match",
			content: `abcdefg class="meow_mix"`,
			want:    []string{"meow_mix"},
		},
		{
			name:    "multiple matches",
			content: `abc class="meow_mix" def class="doggy_dinner" ghi class="gopher_ganache"`,
			want:    []string{"meow_mix", "doggy_dinner", "gopher_ganache"},
		},
		{
			name:    "partial match",
			content: `class="meow_mix`,
			want:    nil,
		},
		{
			name:    "empty match",
			content: `class=""`,
			want:    []string{""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := findClassMatch(tc.content)

			assert.Equal(t, tc.want, got)
		})
	}
}
