package printer

import (
	"io"
	"reflect"

	tableprinter "github.com/landoop/tableprinter"

	"github.com/pkg/errors"
)

func NewTableWriter(out io.Writer) *tableprinter.Printer {
	return tableprinter.New(out)
}

func PrintTable(out io.Writer, v interface{}, getRow func(row interface{}) []string, headers ...string) error {
	if reflect.TypeOf(v).Kind() != reflect.Slice {
		return errors.Errorf("invalid data passed to PrintTable, must be a slice but got %T", v)
	}
	rows := reflect.ValueOf(v)

	// headers := tableprinter.StructParser.ParseHeaders(rows)

	table := NewTableWriter(out)
	if len(headers) > 0 {
		table.Render(headers, nil, nil, false)
	}
	for i := 0; i < rows.Len(); i++ {
		table.Render(getRow(rows.Index(i).Interface()), nil, nil, false)
	}
	return nil
}
