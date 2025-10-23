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

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		walkFilepaths = append(walkFilepaths, s)
	}
	return nil
}

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

	fmt.Println("Matched")
	os.Exit(0)
}

type Pattern struct {
	Pattern  string
	Min      int
	Max      int
	Multiple bool
	Optional bool
	Matched  int
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
					targetNumberMax, _ := strconv.Atoi(splits[1])

					patterns[len(patterns)-1].Min = targetNumberMin
					patterns[len(patterns)-1].Max = targetNumberMax
				} else {
					targetNumberMin, _ := strconv.Atoi(splits[0])
					patterns[len(patterns)-1].Min = targetNumberMin
					patterns[len(patterns)-1].Multiple = true
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

		extras = make(map[int]Lazy)
		ok, _, _ := tryPatterns(line, patterns, groups)
		if ok {
			return true
		}
		for key := len(extras); key >= 0; key-- {
			for {
				if extras[key].Max-1 < 1 {
					break
				}
				extras[key] = Lazy{Max: extras[key].Max - 1, Current: 0, Limit: extras[key].Max - 1}
				patterns = splitPatterns(pattern)
				ok, _, _ := tryPatterns(line, patterns, groups)
				if ok {
					return true
				}
			}
		}
	}
	return false
}

type Lazy struct {
	Max     int
	Limit   int
	Current int
}

var extras map[int]Lazy

func tryPatterns(line []byte, patterns []Pattern, groups []string) (bool, int, []string) {
	originalSize := len(line)
	for patternIndex, pattern := range patterns {
		if pattern.Pattern == "$" {
			if len(line) == 0 {
				return true, originalSize, groups
			} else {
				return false, -1, groups
			}
		}

		if len(line) == 0 {
			if pattern.Optional {
				continue
			}
			return false, -1, groups
		}

		size, foundGroups := matchPattern(pattern.Pattern, line, groups)
		groups = foundGroups
		if size == 0 {
			if pattern.Optional || (pattern.Min != -1 && pattern.Matched >= pattern.Min) {
				continue
			}
			return false, -1, groups
		}

		line = line[size:]

		if pattern.Min != -1 || pattern.Max != -1 {
			patterns[patternIndex].Matched = patterns[patternIndex].Matched + 1
			if patterns[patternIndex].Matched == patterns[patternIndex].Max {
				continue
			}
			ok, size, subGroups := tryPatterns(line, patterns[patternIndex:], groups)
			if ok {
				return true, originalSize - len(line) + size, subGroups
			}
			return false, -1, groups
		}

		if pattern.Multiple {
			limit := -1
			current := -1
			for index, value := range groups {
				if value == "XXX" {
					if val, ok := extras[index]; ok {
						limit = val.Limit
						val.Current++
						current = val.Current
						extras[index] = val
					}
					break
				}
			}
			if limit != -1 && current >= limit {
				continue
			}
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
		if (line[0] >= 'a' && line[0] <= 'z') || (line[0] >= 'A' && line[0] <= 'Z') || (line[0] >= '0' && line[0] <= '9') || line[0] == '_' {
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
				if _, ok := extras[len(groups)-1]; !ok {
					extras[len(groups)-1] = Lazy{Max: totalSize, Limit: -1}
				}
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
