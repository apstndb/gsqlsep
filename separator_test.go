// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gsqlsep

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSeparatorSkipComments(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		wantRemained string
	}{
		{
			desc:         "single line comment (#)",
			str:          "# SELECT 1;\n",
			wantRemained: "",
		},
		{
			desc:         "single line comment (--)",
			str:          "-- SELECT 1;\n",
			wantRemained: "",
		},
		{
			desc:         "multiline comment",
			str:          "/* SELECT\n1; */",
			wantRemained: "",
		},
		{
			desc:         "single line comment (#) and statement",
			str:          "# SELECT 1;\nSELECT 2;",
			wantRemained: "SELECT 2;",
		},
		{
			desc:         "single line comment (--) and statement",
			str:          "-- SELECT 1;\nSELECT 2;",
			wantRemained: "SELECT 2;",
		},
		{
			desc:         "multiline comment and statement",
			str:          "/* SELECT\n1; */ SELECT 2;",
			wantRemained: " SELECT 2;",
		},
		{
			desc:         "single line comment (#) not terminated",
			str:          "# SELECT 1",
			wantRemained: "",
		},
		{
			desc:         "single line comment (--) not terminated",
			str:          "-- SELECT 1",
			wantRemained: "",
		},
		{
			desc:         "multiline comment not terminated",
			str:          "/* SELECT\n1;",
			wantRemained: "",
		},
		{
			desc:         "not comments",
			str:          "SELECT 1;",
			wantRemained: "SELECT 1;",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str, false, nil)
			s.skipComments()

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeComments(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "double quoted string",
			str:          `"test" WHERE`,
			want:         `"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "single quoted string",
			str:          `'test' WHERE`,
			want:         `'test'`,
			wantRemained: " WHERE",
		},
		{
			desc:         "tripled quoted string",
			str:          `"""test""" WHERE`,
			want:         `"""test"""`,
			wantRemained: " WHERE",
		},
		{
			desc:         "quoted string with escape sequence",
			str:          `"te\"st" WHERE`,
			want:         `"te\"st"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "double quoted empty string",
			str:          `"" WHERE`,
			want:         `""`,
			wantRemained: " WHERE",
		},
		{
			desc:         "tripled quoted string with new line",
			str:          "'''t\ne\ns\nt''' WHERE",
			want:         "'''t\ne\ns\nt'''",
			wantRemained: " WHERE",
		},
		{
			desc:         "triple quoted empty string",
			str:          `"""""" WHERE`,
			want:         `""""""`,
			wantRemained: " WHERE",
		},
		{
			desc:         "multi-byte character in string",
			str:          `"テスト" WHERE`,
			want:         `"テスト"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str, false, nil)
			s.consumeString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeRawString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "raw string (r)",
			str:          `r"test" WHERE`,
			want:         `r"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "raw string (R)",
			str:          `R'test' WHERE`,
			want:         `R'test'`,
			wantRemained: " WHERE",
		},
		{
			desc:         "raw string with escape sequence",
			str:          `r"test\abc" WHERE`,
			want:         `r"test\abc"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str, false, nil)
			s.consumeRawString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeRawString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeRawString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeBytesString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "bytes string (b)",
			str:          `b"test" WHERE`,
			want:         `b"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "bytes string (B)",
			str:          `B'test' WHERE`,
			want:         `B'test'`,
			wantRemained: " WHERE",
		},
		{
			desc:         "bytes string with hex escape",
			str:          `b"\x12\x34\x56" WHERE`,
			want:         `b"\x12\x34\x56"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str, false, nil)
			s.consumeBytesString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeBytesString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeBytesString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparatorConsumeRawBytesString(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		str          string
		want         string
		wantRemained string
	}{
		{
			desc:         "raw bytes string (rb)",
			str:          `rb"test" WHERE`,
			want:         `rb"test"`,
			wantRemained: " WHERE",
		},
		{
			desc:         "raw bytes string (RB)",
			str:          `RB"test" WHERE`,
			want:         `RB"test"`,
			wantRemained: " WHERE",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := newSeparator(tt.str, false, nil)
			s.consumeRawBytesString()

			got := s.sb.String()
			if got != tt.want {
				t.Errorf("consumeRawBytesString(%q) = %q, but want = %q", tt.str, got, tt.want)
			}

			remained := string(s.str)
			if remained != tt.wantRemained {
				t.Errorf("consumeRawBytesString(%q) remained %q, but want = %q", tt.str, remained, tt.wantRemained)
			}
		})
	}
}

func TestSeparateInput_SpannerCliCompatible(t *testing.T) {
	const (
		terminatorHorizontal = `;`
		terminatorVertical   = `\G`
		terminatorUndefined  = ``
	)
	for _, tt := range []struct {
		desc  string
		input string
		want  []InputStatement
	}{
		{
			desc:  "single query",
			input: `SELECT "123";`,
			want: []InputStatement{
				{
					Statement:  `SELECT "123"`,
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  "double queries",
			input: `SELECT "123"; SELECT "456";`,
			want: []InputStatement{
				{
					Statement:  `SELECT "123"`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `SELECT "456"`,
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  "quoted identifier",
			input: "SELECT `1`, `2`; SELECT `3`, `4`;",
			want: []InputStatement{
				{
					Statement:  "SELECT `1`, `2`",
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  "SELECT `3`, `4`",
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  "vertical terminator",
			input: `SELECT "123"\G`,
			want: []InputStatement{
				{
					Statement:  `SELECT "123"`,
					Terminator: terminatorVertical,
				},
			},
		},
		{
			desc:  "mixed terminator",
			input: `SELECT "123"; SELECT "456"\G SELECT "789";`,
			want: []InputStatement{
				{
					Statement:  `SELECT "123"`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `SELECT "456"`,
					Terminator: terminatorVertical,
				},
				{
					Statement:  `SELECT "789"`,
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  "sql query",
			input: `SELECT * FROM t1 WHERE id = "123" AND "456"; DELETE FROM t2 WHERE true;`,
			want: []InputStatement{
				{
					Statement:  `SELECT * FROM t1 WHERE id = "123" AND "456"`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `DELETE FROM t2 WHERE true`,
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  "second query is empty",
			input: `SELECT 1; ;`,
			want: []InputStatement{
				{
					Statement:  `SELECT 1`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  ``,
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  "new line just after terminator",
			input: "SELECT 1;\n SELECT 2\\G\n",
			want: []InputStatement{
				{
					Statement:  `SELECT 1`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `SELECT 2`,
					Terminator: terminatorVertical,
				},
			},
		},
		{
			desc:  "horizontal terminator in string",
			input: `SELECT "1;2;3"; SELECT 'TL;DR';`,
			want: []InputStatement{
				{
					Statement:  `SELECT "1;2;3"`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `SELECT 'TL;DR'`,
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  `vertical terminator in string`,
			input: `SELECT r"1\G2\G3"\G SELECT r'4\G5\G6'\G`,
			want: []InputStatement{
				{
					Statement:  `SELECT r"1\G2\G3"`,
					Terminator: terminatorVertical,
				},
				{
					Statement:  `SELECT r'4\G5\G6'`,
					Terminator: terminatorVertical,
				},
			},
		},
		{
			desc:  "terminator in quoted identifier",
			input: "SELECT `1;2`; SELECT `3;4`;",
			want: []InputStatement{
				{
					Statement:  "SELECT `1;2`",
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  "SELECT `3;4`",
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  `query has new line just before terminator`,
			input: "SELECT '123'\n; SELECT '456'\n\\G",
			want: []InputStatement{
				{
					Statement:  `SELECT '123'`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `SELECT '456'`,
					Terminator: terminatorVertical,
				},
			},
		},
		{
			desc:  `DDL`,
			input: "CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id);",
			want: []InputStatement{
				{
					Statement:  "CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id)",
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  `statement with multiple comments`,
			input: "# comment;\nSELECT /* comment */ 1; --comment\nSELECT 2;/* comment */",
			want: []InputStatement{
				{
					Statement:  "SELECT  1",
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  "SELECT 2",
					Terminator: terminatorHorizontal,
				},
			},
		},
		{
			desc:  `only comments`,
			input: "# comment;\n/* comment */--comment\n/* comment */",
			want:  nil,
		},
		{
			desc:  `second query ends in the middle of string`,
			input: `SELECT "123"; SELECT "45`,
			want: []InputStatement{
				{
					Statement:  `SELECT "123"`,
					Terminator: terminatorHorizontal,
				},
				{
					Statement:  `SELECT "45`,
					Terminator: terminatorUndefined,
				},
			},
		},
		{
			desc:  `totally incorrect query`,
			input: `a"""""""""'''''''''b`,
			want: []InputStatement{
				{
					Statement:  `a"""""""""'''''''''b`,
					Terminator: terminatorUndefined,
				},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := SeparateInput(tt.input, `\G`)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InputStatement{})); diff != "" {
				t.Errorf("difference in statements: (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSeparateInputString_SpannerCliCompatible(t *testing.T) {
	for _, tt := range []struct {
		desc  string
		input string
		want  []string
	}{
		{
			desc:  "single query",
			input: `SELECT "123";`,
			want: []string{
				`SELECT "123"`,
			},
		},
		{
			desc:  "double queries",
			input: `SELECT "123"; SELECT "456";`,
			want: []string{
				`SELECT "123"`,
				`SELECT "456"`,
			},
		},
		{
			desc:  "quoted identifier",
			input: "SELECT `1`, `2`; SELECT `3`, `4`;",
			want: []string{
				"SELECT `1`, `2`",
				"SELECT `3`, `4`",
			},
		},
		{
			desc:  "vertical terminator",
			input: `SELECT "123"\G`,
			want: []string{
				`SELECT "123"`,
			},
		},
		{
			desc:  "mixed terminator",
			input: `SELECT "123"; SELECT "456"\G SELECT "789";`,
			want: []string{
				`SELECT "123"`,
				`SELECT "456"`,
				`SELECT "789"`,
			},
		},
		{
			desc:  "sql query",
			input: `SELECT * FROM t1 WHERE id = "123" AND "456"; DELETE FROM t2 WHERE true;`,
			want: []string{
				`SELECT * FROM t1 WHERE id = "123" AND "456"`,
				`DELETE FROM t2 WHERE true`,
			},
		},
		{
			desc:  "second query is empty",
			input: `SELECT 1; ;`,
			want: []string{
				`SELECT 1`,
				``,
			},
		},
		{
			desc:  "new line just after terminator",
			input: "SELECT 1;\n SELECT 2\\G\n",
			want: []string{
				`SELECT 1`,
				`SELECT 2`,
			},
		},
		{
			desc:  "horizontal terminator in string",
			input: `SELECT "1;2;3"; SELECT 'TL;DR';`,
			want: []string{
				`SELECT "1;2;3"`,
				`SELECT 'TL;DR'`,
			},
		},
		{
			desc:  `vertical terminator in string`,
			input: `SELECT r"1\G2\G3"\G SELECT r'4\G5\G6'\G`,
			want: []string{
				`SELECT r"1\G2\G3"`,
				`SELECT r'4\G5\G6'`,
			},
		},
		{
			desc:  "terminator in quoted identifier",
			input: "SELECT `1;2`; SELECT `3;4`;",
			want: []string{
				"SELECT `1;2`",
				"SELECT `3;4`",
			},
		},
		{
			desc:  `query has new line just before terminator`,
			input: "SELECT '123'\n; SELECT '456'\n\\G",
			want: []string{
				`SELECT '123'`,
				`SELECT '456'`,
			},
		},
		{
			desc:  `DDL`,
			input: "CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id);",
			want: []string{
				"CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id)",
			},
		},
		{
			desc:  `statement with multiple comments`,
			input: "# comment;\nSELECT /* comment */ 1; --comment\nSELECT 2;/* comment */",
			want: []string{
				"SELECT  1",
				"SELECT 2",
			},
		},
		{
			desc:  `only comments`,
			input: "# comment;\n/* comment */--comment\n/* comment */",
			want:  nil,
		},
		{
			desc:  `second query ends in the middle of string`,
			input: `SELECT "123"; SELECT "45`,
			want: []string{
				`SELECT "123"`,
				`SELECT "45`,
			},
		},
		{
			desc:  `totally incorrect query`,
			input: `a"""""""""'''''''''b`,
			want: []string{
				`a"""""""""'''''''''b`,
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := SeparateInputString(tt.input, `\G`)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InputStatement{})); diff != "" {
				t.Errorf("difference in statements: (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSeparateInputStringWithComments(t *testing.T) {
	for _, tt := range []struct {
		desc  string
		input string
		want  []string
	}{
		{
			desc:  "single query",
			input: `SELECT "123";`,
			want: []string{
				`SELECT "123"`,
			},
		},
		{
			desc:  "double queries",
			input: `SELECT "123"; SELECT "456";`,
			want: []string{
				`SELECT "123"`,
				`SELECT "456"`,
			},
		},
		{
			desc:  "quoted identifier",
			input: "SELECT `1`, `2`; SELECT `3`, `4`;",
			want: []string{
				"SELECT `1`, `2`",
				"SELECT `3`, `4`",
			},
		},
		{
			desc:  "vertical terminator",
			input: `SELECT "123"\G`,
			want: []string{
				`SELECT "123"`,
			},
		},
		{
			desc:  "mixed terminator",
			input: `SELECT "123"; SELECT "456"\G SELECT "789";`,
			want: []string{
				`SELECT "123"`,
				`SELECT "456"`,
				`SELECT "789"`,
			},
		},
		{
			desc:  "sql query",
			input: `SELECT * FROM t1 WHERE id = "123" AND "456"; DELETE FROM t2 WHERE true;`,
			want: []string{
				`SELECT * FROM t1 WHERE id = "123" AND "456"`,
				`DELETE FROM t2 WHERE true`,
			},
		},
		{
			desc:  "second query is empty",
			input: `SELECT 1; ;`,
			want: []string{
				`SELECT 1`,
				``,
			},
		},
		{
			desc:  "new line just after terminator",
			input: "SELECT 1;\n SELECT 2\\G\n",
			want: []string{
				`SELECT 1`,
				`SELECT 2`,
			},
		},
		{
			desc:  "horizontal terminator in string",
			input: `SELECT "1;2;3"; SELECT 'TL;DR';`,
			want: []string{
				`SELECT "1;2;3"`,
				`SELECT 'TL;DR'`,
			},
		},
		{
			desc:  `vertical terminator in string`,
			input: `SELECT r"1\G2\G3"\G SELECT r'4\G5\G6'\G`,
			want: []string{
				`SELECT r"1\G2\G3"`,
				`SELECT r'4\G5\G6'`,
			},
		},
		{
			desc:  "terminator in quoted identifier",
			input: "SELECT `1;2`; SELECT `3;4`;",
			want: []string{
				"SELECT `1;2`",
				"SELECT `3;4`",
			},
		},
		{
			desc:  `query has new line just before terminator`,
			input: "SELECT '123'\n; SELECT '456'\n\\G",
			want: []string{
				`SELECT '123'`,
				`SELECT '456'`,
			},
		},
		{
			desc:  `DDL`,
			input: "CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id);",
			want: []string{
				"CREATE t1 (\nId INT64 NOT NULL\n) PRIMARY KEY (Id)",
			},
		},
		{
			desc:  `statement with multiple comments`,
			input: "# comment;\nSELECT /* comment */ 1; --comment\nSELECT 2;/* comment */",
			want: []string{
				"# comment;\nSELECT /* comment */ 1",
				"--comment\nSELECT 2",
				"/* comment */",
			},
		},
		{
			desc:  `only comments`,
			input: "# comment;\n/* comment */--comment\n/* comment */",
			want: []string{
				"# comment;\n/* comment */--comment\n/* comment */",
			},
		},
		{
			desc:  `second query ends in the middle of string`,
			input: `SELECT "123"; SELECT "45`,
			want: []string{
				`SELECT "123"`,
				`SELECT "45`,
			},
		},
		{
			desc:  `totally incorrect query`,
			input: `a"""""""""'''''''''b`,
			want: []string{
				`a"""""""""'''''''''b`,
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := SeparateInputStringPreserveComments(tt.input, `\G`)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(InputStatement{})); diff != "" {
				t.Errorf("difference in statements: (-want +got):\n%s", diff)
			}
		})
	}
}
