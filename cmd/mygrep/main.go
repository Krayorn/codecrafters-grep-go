package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var walkFilepaths []string

func main() {
	recursive := false
	pattern := ""
	filenames := make([]string, 0)

	i := 1
	for len(os.Args) > i {
		arg := os.Args[i]
		if arg == "-r" {
			recursive = true
		} else if arg == "-E" {
			i++
			if len(os.Args) < i-1 {
				os.Exit(2)
			}
			pattern = os.Args[i]
		} else {
			filenames = append(filenames, arg)
		}
		i++
	}

	if pattern == "" {
		os.Exit(2)
	}

	if len(filenames) > 0 {
		anyMatch := false

		for _, filename := range filenames {
			filepaths := make([]string, 0)

			fi, _ := os.Stat(filename)
			switch mode := fi.Mode(); {
			case mode.IsDir():
				if !recursive {
					os.Exit(2)
				}
				walkFilepaths = make([]string, 0)
				filepath.WalkDir(filename, walk)
				filepaths = walkFilepaths
			case mode.IsRegular():
				filepaths = append(filepaths, filename)
			}

			for _, filepath := range filepaths {
				file, _ := os.Open(filepath)
				scanner := bufio.NewScanner(file)

				for scanner.Scan() {
					line := scanner.Text()
					ok := matchLine([]byte(line), pattern)
					if ok {
						anyMatch = true
						if len(filenames) == 1 && !recursive {
							fmt.Printf("%s\n", line)
						} else {
							fmt.Printf("%s:%s\n", filepath, line)
						}
					}
				}
			}

		}

		if anyMatch {
			os.Exit(0)
		}
		os.Exit(1)
	}

	line, err := io.ReadAll(os.Stdin) // assume we're only dealing with a single line
	if err != nil {
		os.Exit(2)
	}

	ok := matchLine(line, pattern)
	if !ok {
		os.Exit(1)
	}

	os.Exit(0)
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		walkFilepaths = append(walkFilepaths, s)
	}
	return nil
}

type Pattern struct {
	Pattern  string
	Min      int
	Max      int
	Multiple bool
	Optional bool

	Matched int // reset between each try, used to follow how much a token consumed already
}

func splitPatterns(pattern string) []Pattern {
	patterns := make([]Pattern, 0)

	for len(pattern) > 0 {
		currentPattern := Pattern{Min: -1, Max: -1}
		switch pattern[0] {
		case '\\':
			currentPattern.Pattern = pattern[:2]
			patterns = append(patterns, currentPattern)
			pattern = pattern[2:]
		case '[':
			end := strings.IndexByte(pattern, ']')
			currentPattern.Pattern = pattern[:end+1]
			patterns = append(patterns, currentPattern)
			pattern = pattern[end+1:]
		case '{':
			end := strings.IndexByte(pattern, '}')
			if strings.Contains(string(pattern[1:end]), ",") {
				splits := strings.Split(pattern[1:end], ",")
				if len(splits) > 1 {
					targetNumberMin, _ := strconv.Atoi(splits[0])
					targetNumberMax, err := strconv.Atoi(splits[1])
					if err != nil {
						patterns[len(patterns)-1].Multiple = true

					} else {
						patterns[len(patterns)-1].Max = targetNumberMax
					}

					patterns[len(patterns)-1].Min = targetNumberMin
				}
			} else {
				targetNumber, _ := strconv.Atoi(string(pattern[1]))
				patterns[len(patterns)-1].Max = targetNumber
				patterns[len(patterns)-1].Min = targetNumber
			}
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
			currentPattern.Pattern = pattern[:end+1]
			patterns = append(patterns, currentPattern)

			pattern = pattern[end+1:]
		case '+':
			patterns[len(patterns)-1].Multiple = true
			pattern = pattern[1:]
		case '?':
			patterns[len(patterns)-1].Optional = true
			pattern = pattern[1:]
		case '*':
			patterns[len(patterns)-1].Multiple = true
			patterns[len(patterns)-1].Optional = true
			pattern = pattern[1:]
		default:
			currentPattern.Pattern = string(pattern[0])
			patterns = append(patterns, currentPattern)
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
	if patterns[0].Pattern == "^" {
		onlyFirst = true
	}

	for i := range text {
		if onlyFirst && i > 0 {
			break
		}
		// because we store info in the patterns now, we need to reset them between tries
		patterns = splitPatterns(pattern)
		if patterns[0].Pattern == "^" {
			patterns = patterns[1:]
		}

		line := text[i:]
		groups := make([]string, 0)

		results := tryPatterns(line, patterns, groups)
		if len(results) > 0 {
			return true
		}
	}
	return false
}

type Item struct {
	Line     []byte
	Patterns []Pattern
	Groups   []string
}

type Result struct {
	Size   int
	Groups []string
}

func tryPatterns(line []byte, patterns []Pattern, groups []string) []Result {
	originalSize := len(line)

	queue := make([]Item, 0)
	queue = append(queue, Item{Line: line, Patterns: patterns, Groups: groups})

	results := make([]Result, 0)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if len(item.Patterns) == 0 {
			results = append(results, Result{originalSize - len(item.Line), item.Groups})
			continue
		}

		currentPattern := item.Patterns[0]
		line := item.Line
		groups := item.Groups

		if currentPattern.Pattern == "$" {
			if len(line) == 0 {
				results = append(results, Result{originalSize - len(item.Line), item.Groups})
				continue
			} else {
				continue
			}
		}

		if len(line) == 0 {
			if currentPattern.Optional {
				queue = append(queue, Item{line, item.Patterns[1:], item.Groups})
			}
			continue
		}

		matches := make([]Item, 0)

		if currentPattern.Pattern[0] == '\\' && currentPattern.Pattern[1] == 'd' {
			if line[0] >= '0' && line[0] <= '9' {
				matches = append(matches, Item{line[1:], item.Patterns, item.Groups})
			}
		} else if currentPattern.Pattern[0] == '\\' && currentPattern.Pattern[1] == 'w' {
			if (line[0] >= 'a' && line[0] <= 'z') || (line[0] >= 'A' && line[0] <= 'Z') || (line[0] >= '0' && line[0] <= '9') || line[0] == '_' {
				matches = append(matches, Item{line[1:], item.Patterns, item.Groups})
			}
		} else if currentPattern.Pattern[0] == '\\' && currentPattern.Pattern[1] >= '0' && currentPattern.Pattern[1] <= '9' {
			number, _ := strconv.Atoi(string(currentPattern.Pattern[1]))
			if groups[number-1] == string(line[0:len(groups[number-1])]) {
				matches = append(matches, Item{line[len(groups[number-1]):], item.Patterns, item.Groups})
			}
		} else if currentPattern.Pattern[0] == '[' {
			closingBrackets := bytes.IndexAny([]byte(currentPattern.Pattern), "]")
			if currentPattern.Pattern[1] == '^' {
				if !bytes.ContainsAny([]byte{line[0]}, currentPattern.Pattern[1:closingBrackets]) {
					matches = append(matches, Item{line[1:], item.Patterns, item.Groups})
				}
			} else {
				if bytes.ContainsAny([]byte{line[0]}, currentPattern.Pattern[1:closingBrackets]) {
					matches = append(matches, Item{line[1:], item.Patterns, item.Groups})
				}
			}
		} else if currentPattern.Pattern[0] == '(' {
			c := 0
			rules := make([]string, 0)
			last := 1
			closingParenthesis := -1
			for j := 0; j < len(currentPattern.Pattern); j++ {
				if currentPattern.Pattern[j] == '(' {
					c++
				}
				if currentPattern.Pattern[j] == ')' {
					c--
				}

				if currentPattern.Pattern[j] == '|' && c == 1 {
					rules = append(rules, currentPattern.Pattern[last:j])
					last = j + 1
				}
				if c == 0 {
					closingParenthesis = j
					break
				}
			}
			rules = append(rules, currentPattern.Pattern[last:closingParenthesis])

			for _, rule := range rules {
				subPatterns := splitPatterns(rule)
				groups = append(groups, "XXX")
				subResults := tryPatterns(line, subPatterns, groups)
				if len(subResults) > 0 {
					for _, subResult := range subResults {
						groups[len(groups)-1] = string(line[:subResult.Size])

						matches = append(matches, Item{Line: line[subResult.Size:], Patterns: item.Patterns, Groups: append(groups, subResult.Groups[len(groups):]...)})
					}
				}
			}
		} else {
			if currentPattern.Pattern[0] == '.' {
				matches = append(matches, Item{line[1:], item.Patterns, item.Groups})
			} else if bytes.ContainsAny([]byte{line[0]}, string(currentPattern.Pattern[0])) {
				matches = append(matches, Item{line[1:], item.Patterns, item.Groups})
			}
		}

		if len(matches) > 0 {
			for _, match := range matches {
				if currentPattern.Min != -1 {
					match.Patterns[0].Matched = match.Patterns[0].Matched + 1
				}
				if currentPattern.Multiple {
					queue = append(queue, Item{Line: match.Line, Patterns: match.Patterns, Groups: match.Groups})
				}

				if currentPattern.Max != 1 && match.Patterns[0].Matched < currentPattern.Max {
					queue = append(queue, Item{Line: match.Line, Patterns: match.Patterns, Groups: match.Groups})
				}

				if currentPattern.Min == -1 || match.Patterns[0].Matched >= currentPattern.Min {
					queue = append(queue, Item{Line: match.Line, Patterns: match.Patterns[1:], Groups: match.Groups})
				}
			}
		} else {
			if currentPattern.Optional {
				queue = append(queue, Item{Line: item.Line, Patterns: item.Patterns[1:], Groups: item.Groups})
			}
		}
	}

	return results
}
