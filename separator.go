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

type Status struct {
	WaitingString string
}

func (stmt *InputStatement) StripComments() InputStatement {
	result := SeparateInputString(stmt.Statement)
	if len(result) == 0 {
		return InputStatement{
			Statement:  "",
			Terminator: stmt.Terminator,
		}
	}

	// It can assume InputStatement.Statement doesn't have any terminating characters.
	return InputStatement{
		Statement:  result[0],
		Terminator: stmt.Terminator,
	}
}

// SeparateInput separates input for each statement and returns []InputStatement.
// This function strip all comments in input.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInput(input string, customTerminators ...string) []InputStatement {
	stmts, _ := newSeparator(input, false, customTerminators).separate()
	return stmts
}

// SeparateInputString separates input for each statement and returns []string.
// This function strip all comments in input.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInputString(input string, customTerminators ...string) []string {
	var result []string
	for _, s := range SeparateInput(input, customTerminators...) {
		result = append(result, s.Statement)
	}
	return result
}

// SeparateInputPreserveComments separates input for each statement and returns []InputStatement.
// This function preserve comments in input.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInputPreserveComments(input string, customTerminators ...string) []InputStatement {
	stmts, _ := newSeparator(input, true, customTerminators).separate()
	return stmts
}

// SeparateInputPreserveCommentsWithStatus separates input for each statement and returns []InputStatement and Status.
// This function preserve comments in input.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInputPreserveCommentsWithStatus(input string, customTerminators ...string) ([]InputStatement, Status) {
	stmts, currentDelimiter := newSeparator(input, true, customTerminators).separate()
	return stmts, Status{WaitingString: currentDelimiter}
}

// SeparateInputStringPreserveComments separates input for each statement and returns []string.
// This function preserve comments in input.
// By default, input will be separated by terminating semicolons `;`.
// In addition, customTerminators can be passed, and they will be treated as terminating semicolons.
func SeparateInputStringPreserveComments(input string, customTerminators ...string) []string {
	var result []string
	for _, s := range SeparateInputPreserveComments(input, customTerminators...) {
		result = append(result, s.Statement)
	}
	return result
}

type separator struct {
	str []rune // remaining input
	sb  *strings.Builder
	// terms is custom terminators.
	// It isn't []string to minimize string-rune conversions.
	terms            [][]rune
	preserveComments bool
	currentDelimiter string
}

func newSeparator(s string, preserveComment bool, terms []string) *separator {
	var runeTerms [][]rune
	for _, term := range terms {
		runeTerms = append(runeTerms, []rune(term))
	}
	return &separator{
		str:              []rune(s),
		sb:               &strings.Builder{},
		terms:            runeTerms,
		preserveComments: preserveComment,
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
		if hasStringPrefix(s.str[i:], delim) {
			s.str = s.str[i+len(delim):]
			s.sb.WriteString(delim)
			s.currentDelimiter = ""
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
				s.currentDelimiter = delim
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
	s.currentDelimiter = delim
	return
}

func (s *separator) consumeStringDelimiter() string {
	c := s.str[0]
	// check triple-quoted delim
	if delim := strings.Repeat(string(c), 3); hasStringPrefix(s.str, delim) {
		s.sb.WriteString(delim)
		s.str = s.str[len(delim):]
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
		if prefix := "#"; hasStringPrefix(s.str, prefix) {
			// single line comment "#"
			terminate = "\n"
			i += len(prefix)
		} else if prefix := "--"; hasStringPrefix(s.str, prefix) {
			// single line comment "--"
			terminate = "\n"
			i += len(prefix)
		} else if prefix := "/*"; hasStringPrefix(s.str, prefix) {
			// multi line comments "/* */"
			// NOTE: Nested multiline comments are not supported in Spanner.
			// https://cloud.google.com/spanner/docs/lexical#multiline_comments
			terminate = "*/"
			s.currentDelimiter = terminate
			i += len(prefix)
		} else {
			// out of comment
			return
		}

		// not terminated, but end of string
		if lenStr := len(s.str); i >= lenStr {
			if s.preserveComments {
				s.sb.WriteString(string(s.str))
			}
			s.str = s.str[lenStr:]
			return
		}

		for ; i < len(s.str); i++ {
			if lenT := len(terminate); hasStringPrefix(s.str[i:], terminate) {
				if s.preserveComments {
					s.sb.WriteString(string(s.str[:i+lenT]))
				} else {
					// always replace a comment to a single whitespace.
					s.sb.WriteRune(' ')
				}
				s.str = s.str[i+lenT:]
				i = 0
				s.currentDelimiter = ""
				break
			}
		}

		// not terminated, but end of string
		if lenStr := len(s.str); i >= lenStr {
			if s.preserveComments {
				s.sb.WriteString(string(s.str))
			}
			s.str = s.str[lenStr:]
			return
		}
	}
}

// separate separates input string into multiple Spanner statements.
// This does not validate syntax of statements.
//
// NOTE: Logic for parsing a statement is mostly taken from spansql.
// https://github.com/googleapis/google-cloud-go/blob/master/spanner/spansql/parser.go
func (s *separator) separate() ([]InputStatement, string) {
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
	return statements, s.currentDelimiter
}

func hasPrefix(s, prefix []rune) bool {
	return len(s) >= len(prefix) && slices.Equal(s[0:len(prefix)], prefix)
}

func hasStringPrefix(s []rune, prefix string) bool {
	return hasPrefix(s, []rune(prefix))
}
