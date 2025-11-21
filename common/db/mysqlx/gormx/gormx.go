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

// EmptyCheckOptions defines which values should be considered "empty"
type EmptyCheckOptions struct {
	CheckNull        bool     // Check for NULL (default: true)
	CheckEmptyString bool     // Check for empty string after trim (default: true)
	CheckZero        bool     // Check for numeric 0 (default: true)
	CheckFalse       bool     // Check for boolean false (default: false)
	CheckNegative    bool     // Check for negative numbers (default: false)
	SpecialStrings   []string // Special strings to treat as empty, e.g. ["null", "NULL", "N/A"]
}

// DefaultEmptyCheckOptions returns the default empty check options
func DefaultEmptyCheckOptions() EmptyCheckOptions {
	return EmptyCheckOptions{
		CheckNull:        true,
		CheckEmptyString: true,
		CheckZero:        true,
		CheckFalse:       false,
		CheckNegative:    false,
		SpecialStrings:   nil,
	}
}

// AssignmentNotEmptyColumns creates assignment clauses that only update non-empty values
// It checks for NULL, empty string (after trim), and numeric 0 by default
func AssignmentNotEmptyColumns(names []string) clause.Set {
	return AssignmentNotEmptyColumnsWithOptions(names, DefaultEmptyCheckOptions())
}

// AssignmentNotEmptyColumnsWithOptions creates assignment clauses with custom empty check options
func AssignmentNotEmptyColumnsWithOptions(names []string, opts EmptyCheckOptions) clause.Set {
	values := make(map[string]any)
	for _, name := range names {
		var conditions []string

		// Check for NULL
		if opts.CheckNull {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) IS NULL", name))
		}

		// Check for empty string (after trim)
		if opts.CheckEmptyString {
			conditions = append(conditions, fmt.Sprintf("TRIM(VALUES(%s)) = ''", name))
		}

		// Check for numeric 0
		if opts.CheckZero {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) = 0", name))
		}

		// Check for boolean false
		if opts.CheckFalse {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) = FALSE", name))
		}

		// Check for negative numbers
		if opts.CheckNegative {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) < 0", name))
		}

		// Check for special strings
		for _, specialStr := range opts.SpecialStrings {
			conditions = append(conditions, fmt.Sprintf("VALUES(%s) = '%s'", name, specialStr))
		}

		// Build the CASE statement
		var caseSQL string
		if len(conditions) > 0 {
			conditionsStr := strings.Join(conditions, " OR ")
			caseSQL = fmt.Sprintf(
				"CASE WHEN %s THEN %s ELSE VALUES(%s) END",
				conditionsStr, name, name)
		} else {
			// No conditions, just use the new value
			caseSQL = fmt.Sprintf("VALUES(%s)", name)
		}

		values[name] = field.NewUnsafeFieldRaw(caseSQL)
	}

	return clause.Assignments(values)
}
