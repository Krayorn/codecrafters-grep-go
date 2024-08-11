package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
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

func matchPattern(line []byte, index int, pattern string, onlyLast bool) bool {
	for _, c := range line[index:] {
		if len(pattern) == 0 {
			if onlyLast {
				return false
			}
			fmt.Println("Matched on", string(line[index:]))
			return true
		}

		if pattern[0] == '\\' && pattern[1] == 'd' {
			if c >= '0' && c <= '9' {
				pattern = pattern[2:]
				continue
			}
		}

		if pattern[0] == '\\' && pattern[1] == 'w' {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				pattern = pattern[2:]
				continue
			}
		}

		if pattern[0] == '[' {
			closingBrackets := bytes.IndexAny([]byte(pattern), "]")
			if pattern[1] == '^' {
				if !bytes.ContainsAny([]byte{c}, pattern[1:closingBrackets]) {
					pattern = pattern[closingBrackets+1:]
					continue
				}
				break
			}

			if bytes.ContainsAny([]byte{c}, pattern[1:closingBrackets]) {
				pattern = pattern[closingBrackets+1:]
				continue
			}
		}

		if bytes.ContainsAny([]byte{c}, string(pattern[0])) {
			pattern = pattern[1:]
			continue
		}

		return false
	}

	return len(pattern) <= 0
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
		fmt.Println("Trying", string(line[i:]), pattern)
		if matchPattern(line, i, pattern, onlyLast) {
			return true, nil
		}
	}

	return false, nil
}
