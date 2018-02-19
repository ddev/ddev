package vtclean

import (
	"testing"
)

var tests = map[string]string{
	// "set title" special case
	"\x1b]0;asdjklfasdkljf\atest": "test",

	"hi man\x1b[3Gdude": "hi dude",

	// basic escape
	"\033[12laaa":    "aaa",
	"\033[?1049laaa": "aaa",

	// for the second regex
	"a\033[!pa": "aa",

	// backspace and clear
	"aaa\b\bb":        "aba",
	"aaa\b\b\033[K":   "a",
	"aaa\b\b\033[1K":  "  a",
	"aaa\b\b\033[2Ka": " a ",

	// character movement
	"aaa\033[2Db":        "aba",
	"aaa\033[4D\033[2Cb": "aab",
	"aaa\033[4D\033[1Cb": "aba",
	"aaa\033[1Cb":        "aaab",

	// vt52
	"aaa\033D\033Db": "aba",
	"a\033@b":        "ab",

	// delete and insert
	"aaa\b\b\033[2@": "a  aa",
	"aaa\b\b\033[P":  "aa",
	"aaa\b\b\033[4P": "a",

	// strip color
	"aaa \033[25;25mtest": "aaa test",

	"bbb \033]4;1;rgb:38/54/71\033\\test": "bbb test",
	"ccc \033]4;1;rgb:38/54/71test":       "ccc rgb:38/54/71test",
}

var colorTests = map[string]string{
	"aaa \033[25;25mtest": "aaa \033[25;25mtest\x1b[0m",
}

func TestMain(t *testing.T) {
	for a, b := range tests {
		tmp := Clean(a, false)
		if tmp != b {
			t.Logf("Clean() failed: %#v -> %#v != %#v\n", a, tmp, b)
			t.Fail()
		}
	}
}

func TestColor(t *testing.T) {
	for a, b := range colorTests {
		tmp := Clean(a, true)
		if tmp != b {
			t.Logf("Clean() failed: %#v -> %#v != %#v\n", a, tmp, b)
			t.Fail()
		}
	}
}

func TestWriteBounds(t *testing.T) {
	l := &lineEdit{buf: nil}
	s := "asdf"
	l.Write([]byte(s))
	if l.String() != s {
		t.Fatalf("l.String(): %#v != %#v", l.String(), s)
	}
}
