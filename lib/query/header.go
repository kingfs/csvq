package query

import (
	"strconv"
	"strings"

	"github.com/mithrandie/csvq/lib/parser"
)

const InternalIdColumn = "@__internal_id"

type HeaderField struct {
	View         string
	Column       string
	Aliases      []string
	Number       int
	IsFromTable  bool
	IsJoinColumn bool
	IsGroupKey   bool
}

type Header []HeaderField

func NewDualHeader() Header {
	h := make([]HeaderField, 1)
	return h
}

func NewHeaderWithId(view string, words []string) Header {
	h := make([]HeaderField, len(words)+1)

	h[0].View = view
	h[0].Column = InternalIdColumn

	for i, v := range words {
		h[i+1].View = view
		h[i+1].Column = v
		h[i+1].Number = i + 1
		h[i+1].IsFromTable = true
	}

	return h
}

func NewHeader(view string, words []string) Header {
	h := make([]HeaderField, len(words))

	for i, v := range words {
		h[i].View = view
		h[i].Column = v
		h[i].Number = i + 1
		h[i].IsFromTable = true
	}

	return h
}

func NewHeaderWithAutofill(view string, words []string) Header {
	for i, v := range words {
		if v == "" {
			words[i] = "__@" + strconv.Itoa(i+1) + "__"
		}
	}
	return NewHeader(view, words)
}

func NewEmptyHeader(len int) Header {
	return make([]HeaderField, len)
}

func MergeHeader(h1 Header, h2 Header) Header {
	return append(h1, h2...)
}

func AddHeaderField(h Header, column string, alias string) (header Header, index int) {
	hfield := HeaderField{
		Column: column,
	}
	if 0 < len(alias) && !strings.EqualFold(column, alias) {
		hfield.Aliases = append(hfield.Aliases, alias)
	}

	header = append(h, hfield)
	index = header.Len() - 1
	return
}

func (h Header) Len() int {
	return len(h)
}

func (h Header) TableColumns() []parser.QueryExpression {
	columns := make([]parser.QueryExpression, 0)
	for _, f := range h {
		if !f.IsFromTable {
			continue
		}

		fieldRef := parser.FieldReference{
			Column: parser.Identifier{Literal: f.Column},
		}
		if 0 < len(f.View) {
			fieldRef.View = parser.Identifier{Literal: f.View}
		}

		columns = append(columns, fieldRef)
	}
	return columns
}

func (h Header) TableColumnNames() []string {
	names := make([]string, 0)
	for _, f := range h {
		if !f.IsFromTable {
			continue
		}
		names = append(names, f.Column)
	}
	return names
}

func (h Header) ContainsObject(obj parser.QueryExpression) (int, error) {
	if fref, ok := obj.(parser.FieldReference); ok {
		return h.Contains(fref)
	} else if cnum, ok := obj.(parser.ColumnNumber); ok {
		return h.ContainsNumber(cnum)
	}

	column := parser.FormatFieldIdentifier(obj)

	idx := -1

	for i, f := range h {
		if f.IsFromTable {
			continue
		}

		if !strings.EqualFold(f.Column, column) {
			continue
		}

		if -1 < idx {
			return -1, NewFieldAmbiguousError(obj)
		}
		idx = i
	}

	if idx < 0 {
		return -1, NewFieldNotExistError(obj)
	}
	return idx, nil
}

func (h Header) ContainsNumber(number parser.ColumnNumber) (int, error) {
	view := number.View.Literal
	idx := int(number.Number.Raw())

	if idx < 1 {
		return -1, NewFieldNotExistError(number)
	}

	for i, f := range h {
		if strings.EqualFold(f.View, view) && f.Number == idx {
			return i, nil
		}
	}
	return -1, NewFieldNotExistError(number)
}

func (h Header) Contains(fieldRef parser.FieldReference) (int, error) {
	var view string
	if 0 < len(fieldRef.View.Literal) {
		view = fieldRef.View.Literal
	}
	column := fieldRef.Column.Literal

	idx := -1

	for i, f := range h {
		if 0 < len(view) {
			if !strings.EqualFold(f.View, view) || !strings.EqualFold(f.Column, column) {
				continue
			}
		} else {
			isEqual := strings.EqualFold(f.Column, column)
			if isEqual && f.IsJoinColumn {
				idx = i
				break
			}

			if !isEqual && !InStrSliceWithCaseInsensitive(column, f.Aliases) {
				continue
			}
		}

		if -1 < idx {
			return -1, NewFieldAmbiguousError(fieldRef)
		}
		idx = i
	}

	if idx < 0 {
		return -1, NewFieldNotExistError(fieldRef)
	}

	return idx, nil
}

func (h Header) ContainsInternalId(viewName string) (int, error) {
	fieldRef := parser.FieldReference{
		View:   parser.Identifier{Literal: viewName},
		Column: parser.Identifier{Literal: InternalIdColumn},
	}
	return h.Contains(fieldRef)
}

func (h Header) Update(reference string, fields []parser.QueryExpression) error {
	if fields != nil {
		if len(fields) != h.Len() {
			return NewFieldLengthNotMatchError()
		}

		names := make([]string, len(fields))
		for i, v := range fields {
			f, _ := v.(parser.Identifier)
			if InStrSliceWithCaseInsensitive(f.Literal, names) {
				return NewDuplicateFieldNameError(f)
			}
			names[i] = f.Literal
		}
	}

	for i := range h {
		h[i].View = reference
		if fields != nil {
			h[i].Column = fields[i].(parser.Identifier).Literal
		}
		h[i].Aliases = nil
	}
	return nil
}

func (h Header) Copy() Header {
	header := make(Header, h.Len())
	copy(header, h)
	return header

}
