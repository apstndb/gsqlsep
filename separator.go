//
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
// derived from:
//   github.com/cloudspannerecosystem/spanner-cli/separator.go
//   github.com/googleapis/google-cloud-go/spanner/spansql/parser.go

package gsqlsep

import (
	"strings"

	"golang.org/x/exp/slices"
)

type InputStatement struct {
	Statement  string
	Terminator string
}

// SeparateInput separates input for each statement and returns []InputStatement.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInput(input string, customTerminators ...string) []InputStatement {
	return newSeparator(input, customTerminators).separate()
}

// SeparateInputString separates input for each statement and returns []string.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInputString(input string, customTerminators ...string) []string {
	var result []string
	for _, s := range SeparateInput(input, customTerminators...) {
		result = append(result, s.Statement)
	}
	return result
}

type separator struct {
	str []rune // remaining input
	sb  *strings.Builder
	// terms is custom terminators.
	// It isn't []string to minimize string-rune conversions.
	terms [][]rune
}

func newSeparator(s string, terms []string) *separator {
	var runeTerms [][]rune
	for _, term := range terms {
		runeTerms = append(runeTerms, []rune(term))
	}
	return &separator{
		str:   []rune(s),
		sb:    &strings.Builder{},
		terms: runeTerms,
	}
}

func (s *separator) consumeRawString() {
	// consume 'r' or 'R'
	s.sb.WriteRune(s.str[0])
	s.str = s.str[1:]

	delim := s.consumeStringDelimiter()
	s.consumeStringContent(delim, true)
}

func (s *separator) consumeBytesString() {
	// consume 'b' or 'B'
	s.sb.WriteRune(s.str[0])
	s.str = s.str[1:]

	delim := s.consumeStringDelimiter()
	s.consumeStringContent(delim, false)
}

func (s *separator) consumeRawBytesString() {
	// consume 'rb', 'Rb', 'rB', or 'RB'
	s.sb.WriteRune(s.str[0])
	s.sb.WriteRune(s.str[1])
	s.str = s.str[2:]

	delim := s.consumeStringDelimiter()
	s.consumeStringContent(delim, true)
}

func (s *separator) consumeString() {
	delim := s.consumeStringDelimiter()
	s.consumeStringContent(delim, false)
}

func (s *separator) consumeStringContent(delim string, raw bool) {
	var i int
	for i < len(s.str) {
		// check end of string
		switch {
		// check single-quoted delim
		case len(delim) == 1 && string(s.str[i]) == delim:
			s.str = s.str[i+1:]
			s.sb.WriteString(delim)
			return
		// check triple-quoted delim
		case len(delim) == 3 && len(s.str) >= i+3 && string(s.str[i:i+3]) == delim:
			s.str = s.str[i+3:]
			s.sb.WriteString(delim)
			return
		}

		// escape sequence
		if s.str[i] == '\\' {
			if raw {
				// raw string treats escape character as backslash
				s.sb.WriteRune('\\')
				i++
				continue
			}

			// invalid escape sequence
			if i+1 >= len(s.str) {
				s.sb.WriteRune('\\')
				return
			}

			s.sb.WriteRune('\\')
			s.sb.WriteRune(s.str[i+1])
			i += 2
			continue
		}
		s.sb.WriteRune(s.str[i])
		i++
	}
	s.str = s.str[i:]
	return
}

func (s *separator) consumeStringDelimiter() string {
	c := s.str[0]
	// check triple-quoted delim
	if len(s.str) >= 3 && s.str[1] == c && s.str[2] == c {
		delim := strings.Repeat(string(c), 3)
		s.sb.WriteString(delim)
		s.str = s.str[3:]
		return delim
	}
	s.str = s.str[1:]
	s.sb.WriteRune(c)
	return string(c)
}

func (s *separator) skipComments() {
	var i int
	for i < len(s.str) {
		var terminate string
		if s.str[i] == '#' {
			// single line comment "#"
			terminate = "\n"
			i++
		} else if i+1 < len(s.str) && s.str[i] == '-' && s.str[i+1] == '-' {
			// single line comment "--"
			terminate = "\n"
			i += 2
		} else if i+1 < len(s.str) && s.str[i] == '/' && s.str[i+1] == '*' {
			// multi line comments "/* */"
			// NOTE: Nested multiline comments are not supported in Spanner.
			// https://cloud.google.com/spanner/docs/lexical#multiline_comments
			terminate = "*/"
			i += 2
		}

		// no comment found
		if terminate == "" {
			return
		}

		// not terminated, but end of string
		if i >= len(s.str) {
			s.str = s.str[len(s.str):]
			return
		}

		for ; i < len(s.str); i++ {
			if l := len(terminate); l == 1 {
				if string(s.str[i]) == terminate {
					s.str = s.str[i+1:]
					i = 0
					break
				}
			} else if l == 2 {
				if i+1 < len(s.str) && string(s.str[i:i+2]) == terminate {
					s.str = s.str[i+2:]
					i = 0
					break
				}
			}
		}

		// not terminated, but end of string
		if i >= len(s.str) {
			s.str = s.str[len(s.str):]
			return
		}
	}
}

// separate separates input string into multiple Spanner statements.
// This does not validate syntax of statements.
//
// NOTE: Logic for parsing a statement is mostly taken from spansql.
// https://github.com/googleapis/google-cloud-go/blob/master/spanner/spansql/parser.go
func (s *separator) separate() []InputStatement {
	var statements []InputStatement
	for len(s.str) > 0 {
		s.skipComments()
		if len(s.str) == 0 {
			break
		}

		switch s.str[0] {
		// possibly string literal
		case '"', '\'', 'r', 'R', 'b', 'B':
			// valid string prefix: "b", "B", "r", "R", "br", "bR", "Br", "BR"
			// https://cloud.google.com/spanner/docs/lexical#string_and_bytes_literals
			raw, bytes, str := false, false, false
			for i := 0; i < 3 && i < len(s.str); i++ {
				switch {
				case !raw && (s.str[i] == 'r' || s.str[i] == 'R'):
					raw = true
					continue
				case !bytes && (s.str[i] == 'b' || s.str[i] == 'B'):
					bytes = true
					continue
				case s.str[i] == '"' || s.str[i] == '\'':
					str = true
					switch {
					case raw && bytes:
						s.consumeRawBytesString()
					case raw:
						s.consumeRawString()
					case bytes:
						s.consumeBytesString()
					default:
						s.consumeString()
					}
				}
				break
			}
			if !str {
				s.sb.WriteRune(s.str[0])
				s.str = s.str[1:]
			}
		// quoted identifier
		case '`':
			s.sb.WriteRune(s.str[0])
			s.str = s.str[1:]
			s.consumeStringContent("`", false)
		// horizontal delim
		case ';':
			statements = append(statements, InputStatement{
				Statement:  strings.TrimSpace(s.sb.String()),
				Terminator: ";",
			})
			s.sb.Reset()
			s.str = s.str[1:]
		default:
			// TODO: may need some optimization
			var found bool
			for _, term := range s.terms {
				if hasPrefix(s.str, term) {
					statements = append(statements, InputStatement{
						Statement:  strings.TrimSpace(s.sb.String()),
						Terminator: string(term),
					})
					s.sb.Reset()
					s.str = s.str[len(term):]
					found = true
					break
				}
			}

			if !found {
				s.sb.WriteRune(s.str[0])
				s.str = s.str[1:]
			}
		}
	}

	// flush remained
	if s.sb.Len() > 0 {
		if str := strings.TrimSpace(s.sb.String()); len(str) > 0 {
			statements = append(statements, InputStatement{
				Statement:  str,
				Terminator: "",
			})
			s.sb.Reset()
		}
	}
	return statements
}

func hasPrefix(s, prefix []rune) bool {
	return len(s) >= len(prefix) && slices.Equal(s[0:len(prefix)], prefix)
}
