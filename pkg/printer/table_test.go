package printer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

type testType struct {
	A, B interface{}
}

func TestPrintTable(t *testing.T) {
	v := []testType{
		{A: "foo", B: "bar"},
		{A: "baz", B: "qux"},
		{A: 123, B: true},
	}

	b := &bytes.Buffer{}

	err := PrintTable(b, v, func(r interface{}) []string {
		row, ok := r.(testType)
		require.True(t, ok)
		return []string{row.A.(string), row.B.(string)}
	},
		"A", "B")

	require.Nil(t, err)
	require.Equal(t, "A     B\nfoo   bar\nbaz   qux\n123   true\n", b.String())
}

func TestPrintTable_WithoutHeaders(t *testing.T) {
	v := []testType{
		{A: "foo", B: "bar"},
	}

	b := &bytes.Buffer{}

	err := PrintTable(b, v, func(r interface{}) []string {
		row, ok := r.(testType)
		require.True(t, ok)
		return []string{row.A.(string), row.B.(string)}
	})

	require.Nil(t, err)
	require.Equal(t, "foo   bar\n", b.String())
}
