package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
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
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	ok, err := matchLine(line, pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !ok {
		os.Exit(1)
	}
}

func matchLine(line []byte, pattern string) (bool, error) {
	var ok bool
	if pattern == "\\d" {
		ok = bytes.ContainsAny(line, "0123456789")
	} else if pattern == "\\w" {
		ok = bytes.ContainsFunc(line, func(c rune) bool {
			return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		})
	} else if strings.HasPrefix(pattern, "[") && strings.HasSuffix(pattern, "]") {
		if strings.HasPrefix(pattern, "[^") {
			for _, c := range line {
				ok = bytes.ContainsAny([]byte{c}, pattern[2:len(pattern)-1])
				if !ok {
					return true, nil
				}
			}
			ok = false
		} else {
			ok = bytes.ContainsAny(line, pattern[1:len(pattern)-1])
		}
	} else {
		ok = bytes.ContainsAny(line, pattern)
	}

	return ok, nil
}
