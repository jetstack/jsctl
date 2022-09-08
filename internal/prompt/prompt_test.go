package prompt_test

import (
	"bytes"
	"testing"

	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/stretchr/testify/assert"
)

func TestYesNo(t *testing.T) {
	t.Parallel()

	t.Run("It should return true on yes", func(t *testing.T) {
		input := bytes.NewBufferString("y\n")
		output := bytes.NewBuffer([]byte{})

		ok, err := prompt.YesNo(input, output, "you sure about that %s?", "bob")
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("It should false true on no", func(t *testing.T) {
		input := bytes.NewBufferString("n\n")
		output := bytes.NewBuffer([]byte{})

		ok, err := prompt.YesNo(input, output, "you sure about that %s?", "bob")
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("It should false true on any other input", func(t *testing.T) {
		input := bytes.NewBufferString("egg\n")
		output := bytes.NewBuffer([]byte{})

		ok, err := prompt.YesNo(input, output, "you sure about that %s?", "bob")
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestSuggest(t *testing.T) {
	t.Parallel()

	t.Run("It should output a single suggestion message", func(t *testing.T) {
		const expected = "1 suggested action(s):\n\n1) A small suggestion\n"
		output := bytes.NewBuffer([]byte{})

		assert.NoError(t, prompt.Suggest(output,
			prompt.NewSuggestion(
				prompt.WithMessage("A %s suggestion", "small"),
			),
		))

		assert.EqualValues(t, expected, output.String())
	})

	t.Run("It should output multiple suggestion messages", func(t *testing.T) {
		const expected = "2 suggested action(s):\n\n1) A small suggestion\n\n2) A medium suggestion\n"
		output := bytes.NewBuffer([]byte{})

		assert.NoError(t, prompt.Suggest(output,
			prompt.NewSuggestion(
				prompt.WithMessage("A %s suggestion", "small"),
			),
			prompt.NewSuggestion(
				prompt.WithMessage("A %s suggestion", "medium"),
			),
		))

		assert.EqualValues(t, expected, output.String())
	})

	t.Run("It should output a suggestion with a command to run", func(t *testing.T) {
		const expected = "1 suggested action(s):\n\n1) A small suggestion\n\trm -rf /\n"
		output := bytes.NewBuffer([]byte{})

		assert.NoError(t, prompt.Suggest(output,
			prompt.NewSuggestion(
				prompt.WithMessage("A %s suggestion", "small"),
				prompt.WithCommand("rm -rf /"),
			),
		))

		assert.EqualValues(t, expected, output.String())
	})

	t.Run("It should output a suggestion with a link", func(t *testing.T) {
		const expected = "1 suggested action(s):\n\n1) A small suggestion\n\thttps://google.com\n"
		output := bytes.NewBuffer([]byte{})

		assert.NoError(t, prompt.Suggest(output,
			prompt.NewSuggestion(
				prompt.WithMessage("A %s suggestion", "small"),
				prompt.WithLink("https://google.com"),
			),
		))

		assert.EqualValues(t, expected, output.String())
	})
}
