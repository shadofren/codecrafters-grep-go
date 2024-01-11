package main

import (
	"fmt"
	"io"
	"os"
)

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

	ok, err := matchLine(line, pattern, false)
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

func matchLine(line []byte, pattern string, onlyStart bool) (bool, error) {

	// base case
	if len(pattern) == 0 {
		return true, nil
	}
	if len(line) == 0 {
		return false, nil
	}

	if onlyStart {
		// recursion
		switch pattern[0] {
		case '\\':
			if len(pattern) == 1 {
				return false, nil
			}
			if pattern[1] == 'd' {
				if isDigit(line[0]) {
					return matchLine(line[1:], pattern[2:], onlyStart)
				}
				return false, nil
			} else if pattern[1] == 'w' {
				if isAlphaNumeric(line[0]) {
					return matchLine(line[1:], pattern[2:], onlyStart)
				}
				return false, nil
			}
		case '+':
			return false, nil
		case '*':
			return false, nil
		default:
			if line[0] == pattern[0] {
				return matchLine(line[1:], pattern[1:], onlyStart)
			}
			return false, nil
		}
	} else {
		for i := 0; i < len(line); i++ {
			match, err := matchLine(line[i:], pattern, true)
			if match {
				return match, err
			}
		}
	}
	return false, nil
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlpha(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func isAlphaNumeric(ch byte) bool {
	return isDigit(ch) || isAlpha(ch) || ch == '_'
}
