package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type TokenType byte

const (
	Literal TokenType = iota
	Digit
	AlphaNumeric
	LineStart
	LineEnd
	Blank
	PositiveGroup
	NegativeGroup
	WildCard
)

// parse the pattern into tokens
type Token struct {
	Type       TokenType
	InnerToken []*Token // can be a group
	Raw        string
	ZeroOrMore bool
}

func (t *Token) Print() {
	fmt.Printf("%s: ZeroOrMore = %v\n", t.Raw, t.ZeroOrMore)
}

type PatternMatcher struct {
	Data    []byte
	Pattern []*Token
}

// Usage: echo <input_text> | your_grep.sh -E <pattern>
func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]

	line, err := io.ReadAll(os.Stdin) // assume we're only dealing with a single line
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	if line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	tokens := parsePattern(pattern)
	fmt.Printf("parsed %d tokens\n", len(tokens))
	for _, tok := range tokens {
		tok.Print()
	}

	p := &PatternMatcher{Data: line, Pattern: tokens}

	ok, err := p.Match(0, 0, false)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !ok {
		fmt.Println("not ok")
		os.Exit(1)
	}
	fmt.Println("ok")

	// default exit code is 0 which means success
}

func parsePattern(pattern string) []*Token {
	tokens := make([]*Token, 0)
	i := 0
	n := len(pattern)
	for i < n {
		if i == 0 && pattern[i] == '^' {
			tokens = append(tokens, &Token{Type: LineStart, Raw: "^"})
		} else if i == n-1 && pattern[i] == '$' {
			tokens = append(tokens, &Token{Type: LineEnd, Raw: "$"})
		} else {
			switch pattern[i] {
			case '[':
				endIdx := i + strings.Index(pattern[i:], "]")
				var token *Token
				if pattern[i+1] == '^' {
					token = &Token{Type: NegativeGroup, Raw: pattern[i : endIdx+1]}
					i++
				} else {
					token = &Token{Type: PositiveGroup, Raw: pattern[i : endIdx+1]}
				}
				for _, c := range pattern[i+1 : endIdx] {
					inner := &Token{Type: Literal, Raw: string(c)}
					token.InnerToken = append(token.InnerToken, inner)
				}
				i = endIdx
				tokens = append(tokens, token)
			case '+':
				last := tokens[len(tokens)-1]
				// have a must match entry, and then a ZeroOrMore entry to represent one or more entry
				tokens = append(tokens, &Token{last.Type, last.InnerToken, last.Raw, true})
			case '*':
				last := tokens[len(tokens)-1]
				last.ZeroOrMore = true
			case '\\':
				nxt := pattern[i+1]
				if nxt == 'w' {
					tokens = append(tokens, &Token{Type: AlphaNumeric, Raw: pattern[i : i+2]})
				} else if nxt == 'd' {
					tokens = append(tokens, &Token{Type: Digit, Raw: pattern[i : i+2]})
				}
				i++
			default:
				tokens = append(tokens, &Token{Type: Literal, Raw: pattern[i : i+1]})
			}
		}
		i++
	}
	return tokens
}

func (p *PatternMatcher) Match(dIdx, pIdx int, start bool) (bool, error) {
	if pIdx == len(p.Pattern) {
		// no more token
		return true, nil
	}
	// Line end
	if pIdx == len(p.Pattern)-1 && p.Pattern[pIdx].Type == LineEnd {
		return dIdx == len(p.Data), nil
	}
	// Line start
	if pIdx == 0 && p.Pattern[pIdx].Type == LineStart {
		if dIdx != 0 {
			return false, nil
		}
		return p.Match(0, 1, true)
	}
	if dIdx == len(p.Data) {
		return false, nil
	}

	var err error
	if start {
		// normal cases
		tok := p.Pattern[pIdx]
    match, _ := p.MatchSingleToken(p.Data[dIdx], tok)
		if tok.ZeroOrMore {
      if match {
        // consume the character, keep the token
        ok, _ := p.Match(dIdx + 1, pIdx, true)
        if ok {
          return ok, nil
        }
        // keep the character, consume the token
        return p.Match(dIdx, pIdx+1, true)
      } else {
        // no match, have to skip this token
        return p.Match(dIdx, pIdx+1, true)
      }
		} else {
      if match {
        return p.Match(dIdx+1, pIdx+1, true)
      }
      return false, nil
    }
	} else {
		for i := 0; i < len(p.Data); i++ {
			match, err := p.Match(i, pIdx, true)
			if match {
				return match, err
			}
		}
	}

	return false, err
}

func (p *PatternMatcher) MatchSingleToken(ch byte, tok *Token) (bool, error) {
	switch tok.Type {
	case Literal:
		return ch == tok.Raw[0], nil
	case Digit:
		return isDigit(ch), nil
	case AlphaNumeric:
		return isAlphaNumeric(ch), nil
	case PositiveGroup:
		// match each of the underlying Token
		for _, innerTok := range tok.InnerToken {
			// assume all are literal for now
			match, err := p.MatchSingleToken(ch, innerTok)
			if match {
				return match, err
			}
		}
    return false, nil
	case NegativeGroup:
		for _, innerTok := range tok.InnerToken {
			match, _ := p.MatchSingleToken(ch, innerTok)
			if match {
				return false, nil
			}
		}
    return true, nil
	default:
		return false, nil
	}
}

/* func matchLine(line []byte, pattern []*Token, start bool) (bool, error) { */
/**/
/* 	fmt.Println("matching", line, pattern, start) */
/* 	// base case */
/* 	if len(pattern) == 0 { */
/* 		return true, nil */
/* 	} */
/* 	if pattern[0] == '^' { */
/* 		return matchLine(line, pattern[1:], true) */
/* 	} */
/* 	if pattern[0] == '$' { */
/* 		if len(line) != 0 { */
/* 			return false, nil */
/* 		} */
/* 		return true, nil */
/* 	} */
/* 	if len(line) == 0 { */
/* 		return false, nil */
/* 	} */
/**/
/* 	if start { */
/* 		// recursion */
/* 		var match bool */
/* 		nextPos := 1 */
/* 	sw: */
/* 		switch pattern[0] { */
/* 		case '\\': */
/* 			if len(pattern) == 1 { */
/* 				break */
/* 			} */
/* 			if pattern[1] == 'd' { */
/* 				if isDigit(line[0]) { */
/* 					match = true */
/* 				} */
/* 			} else if pattern[1] == 'w' { */
/* 				if isAlphaNumeric(line[0]) { */
/* 					match = true */
/* 				} */
/* 			} */
/* 			nextPos = 2 */
/* 		case '[': // character group */
/* 			fmt.Println("character group") */
/* 			endIdx := strings.Index(pattern, "]") */
/* 			nextPos = endIdx + 1 */
/* 			// check for positive / negative character group */
/* 			if pattern[1] == '^' { */
/* 				fmt.Println("checking negative", line, pattern) */
/* 				for i := 2; i < endIdx; i++ { */
/* 					if line[0] == pattern[i] { */
/* 						break sw */
/* 					} */
/* 				} */
/* 			} else { */
/* 				fmt.Println("checking positive", line, pattern) */
/* 				for i := 1; i < endIdx; i++ { */
/* 					if line[0] == pattern[i] { */
/* 						match = true */
/* 						break sw */
/* 					} */
/* 				} */
/* 			} */
/* 		default: */
/* 			if line[0] == pattern[0] { */
/* 				match = true */
/* 			} */
/* 		} */
/* 		if match { */
/* 			return matchLine(line[1:], pattern[nextPos:], true) */
/* 		} */
/* 		return false, nil */
/* 	} else { */
/* 		for i := 0; i < len(line); i++ { */
/* 			match, err := matchLine(line[i:], pattern, true) */
/* 			if match { */
/* 				return match, err */
/* 			} */
/* 		} */
/* 	} */
/* 	return false, nil */
/* } */
/**/

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlpha(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func isAlphaNumeric(ch byte) bool {
	return isDigit(ch) || isAlpha(ch) || ch == '_'
}
