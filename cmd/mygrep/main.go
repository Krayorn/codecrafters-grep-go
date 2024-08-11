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

func matchPattern(line []byte, index int, pattern string) bool {

	for _, c := range line[index:] {
		if len(pattern) == 0 {
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

		if bytes.ContainsAny([]byte{c}, pattern) {
			continue
		}

		return false
	}
	return true
}

func matchLine(line []byte, pattern string) (bool, error) {
	for i := range line {
		if matchPattern(line, i, pattern) {
			return true, nil
		}
	}

	return false, nil
}
