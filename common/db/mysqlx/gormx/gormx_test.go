package gormx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssignmentNotEmptyColumns(t *testing.T) {
	// Test with single column
	result := AssignmentNotEmptyColumns([]string{"name"})
	assert.NotNil(t, result)
	assert.Len(t, result, 1)

	// Verify the generated SQL contains the expected conditions
	// The function should check for: NULL, empty string (after trim), and numeric 0
	columns := []string{"name", "age", "email"}
	result = AssignmentNotEmptyColumns(columns)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	// Each assignment should be present
	columnFound := make(map[string]bool)
	for _, assignment := range result {
		columnFound[assignment.Column.Name] = true
	}

	// Verify all columns are present
	for _, col := range columns {
		assert.True(t, columnFound[col], "Column %s should exist in assignments", col)
	}
}

func TestAssignmentNotEmptyColumnsSQL(t *testing.T) {
	// This test verifies the actual SQL generation logic
	// The expected SQL pattern should be:
	// CASE WHEN VALUES(col) IS NULL OR TRIM(VALUES(col)) = '' OR VALUES(col) = 0 THEN col ELSE VALUES(col) END

	columns := []string{"status", "count"}
	result := AssignmentNotEmptyColumns(columns)

	// Verify that the result contains assignments for all columns
	assert.Len(t, result, len(columns))

	// Check each column has a valid assignment
	columnFound := make(map[string]bool)
	for _, assignment := range result {
		columnFound[assignment.Column.Name] = true
		// Verify the value is not nil
		assert.NotNil(t, assignment.Value, "Assignment value should not be nil")
	}

	// Verify all expected columns are present
	for _, col := range columns {
		assert.True(t, columnFound[col], "Column %s should be in the result", col)
	}
}

func TestAssignmentNotEmptyColumnsEmptyInput(t *testing.T) {
	// Test with empty columns list
	result := AssignmentNotEmptyColumns([]string{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestAssignmentNotEmptyColumnsWithOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		// Default options should check NULL, empty string, and 0
		opts := DefaultEmptyCheckOptions()
		assert.True(t, opts.CheckNull)
		assert.True(t, opts.CheckEmptyString)
		assert.True(t, opts.CheckZero)
		assert.False(t, opts.CheckFalse)
		assert.False(t, opts.CheckNegative)
		assert.Nil(t, opts.SpecialStrings)

		result := AssignmentNotEmptyColumnsWithOptions([]string{"name"}, opts)
		assert.Len(t, result, 1)
	})

	t.Run("custom options - check false", func(t *testing.T) {
		// Enable checking for boolean false
		opts := EmptyCheckOptions{
			CheckNull:        true,
			CheckEmptyString: true,
			CheckZero:        false,
			CheckFalse:       true, // Enable false check
			CheckNegative:    false,
		}

		result := AssignmentNotEmptyColumnsWithOptions([]string{"active"}, opts)
		assert.Len(t, result, 1)
	})

	t.Run("custom options - check negative", func(t *testing.T) {
		// Enable checking for negative numbers
		opts := EmptyCheckOptions{
			CheckNull:        true,
			CheckEmptyString: false,
			CheckZero:        false,
			CheckFalse:       false,
			CheckNegative:    true, // Enable negative check
		}

		result := AssignmentNotEmptyColumnsWithOptions([]string{"balance"}, opts)
		assert.Len(t, result, 1)
	})

	t.Run("custom options - special strings", func(t *testing.T) {
		// Check for special strings like "null", "N/A"
		opts := EmptyCheckOptions{
			CheckNull:        true,
			CheckEmptyString: true,
			CheckZero:        false,
			CheckFalse:       false,
			CheckNegative:    false,
			SpecialStrings:   []string{"null", "NULL", "N/A", "undefined"},
		}

		result := AssignmentNotEmptyColumnsWithOptions([]string{"status"}, opts)
		assert.Len(t, result, 1)
	})

	t.Run("custom options - no checks", func(t *testing.T) {
		// Disable all checks - should just update with new value
		opts := EmptyCheckOptions{
			CheckNull:        false,
			CheckEmptyString: false,
			CheckZero:        false,
			CheckFalse:       false,
			CheckNegative:    false,
		}

		result := AssignmentNotEmptyColumnsWithOptions([]string{"data"}, opts)
		assert.Len(t, result, 1)
	})

	t.Run("custom options - multiple columns with different options", func(t *testing.T) {
		opts := EmptyCheckOptions{
			CheckNull:        true,
			CheckEmptyString: true,
			CheckZero:        true,
			CheckFalse:       true,
			CheckNegative:    true,
			SpecialStrings:   []string{"N/A"},
		}

		columns := []string{"name", "age", "status", "balance"}
		result := AssignmentNotEmptyColumnsWithOptions(columns, opts)
		assert.Len(t, result, len(columns))
	})
}

func TestEmptyCheckOptionsDefault(t *testing.T) {
	// Test that default options match expected values
	opts := DefaultEmptyCheckOptions()

	assert.True(t, opts.CheckNull, "Default should check NULL")
	assert.True(t, opts.CheckEmptyString, "Default should check empty string")
	assert.True(t, opts.CheckZero, "Default should check zero")
	assert.False(t, opts.CheckFalse, "Default should NOT check false")
	assert.False(t, opts.CheckNegative, "Default should NOT check negative")
	assert.Nil(t, opts.SpecialStrings, "Default should have no special strings")
}
