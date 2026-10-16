package dotenv_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/dotenv"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		values map[string]string
		want   string
	}{
		{
			name:   "empty file",
			text:   "",
			values: map[string]string{"B": "2", "A": "1"},
			want:   "A=1\nB=2\n",
		},
		{
			name:   "append without trailing newline",
			text:   "X=1",
			values: map[string]string{"A": "a"},
			want:   "X=1\nA=a\n",
		},
		{
			name:   "replace in place keeps order and similar names",
			text:   "DB_HOST=old\nDB_HOST_EXTRA=keep\nOTHER=x\n",
			values: map[string]string{"DB_HOST": "new"},
			want:   "DB_HOST=new\nDB_HOST_EXTRA=keep\nOTHER=x\n",
		},
		{
			name:   "references to other variables stay for the framework",
			text:   "APP_NAME=Laravel\nMAIL_HOST=127.0.0.1\nMAIL_FROM_NAME=\"${APP_NAME}\"\nVITE_APP_NAME=\"${APP_NAME}\"\n",
			values: map[string]string{"MAIL_HOST": "mailpit"},
			want:   "APP_NAME=Laravel\nMAIL_HOST=mailpit\nMAIL_FROM_NAME=\"${APP_NAME}\"\nVITE_APP_NAME=\"${APP_NAME}\"\n",
		},
		{
			name:   "a value already set keeps how it is written",
			text:   "DB_HOST=db # comment\nDB_USER='db'\nexport DB_PASS=\"db\"\nDB_NAME=\n",
			values: map[string]string{"DB_HOST": "db", "DB_USER": "db", "DB_PASS": "db", "DB_NAME": ""},
			want:   "DB_HOST=db # comment\nDB_USER='db'\nexport DB_PASS=\"db\"\nDB_NAME=\n",
		},
		{
			name:   "a value that only some parsers read as the new one is rewritten",
			text:   "ESCAPED=a$$b\nREF=${DB_HOST}\nSLASH=\"a\\\\b\"\n",
			values: map[string]string{"ESCAPED": "a$b", "REF": "${DB_HOST}", "SLASH": `a\b`},
			want:   "ESCAPED='a$b'\nREF='${DB_HOST}'\nSLASH='a\\b'\n",
		},
		{
			name:   "spaces around equals",
			text:   "KEY = old\n",
			values: map[string]string{"KEY": "new"},
			want:   "KEY = new\n",
		},
		{
			name:   "keeps trailing comment",
			text:   "A=old # unquoted comment\nB=\"old\" # quoted comment\n",
			values: map[string]string{"A": "new", "B": "new"},
			want:   "A=new # unquoted comment\nB=new # quoted comment\n",
		},
		{
			name:   "replaces multi-line value whole",
			text:   "BEFORE=1\nKEY=\"line1\nline2\nline3\"\nAFTER=2\n",
			values: map[string]string{"KEY": "single"},
			want:   "BEFORE=1\nKEY=single\nAFTER=2\n",
		},
		{
			name:   "escaped quote inside the old value",
			text:   "KEY=\"a \\\" b\"\nAFTER=2\n",
			values: map[string]string{"KEY": "new"},
			want:   "KEY=new\nAFTER=2\n",
		},
		{
			name:   "single-quoted old value",
			text:   "KEY='old # not comment'\n",
			values: map[string]string{"KEY": "new"},
			want:   "KEY=new\n",
		},
		{
			name:   "uncomments only the first commented assignment",
			text:   "# DB_HOST=127.0.0.1\n#DB_HOST=other\n",
			values: map[string]string{"DB_HOST": "db"},
			want:   "DB_HOST=db\n#DB_HOST=other\n",
		},
		{
			name:   "active assignment wins over commented one",
			text:   "# KEY=example\nKEY=old\n",
			values: map[string]string{"KEY": "new"},
			want:   "# KEY=example\nKEY=new\n",
		},
		{
			name:   "keeps export prefix and appends with it",
			text:   "export APP_NAME=\"old\"\n# export DEBUG=false\n",
			values: map[string]string{"APP_NAME": "app", "DEBUG": "true", "NEW_ONE": "x"},
			want:   "export APP_NAME=app\nexport DEBUG=true\nexport NEW_ONE=x\n",
		},
		{
			name:   "CRLF file stays CRLF",
			text:   "A=old\r\nB=keep\r\n",
			values: map[string]string{"A": "new", "C": "c"},
			want:   "A=new\r\nB=keep\r\nC=c\r\n",
		},
		{
			name:   "CRLF commented line",
			text:   "# A=old\r\n",
			values: map[string]string{"A": "new"},
			want:   "A=new\r\n",
		},
		{
			name:   "dotted name",
			text:   "database.default.hostname = localhost\n",
			values: map[string]string{"database.default.hostname": "db"},
			want:   "database.default.hostname = db\n",
		},
		{
			name:   "template and dollar characters are written literally",
			text:   "OTHER={{ not a template }}\n",
			values: map[string]string{"KEY": "{{ .X }} $Y $$"},
			want:   "OTHER={{ not a template }}\nKEY='{{ .X }} $Y $$'\n",
		},
		{
			name:   "empty old value",
			text:   "KEY=\nNEXT=1\n",
			values: map[string]string{"KEY": "v"},
			want:   "KEY=v\nNEXT=1\n",
		},
		{
			name:   "UTF-8 BOM before the first assignment",
			text:   "\xef\xbb\xbfKEY=old\n",
			values: map[string]string{"KEY": "new"},
			want:   "\xef\xbb\xbfKEY=new\n",
		},
		{
			name:   "UTF-8 BOM before a commented assignment",
			text:   "\xef\xbb\xbf# KEY=old\n",
			values: map[string]string{"KEY": "new"},
			want:   "\xef\xbb\xbfKEY=new\n",
		},
		{
			name:   "colon separator",
			text:   "KEY: old\n",
			values: map[string]string{"KEY": "new"},
			want:   "KEY: new\n",
		},
		{
			name:   "prose comment with a colon is not an assignment",
			text:   "# KEY: describes the key\n",
			values: map[string]string{"KEY": "new"},
			want:   "# KEY: describes the key\nKEY=new\n",
		},
		{
			name:   "lines inside a multi-line value are not assignments",
			text:   "TEXT=\"first\nKEY=inside\n# OTHER=inside\"\nKEY=real\n",
			values: map[string]string{"KEY": "new", "OTHER": "o"},
			want:   "TEXT=\"first\nKEY=inside\n# OTHER=inside\"\nKEY=new\nOTHER=o\n",
		},
		{
			name:   "unterminated quoted value is replaced up to the line end",
			text:   "KEY=\"unterminated\nNEXT=1\n",
			values: map[string]string{"KEY": "fixed"},
			want:   "KEY=fixed\nNEXT=1\n",
		},
		{
			name:   "indented assignment and tab before value",
			text:   "  KEY=\told\n",
			values: map[string]string{"KEY": "new"},
			want:   "  KEY=\tnew\n",
		},
		{
			name:   "unquoted value with # but no space before it",
			text:   "KEY=a#b\n",
			values: map[string]string{"KEY": "new"},
			want:   "KEY=new\n",
		},
		{
			name:   "name that is a prefix of another is matched exactly",
			text:   "KEY_LONG=1\nKEY=2\n",
			values: map[string]string{"KEY": "3"},
			want:   "KEY_LONG=1\nKEY=3\n",
		},
		{
			name:   "no values leaves text unchanged",
			text:   "A=1\r\n# comment",
			values: nil,
			want:   "A=1\r\n# comment",
		},
		{
			name:   "duplicate active assignments are all updated",
			text:   "KEY=1\nKEY=2\n",
			values: map[string]string{"KEY": "3"},
			want:   "KEY=3\nKEY=3\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dotenv.Update(tc.text, tc.values)
			require.Equal(t, tc.want, got)
			parsed, err := dotenv.Parse(".env", got, nil)
			require.NoError(t, err, "updated text must parse")
			for k, v := range tc.values {
				require.Equal(t, v, parsed[k], "value of %s", k)
			}
		})
	}
}

func TestParseLiteral(t *testing.T) {
	t.Setenv("DOTENV_TEST_VAR", "from-env")
	text := `UNQUOTED=asd$asd
DOUBLE="asd$asd"
SINGLE='asd$asd'
BRACED=${DOTENV_TEST_VAR:-default}
ESCAPED="a\$b"
DOUBLED=$$
ESCAPES="say \"hi\"\tthere"
COMMENT=a$b # $not part of the value
SPACED = $x
COLON: $y
export EXPORTED=$z
MULTI="line1
$DOTENV_TEST_VAR line2"
# COMMENTED=$q
`
	got, err := dotenv.ParseLiteral(".env", text)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"UNQUOTED": "asd$asd",
		"DOUBLE":   "asd$asd",
		"SINGLE":   "asd$asd",
		"BRACED":   "${DOTENV_TEST_VAR:-default}",
		"ESCAPED":  "a$b",
		"DOUBLED":  "$$",
		"ESCAPES":  "say \"hi\"\tthere",
		"COMMENT":  "a$b",
		"SPACED":   "$x",
		"COLON":    "$y",
		"EXPORTED": "$z",
		"MULTI":    "line1\n$DOTENV_TEST_VAR line2",
	}, got)

	_, err = dotenv.ParseLiteral(".env", `BROKEN="unterminated`+"\n")
	require.ErrorContains(t, err, ".env")
}

func TestUpdateRoundTripTricky(t *testing.T) {
	text := dotenv.Update("", trickyValues)
	// Updating again must replace each value in place, multi-line ones included
	text = dotenv.Update(text, trickyValues)
	got, err := dotenv.Parse(".env", text, nil)
	require.NoError(t, err)
	require.Equal(t, trickyValues, got)
}
