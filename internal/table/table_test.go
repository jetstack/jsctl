package table_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jetstack/jsctl/internal/table"
	"github.com/stretchr/testify/assert"
)

func TestBuilder_Build(t *testing.T) {
	tbl := table.NewBuilder([]string{
		"key", "value",
	})

	tbl.AddRow("a", 1)
	tbl.AddRow("b", 2)
	tbl.AddRow("c", 3)

	buffer := bytes.NewBuffer([]byte{})
	assert.NoError(t, tbl.Build(buffer))

	const expected = `
KEY    VALUE
a      1
b      2
c      3
`

	assert.EqualValues(t, strings.TrimPrefix(expected, "\n"), buffer.String())
}
