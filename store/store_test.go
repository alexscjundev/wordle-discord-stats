package store

import "testing"

func TestNormalizeFixedNick(t *testing.T) {
	cases := []struct{ in, want string }{
		{`good food muncher \(food in nc\)`, `good food muncher (food in nc)`},
		{`no parens at all`, `no parens at all`},
		{`already clean (food in nc)`, `already clean (food in nc)`},
		{`double \\(backslash\\)`, `double (backslash)`},
		{`trailing backslash \`, `trailing backslash \`},
	}
	for _, c := range cases {
		if got := normalizeFixedNick(c.in); got != c.want {
			t.Errorf("normalizeFixedNick(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
