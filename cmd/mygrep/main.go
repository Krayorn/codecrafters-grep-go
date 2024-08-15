package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
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
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	ok, err := matchLine(line, pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !ok {
		fmt.Println("Did not match")
		os.Exit(1)
	}
	fmt.Println("Matched")
}

func matchPattern(line []byte, index int, pattern string, onlyLast bool) (bool, int) {
	sizeToCut := 1
	matchedPreviously := false
	line = line[index:]
	i := 0

	groups := make(map[int]string)
	for i = 0; i < len(line); i++ {
		c := line[i]
		matched := false
		if len(pattern) == 0 {
			if onlyLast {
				return false, -1
			}
			//fmt.Println("Matched on", string(line))
			return true, i
		}

		if pattern[0] == '\\' && pattern[1] == 'd' {
			sizeToCut = 2
			if c >= '0' && c <= '9' {
				matched = true
			}
		} else if pattern[0] == '\\' && pattern[1] == 'w' {
			sizeToCut = 2
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				matched = true
			}
		} else if pattern[0] == '\\' && pattern[1] >= '0' && pattern[1] <= '9' {
			sizeToCut = 2
			number, _ := strconv.Atoi(string(pattern[1]))
			fmt.Println(groups[number], string(line[i:i+len(groups[number])]))
			if groups[number] == string(line[i:i+len(groups[number])]) {
				i += len(groups[number]) - 1
				matched = true
			}
		} else if pattern[0] == '(' {
			closingParenthesis := -1
			c := 1
			last := 1
			rules := make([]string, 0)
			for j := 1; j < len(pattern); j++ {
				if pattern[j] == '(' {
					c++
				}
				if pattern[j] == ')' {
					c--
				}

				if pattern[j] == '|' {
					if c == 1 {
						rules = append(rules, pattern[last:j])
						last = j + 1
					}
				}
				if c == 0 {
					closingParenthesis = j
					break
				}
			}

			rules = append(rules, pattern[last:closingParenthesis])
			sizeToCut = closingParenthesis + 1
			for _, rule := range rules {
				ok, size := matchPattern(line[i:], 0, rule, false)
				if ok {
					groups[1] = string(line[i : i+size])
					fmt.Println("groups", groups)
					i += size - 1
					matched = true
					break
				}
			}
		} else if pattern[0] == '[' {
			closingBrackets := bytes.IndexAny([]byte(pattern), "]")
			sizeToCut = closingBrackets + 1
			if pattern[1] == '^' {
				if !bytes.ContainsAny([]byte{c}, pattern[1:closingBrackets]) {
					matched = true
				}
			} else {
				if bytes.ContainsAny([]byte{c}, pattern[1:closingBrackets]) {
					matched = true
				}
			}
		} else {
			sizeToCut = 1
			if pattern[0] == '.' {
				matched = true
			} else if bytes.ContainsAny([]byte{c}, string(pattern[0])) {
				matched = true
			}
		}

		fmt.Println("Checking at", string(c), pattern, matched, matchedPreviously)

		if sizeToCut < len(pattern) && pattern[sizeToCut] == '?' {
			sizeToCut++
			if !matched {
				i--
			}
			matched = true
		}

		if matched {
			if sizeToCut < len(pattern) && pattern[sizeToCut] == '+' {
				matchedPreviously = true
			} else {
				matchedPreviously = false
				pattern = pattern[sizeToCut:]
			}
			continue
		}

		if matchedPreviously {
			i-- // retry with next patterng
			matchedPreviously = false
			sizeToCut++
			pattern = pattern[sizeToCut:]
			continue
		}

		return false, -1
	}

	for len(pattern) > 0 {
		if pattern[0] == '\\' && pattern[1] == 'd' {
			sizeToCut = 2
		} else if pattern[0] == '\\' && pattern[1] == 'w' {
			sizeToCut = 2
		} else if pattern[0] == '[' {
			closingBrackets := bytes.IndexAny([]byte(pattern), "]")
			sizeToCut = closingBrackets + 1
		} else {
			sizeToCut = 1
		}

		if sizeToCut < len(pattern) && pattern[sizeToCut] == '?' {
			sizeToCut++
			pattern = pattern[sizeToCut:]
			matchedPreviously = false
			continue
		}
		if sizeToCut < len(pattern) && pattern[sizeToCut] == '+' && matchedPreviously {
			sizeToCut++
			pattern = pattern[sizeToCut:]
			matchedPreviously = false
			continue
		}

		break
	}

	return len(pattern) <= 0, i
}

func matchLine(line []byte, pattern string) (bool, error) {
	options := line
	onlyLast := false

	if pattern[0] == '^' {
		options = line[0:1]
		pattern = pattern[1:]
	}

	if pattern[len(pattern)-1] == '$' {
		onlyLast = true
		pattern = pattern[:len(pattern)-1]
	}

	for i := range options {
		//fmt.Println("Trying", string(line[i:]), pattern)
		if ok, _ := matchPattern(line, i, pattern, onlyLast); ok {
			return true, nil
		}
	}

	return false, nil
}
