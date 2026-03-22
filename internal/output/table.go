package output

import (
	"fmt"
	"strings"
)

// Table renders a simple aligned text table.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a table with the given headers.
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{headers: headers, widths: widths}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cols ...string) {
	for len(cols) < len(t.headers) {
		cols = append(cols, "")
	}
	for i, c := range cols {
		if i < len(t.widths) && len(c) > t.widths[i] {
			t.widths[i] = len(c)
		}
	}
	t.rows = append(t.rows, cols)
}

// String renders the table as a string.
func (t *Table) String() string {
	var b strings.Builder

	// Header
	for i, h := range t.headers {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%-*s", t.widths[i], h)
	}
	b.WriteString("\n")

	// Separator
	for i, w := range t.widths {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(strings.Repeat("─", w))
	}
	b.WriteString("\n")

	// Rows
	for _, row := range t.rows {
		for i, c := range row {
			if i >= len(t.widths) {
				break
			}
			if i > 0 {
				b.WriteString("  ")
			}
			fmt.Fprintf(&b, "%-*s", t.widths[i], c)
		}
		b.WriteString("\n")
	}

	return b.String()
}
