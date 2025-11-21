package gormx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssignmentNotEmptyColumns(t *testing.T) {
	t.Run("basic string columns and input methods", func(t *testing.T) {
		// Test 1: Plain string columns (default string type)
		result := AssignmentNotEmptyColumns("name", "email", "address")
		assert.Len(t, result, 3)

		// Verify all columns are present
		columnFound := make(map[string]bool)
		for _, assignment := range result {
			columnFound[assignment.Column.Name] = true
			assert.NotNil(t, assignment.Value)
		}
		assert.True(t, columnFound["name"])
		assert.True(t, columnFound["email"])
		assert.True(t, columnFound["address"])

		// Test 2: Prefix notation for different types
		result = AssignmentNotEmptyColumns(
			"domain_id",
			"numeric:age",
			"uuid:user_id",
		)
		assert.Len(t, result, 3)

		// Test 3: TypedColumn functions
		result = AssignmentNotEmptyColumns(
			"name",
			NumericColumn("count"),
			UUIDColumn("order_id"),
		)
		assert.Len(t, result, 3)

		// Test 4: Mixed input methods
		result = AssignmentNotEmptyColumns(
			"name",
			"numeric:age",
			NumericColumn("balance"),
			"uuid:domain_id",
		)
		assert.Len(t, result, 4)

		// Test 5: Empty input
		result = AssignmentNotEmptyColumns()
		assert.Len(t, result, 0)
	})

	t.Run("numeric column types and variations", func(t *testing.T) {
		// Test 1: Basic numeric column (no zero)
		result := AssignmentNotEmptyColumns(NumericColumn("age"))
		assert.Len(t, result, 1)

		// Test 2: Numeric with zero allowed
		result = AssignmentNotEmptyColumns(NumericColumnAllowZero("score"))
		assert.Len(t, result, 1)

		// Test 3: Numeric positive only
		result = AssignmentNotEmptyColumns(NumericColumnPositive("quantity"))
		assert.Len(t, result, 1)

		// Test 4: Prefix notation variations
		result = AssignmentNotEmptyColumns(
			"numeric:count",
			"numeric+zero:score",
			"numeric+positive:amount",
		)
		assert.Len(t, result, 3)

		// Test 5: Short prefix aliases
		result = AssignmentNotEmptyColumns(
			"num:age",
			"n+z:balance",
			"n+p:quantity",
		)
		assert.Len(t, result, 3)
	})

	t.Run("all column types and custom empty values", func(t *testing.T) {
		// Test 1: UUID type
		result := AssignmentNotEmptyColumns(
			UUIDColumn("domain_id"),
			"uuid:user_id",
			"u:order_id",
		)
		assert.Len(t, result, 3)

		// Test 2: DateTime type
		result = AssignmentNotEmptyColumns(
			DateTimeColumn("created_at"),
			"datetime:updated_at",
			"dt:deleted_at",
		)
		assert.Len(t, result, 3)

		// Test 3: JSON type
		result = AssignmentNotEmptyColumns(
			JSONColumn("metadata"),
			"json:config",
			"j:settings",
		)
		assert.Len(t, result, 3)

		// Test 4: String with custom empty values
		result = AssignmentNotEmptyColumns(
			StringColumn("status", "N/A", "null", "NULL"),
			"string:type:unknown:pending",
		)
		assert.Len(t, result, 2)

		// Test 5: Complex mixed scenario (e-commerce order)
		result = AssignmentNotEmptyColumns(
			"uuid:order_id",
			"uuid:user_id",
			"customer_name",
			"string:status:unknown:pending",
			"numeric+positive:quantity",
			"numeric+positive:amount",
			"numeric+zero:discount",
			"datetime:order_time",
			"json:metadata",
			"remark",
		)
		assert.Len(t, result, 10)

		// Verify all columns are present
		columnNames := []string{"order_id", "user_id", "customer_name", "status",
			"quantity", "amount", "discount", "order_time", "metadata", "remark"}
		columnFound := make(map[string]bool)
		for _, assignment := range result {
			columnFound[assignment.Column.Name] = true
		}
		for _, name := range columnNames {
			assert.True(t, columnFound[name], "Column %s should exist", name)
		}
	})
}
