// Package prompt provides functions for prompting decisions via the CLI.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// YesNo writes a formatted message to output and waits for user input with either a y or n character. Any input other
// than a y will return false.
func YesNo(input io.Reader, output io.Writer, format string, args ...interface{}) (bool, error) {
	message := fmt.Sprintf(format, args...) + " (y/N): "

	fmt.Fprint(output, message)
	response, err := bufio.NewReader(input).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	if strings.TrimSpace(strings.ToLower(response)) != "y" {
		return false, nil
	}

	return true, nil
}

type (
	// The Suggestion type represents a recommended action a user should take. It can either be a message but can
	// also include a suggested command.
	Suggestion struct {
		Message string
		Command string
		Link    string
	}

	// The SuggestionOption is a function that modifies the fields of a Suggestion.
	SuggestionOption func(a *Suggestion)
)

// Suggest zero or more actions to be taken by the user, writing the output to the provided io.Writer implementation.
func Suggest(output io.Writer, suggestions ...Suggestion) error {
	if _, err := fmt.Fprintf(output, "%v suggested action(s):\n", len(suggestions)); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	for i, suggestion := range suggestions {
		if _, err := fmt.Fprintf(output, "\n%v) %s\n", i+1, suggestion.Message); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}

		if suggestion.Command != "" {
			if _, err := fmt.Fprintf(output, "\t%s\n", suggestion.Command); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
		}

		if suggestion.Link != "" {
			if _, err := fmt.Fprintf(output, "\t%s\n", suggestion.Link); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
		}
	}

	return nil
}

// NewSuggestion creates a new Suggestion type and applies all SuggestionOption functions to it.
func NewSuggestion(opts ...SuggestionOption) Suggestion {
	var a Suggestion
	for _, opt := range opts {
		opt(&a)
	}
	return a
}

// WithMessage sets the formatted message of a Suggestion.
func WithMessage(format string, args ...interface{}) SuggestionOption {
	return func(a *Suggestion) {
		a.Message = fmt.Sprintf(format, args...)
	}
}

// WithCommand sets the formatted command of a Suggestion.
func WithCommand(format string, args ...interface{}) SuggestionOption {
	return func(a *Suggestion) {
		a.Command = fmt.Sprintf(format, args...)
	}
}

// WithLink sets the link of a Suggestion for a user to navigate to.
func WithLink(url string) SuggestionOption {
	return func(a *Suggestion) {
		a.Link = url
	}
}
