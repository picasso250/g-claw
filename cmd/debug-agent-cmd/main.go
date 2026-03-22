package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func main() {
	raw := os.Getenv("AGENT_CMD")
	if strings.TrimSpace(raw) == "" {
		fmt.Println("AGENT_CMD is empty")
		os.Exit(1)
	}

	parts, err := splitCommandLine(raw)
	if err != nil {
		fmt.Printf("split error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("raw: %s\n", raw)
	fmt.Printf("count: %d\n", len(parts))
	for i, part := range parts {
		fmt.Printf("[%d] %s\n", i, part)
	}
}

func splitCommandLine(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuote := rune(0)

	for _, r := range strings.TrimSpace(command) {
		switch {
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '"' || r == '\'':
			inQuote = r
		case r == ' ' || r == '\t':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if inQuote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	if len(args) == 0 {
		return nil, errors.New("empty command")
	}

	return args, nil
}
