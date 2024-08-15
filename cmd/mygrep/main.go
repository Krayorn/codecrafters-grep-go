package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Usage: echo <input_text> | your_program.sh -E <pattern>
func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]

	line, err := io.ReadAll(os.Stdin) // assume we're only dealing with a single line
	if err != nil {
		os.Exit(2)
	}

	ok := matchLine(line, pattern)
	if !ok {
		fmt.Println("Did not match")
		os.Exit(1)
	}

	fmt.Println("Matched")
}

func splitPatterns(pattern string) []string {
	patterns := make([]string, 0)

	for len(pattern) > 0 {
		switch pattern[0] {
		case '\\':
			patterns = append(patterns, pattern[:2])
			pattern = pattern[2:]
		case '[':
			end := strings.IndexByte(pattern, ']')
			patterns = append(patterns, pattern[:end+1])
			pattern = pattern[end+1:]
		case '(':
			end := -1
			c := 0
			for i := 0; i < len(pattern); i++ {
				if pattern[i] == '(' {
					c++
				}
				if pattern[i] == ')' {
					c--
				}
				if c == 0 {
					end = i
					break
				}
			}
			patterns = append(patterns, pattern[:end+1])
			pattern = pattern[end+1:]
		case '+', '?':
			patterns[len(patterns)-1] += string(pattern[0])
			pattern = pattern[1:]
		default:
			patterns = append(patterns, string(pattern[0]))
			pattern = pattern[1:]
		}
	}
	return patterns
}

func matchLine(text []byte, pattern string) bool {
	patterns := splitPatterns(pattern)
	if len(patterns) == 0 {
		return true
	}

	onlyFirst := false
	if patterns[0] == "^" {
		patterns = patterns[1:]
		onlyFirst = true
	}

	for i := range text {
		if onlyFirst && i > 0 {
			break
		}

		line := text[i:]
		groups := make([]string, 0)

		fmt.Println("Test", string(line))

		ok, _, _ := tryPatterns(line, patterns, groups)
		if ok {
			return true
		}
	}
	return false
}

func tryPatterns(line []byte, patterns []string, groups []string) (bool, int, []string) {
	originalSize := len(line)
	for patternIndex, pattern := range patterns {
		if pattern == "$" {
			if len(line) == 0 {
				return true, originalSize, groups
			} else {
				return false, -1, groups
			}
		}

		if len(line) == 0 {
			return false, -1, groups
		}
		size, foundGroups := matchPattern(pattern, line, groups)
		groups = foundGroups
		if size == 0 {
			if pattern[len(pattern)-1] == '?' {
				continue
			}
			return false, -1, groups
		}

		line = line[size:]
		if pattern[len(pattern)-1] == '+' {
			ok, size, subGroups := tryPatterns(line, patterns[patternIndex:], groups)
			if ok {
				return true, originalSize - len(line) + size, subGroups
			}
		}
	}

	return true, originalSize - len(line), groups
}

func matchPattern(pattern string, line []byte, groups []string) (int, []string) {
	if pattern[0] == '\\' && pattern[1] == 'd' {
		if line[0] >= '0' && line[0] <= '9' {
			return 1, groups
		}
	} else if pattern[0] == '\\' && pattern[1] == 'w' {
		if (line[0] >= 'a' && line[0] <= 'z') || (line[0] >= 'A' && line[0] <= 'Z') || (line[0] >= '0' && line[0] <= '9') {
			return 1, groups
		}
	} else if pattern[0] == '\\' && pattern[1] >= '0' && pattern[1] <= '9' {
		number, _ := strconv.Atoi(string(pattern[1]))
		if groups[number-1] == string(line[0:len(groups[number-1])]) {
			return len(groups[number-1]), groups
		}
	} else if pattern[0] == '[' {
		closingBrackets := bytes.IndexAny([]byte(pattern), "]")
		if pattern[1] == '^' {
			if !bytes.ContainsAny([]byte{line[0]}, pattern[1:closingBrackets]) {
				return 1, groups
			}
		} else {
			if bytes.ContainsAny([]byte{line[0]}, pattern[1:closingBrackets]) {
				return 1, groups
			}
		}
	} else if pattern[0] == '(' {
		c := 0
		rules := make([]string, 0)
		last := 1
		closingParenthesis := -1
		for j := 0; j < len(pattern); j++ {
			if pattern[j] == '(' {
				c++
			}
			if pattern[j] == ')' {
				c--
			}

			if pattern[j] == '|' && c == 1 {
				rules = append(rules, pattern[last:j])
				last = j + 1
			}
			if c == 0 {
				closingParenthesis = j
				break
			}
		}
		rules = append(rules, pattern[last:closingParenthesis])

		for _, rule := range rules {
			subPatterns := splitPatterns(rule)
			groups = append(groups, "XXX")
			ok, totalSize, subGroups := tryPatterns(line, subPatterns, groups)
			if ok {
				groups[len(groups)-1] = string(line[:totalSize])
				groups = append(groups, subGroups[len(groups):]...)
				return totalSize, groups
			}
		}

	} else {
		if pattern[0] == '.' {
			return 1, groups
		} else if bytes.ContainsAny([]byte{line[0]}, string(pattern[0])) {
			return 1, groups
		}
	}

	return 0, groups
}
