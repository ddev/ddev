package heredoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndent(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		indent string
	}{
		{
			name:   "spaces only",
			input:  "  \t  \t   \n  ",
			want:   "  \t  \t   \n  ",
			indent: "  ",
		},
		{
			name:   "nothing to add or remove",
			input:  "one\ntwo\nthree\n",
			want:   "one\ntwo\nthree\n",
			indent: "",
		},
		{
			name:   "two spaces indent",
			input:  "one\n\ttwo\n\tthree\nfour\n\t",
			want:   "  one\n  \ttwo\n  \tthree\n  four\n  \t",
			indent: "  ",
		},
		{
			name:   "tab indent",
			input:  "one\n  two\n  three\nfour\n  ",
			want:   "\tone\n\t  two\n\t  three\n\tfour\n\t  ",
			indent: "\t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Indent(tt.input, tt.indent)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDoc(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "nothing to add or remove",
			input: "one\ntwo\nthree\n",
			want:  "one\ntwo\nthree\n",
		},
		{
			name:  "tabs to two spaces indent",
			input: "\n\t\tone\n\t\t\ttwo\n\t\t\tthree\n\t\tfour\n\t",
			want:  "one\n\ttwo\n\tthree\nfour\n",
		},
		{
			name:  "spaces to two spaces indent",
			input: "\n    one\n      two\n      three\n    four\n  ",
			want:  "one\n  two\n  three\nfour\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Doc(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
func TestDocIndent(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		indent string
	}{
		{
			name:   "nothing to add or remove",
			input:  "one\ntwo\nthree\n",
			want:   "one\ntwo\nthree\n",
			indent: "",
		},
		{
			name:   "two spaces indent",
			input:  "\n\t\tone\n\t\t\ttwo\n\t\t\tthree\n\t\tfour\n\t",
			want:   "  one\n  \ttwo\n  \tthree\n  four\n",
			indent: "  ",
		},
		{
			name:   "tab indent",
			input:  "\n    one\n      two\n      three\n    four\n  ",
			want:   "\tone\n\t  two\n\t  three\n\tfour\n",
			indent: "\t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DocIndent(tt.input, tt.indent)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDocI2S(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "two spaces indent",
			input: "one\ntwo\nthree\n",
			want:  "  one\n  two\n  three\n",
		},
		{
			name:  "tabs to two spaces indent",
			input: "\n\t\tone\n\t\t\ttwo\n\t\t\tthree\n\t\tfour\n\t",
			want:  "  one\n  \ttwo\n  \tthree\n  four\n",
		},
		{
			name:  "spaces to two spaces indent",
			input: "\n    one\n      two\n      three\n    four\n  ",
			want:  "  one\n    two\n    three\n  four\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DocI2S(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
