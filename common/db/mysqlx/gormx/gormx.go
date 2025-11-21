package gormx

import (
	"fmt"
	"strings"
	"time"

	"github.com/gfa-inc/gfa/core"
	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

type Pagination struct {
	Current  *int `form:"current,default=1"`
	PageSize *int `form:"pageSize,default=20"`
}

// ToOffsetAndLimit converts the current page and page size to offset and limit for database queries.
func (p *Pagination) ToOffsetAndLimit() (int, int) {
	offset := (*p.Current - 1) * (*p.PageSize)
	limit := *p.PageSize

	p.Current = nil
	p.PageSize = nil

	return offset, limit
}

type Dao interface {
	TableName() string
	GetFieldByName(fieldName string) (field.OrderExpr, bool)
}

type Sorter struct {
	Field string `form:"sorterField"`
	Order string `form:"sorterOrder"`
}

func (s *Sorter) ToOrderExpr(dao Dao, defaultSorter *Sorter) (field.Expr, error) {
	if s.Field == "" && defaultSorter != nil {
		s.Field = defaultSorter.Field
		s.Order = defaultSorter.Order
	}

	f, ok := dao.GetFieldByName(s.Field)
	if !ok {
		err := core.NewParamErr(fmt.Errorf("field %s not found in table %s", s.Field, dao.TableName()))
		return nil, err
	}
	s.Field = ""

	order := s.Order
	s.Order = ""

	if order == "ascend" {
		return f.Asc(), nil
	} else if order == "descend" {
		return f.Desc(), nil
	} else {
		return f.Desc(), nil
	}
}

func NewDescendSorter(field string) *Sorter {
	return &Sorter{
		Field: field,
		Order: "descend",
	}
}

func NewAscendSorter(field string) *Sorter {
	return &Sorter{
		Field: field,
		Order: "ascend",
	}
}

type DateTimeRange struct {
	StartTime string `form:"start_time" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	EndTime   string `form:"end_time" binding:"omitempty,datetime=2006-01-02 15:04:05"`
}

func (s *DateTimeRange) ToExpr(field field.Time) field.Expr {
	if s.StartTime == "" || s.EndTime == "" {
		return nil
	}

	startTime, _ := time.Parse(time.DateTime, s.StartTime)
	endTime, _ := time.Parse(time.DateTime, s.EndTime)

	s.StartTime = ""
	s.EndTime = ""
	return field.Between(startTime, endTime)
}

// ColumnType defines the data type of a column
type ColumnType string

const (
	TypeString   ColumnType = "string"
	TypeNumeric  ColumnType = "numeric"
	TypeUUID     ColumnType = "uuid"
	TypeDateTime ColumnType = "datetime"
	TypeJSON     ColumnType = "json"
	TypeBoolean  ColumnType = "boolean"
)

// TypedColumn defines a column with its type information
type TypedColumn struct {
	Name              string
	Type              ColumnType
	CustomEmptyValues []string
	AllowZero         bool
	AllowNegative     bool
}

// NumericColumn creates a numeric type column (checks: NULL, empty string, 0)
func NumericColumn(name string) TypedColumn {
	return TypedColumn{
		Name:          name,
		Type:          TypeNumeric,
		AllowZero:     false,
		AllowNegative: true,
	}
}

// NumericColumnAllowZero creates a numeric type column that allows zero value
func NumericColumnAllowZero(name string) TypedColumn {
	return TypedColumn{
		Name:          name,
		Type:          TypeNumeric,
		AllowZero:     true,
		AllowNegative: true,
	}
}

// NumericColumnPositive creates a numeric type column that only allows positive numbers
func NumericColumnPositive(name string) TypedColumn {
	return TypedColumn{
		Name:          name,
		Type:          TypeNumeric,
		AllowZero:     false,
		AllowNegative: false,
	}
}

// UUIDColumn creates a UUID type column (checks: NULL, empty string, all-zero UUID)
func UUIDColumn(name string) TypedColumn {
	return TypedColumn{Name: name, Type: TypeUUID}
}

// DateTimeColumn creates a datetime type column (checks: NULL, empty string, zero date)
func DateTimeColumn(name string) TypedColumn {
	return TypedColumn{Name: name, Type: TypeDateTime}
}

// JSONColumn creates a JSON type column (checks: NULL, empty string, '{}', '[]', 'null')
func JSONColumn(name string) TypedColumn {
	return TypedColumn{Name: name, Type: TypeJSON}
}

// StringColumn creates a string type column with optional custom empty values
func StringColumn(name string, customEmptyValues ...string) TypedColumn {
	return TypedColumn{
		Name:              name,
		Type:              TypeString,
		CustomEmptyValues: customEmptyValues,
	}
}

// BooleanColumn creates a boolean type column (checks: NULL, empty string)
func BooleanColumn(name string) TypedColumn {
	return TypedColumn{Name: name, Type: TypeBoolean}
}

// AssignmentNotEmptyColumns creates assignment clauses that only update non-empty values
//
// Supports three input methods (can be mixed):
//
//  1. Plain string (default to string type):
//     "name", "email", "address"
//
//  2. Prefix notation (concise):
//     "numeric:age"           -> numeric type (no zero)
//     "numeric+zero:score"    -> numeric type (allow zero)
//     "numeric+positive:qty"  -> numeric type (positive only)
//     "uuid:domain_id"        -> UUID type
//     "datetime:created_at"   -> datetime type
//     "json:metadata"         -> JSON type
//     "string:status:N/A:null" -> string type with custom empty values
//
//  3. TypedColumn (explicit, most clear):
//     NumericColumn("age")
//     UUIDColumn("domain_id")
//
// Example:
//
//	AssignmentNotEmptyColumns(
//	    "name",                      // plain string
//	    "numeric:age",               // prefix notation
//	    NumericColumn("balance"),    // typed column
//	)
func AssignmentNotEmptyColumns(columns ...any) clause.Set {
	typedColumns := make([]TypedColumn, len(columns))

	for i, col := range columns {
		switch v := col.(type) {
		case string:
			typedColumns[i] = parseColumnString(v)
		case TypedColumn:
			typedColumns[i] = v
		default:
			typedColumns[i] = TypedColumn{
				Name: fmt.Sprint(v),
				Type: TypeString,
			}
		}
	}

	return buildAssignments(typedColumns)
}

// parseColumnString parses column string with optional type prefix
func parseColumnString(s string) TypedColumn {
	if !strings.Contains(s, ":") {
		return TypedColumn{Name: s, Type: TypeString}
	}

	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return TypedColumn{Name: s, Type: TypeString}
	}

	typePrefix := parts[0]
	columnName := parts[1]

	switch {
	case typePrefix == "numeric" || typePrefix == "num" || typePrefix == "n":
		return TypedColumn{
			Name:          columnName,
			Type:          TypeNumeric,
			AllowZero:     false,
			AllowNegative: true,
		}

	case typePrefix == "numeric+zero" || typePrefix == "num+zero" || typePrefix == "n+z":
		return TypedColumn{
			Name:          columnName,
			Type:          TypeNumeric,
			AllowZero:     true,
			AllowNegative: true,
		}

	case typePrefix == "numeric+positive" || typePrefix == "num+positive" || typePrefix == "n+p":
		return TypedColumn{
			Name:          columnName,
			Type:          TypeNumeric,
			AllowZero:     false,
			AllowNegative: false,
		}

	case typePrefix == "uuid" || typePrefix == "u":
		return TypedColumn{Name: columnName, Type: TypeUUID}

	case typePrefix == "datetime" || typePrefix == "dt" || typePrefix == "d":
		return TypedColumn{Name: columnName, Type: TypeDateTime}

	case typePrefix == "json" || typePrefix == "j":
		return TypedColumn{Name: columnName, Type: TypeJSON}

	case typePrefix == "boolean" || typePrefix == "bool" || typePrefix == "b":
		return TypedColumn{Name: columnName, Type: TypeBoolean}

	case typePrefix == "string" || typePrefix == "str" || typePrefix == "s":
		customEmptyValues := parts[2:]
		return TypedColumn{
			Name:              columnName,
			Type:              TypeString,
			CustomEmptyValues: customEmptyValues,
		}

	default:
		return TypedColumn{Name: columnName, Type: TypeString}
	}
}

// buildAssignments constructs the assignment clauses
func buildAssignments(columns []TypedColumn) clause.Set {
	values := make(map[string]any)

	for _, col := range columns {
		conditions := buildEmptyConditions(col)

		if len(conditions) == 0 {
			values[col.Name] = field.NewUnsafeFieldRaw(fmt.Sprintf("VALUES(%s)", col.Name))
			continue
		}

		conditionsStr := strings.Join(conditions, " OR ")
		caseSQL := fmt.Sprintf(
			"CASE WHEN %s THEN %s ELSE VALUES(%s) END",
			conditionsStr, col.Name, col.Name)

		values[col.Name] = field.NewUnsafeFieldRaw(caseSQL)
	}

	return clause.Assignments(values)
}

// buildEmptyConditions builds empty value check conditions based on column type
func buildEmptyConditions(col TypedColumn) []string {
	var conditions []string

	// All types check NULL and empty string
	conditions = append(conditions, fmt.Sprintf("VALUES(%s) IS NULL", col.Name))
	conditions = append(conditions, fmt.Sprintf("TRIM(VALUES(%s)) = ''", col.Name))

	switch col.Type {
	case TypeNumeric:
		if !col.AllowZero {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) = 0", col.Name))
		}
		if !col.AllowNegative {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) < 0", col.Name))
		}

	case TypeUUID:
		conditions = append(conditions,
			fmt.Sprintf("VALUES(%s) = '00000000-0000-0000-0000-000000000000'", col.Name))

	case TypeDateTime:
		conditions = append(conditions, fmt.Sprintf("VALUES(%s) = '0000-00-00'", col.Name))
		conditions = append(conditions, fmt.Sprintf("VALUES(%s) = '0000-00-00 00:00:00'", col.Name))

	case TypeJSON:
		conditions = append(conditions, fmt.Sprintf("VALUES(%s) = '{}'", col.Name))
		conditions = append(conditions, fmt.Sprintf("VALUES(%s) = '[]'", col.Name))
		conditions = append(conditions, fmt.Sprintf("VALUES(%s) = 'null'", col.Name))
	}

	// Custom empty values for all types
	for _, emptyVal := range col.CustomEmptyValues {
		conditions = append(conditions,
			fmt.Sprintf("VALUES(%s) = '%s'", col.Name, escapeSQLString(emptyVal)))
	}

	return conditions
}

// escapeSQLString escapes single quotes in SQL strings
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
