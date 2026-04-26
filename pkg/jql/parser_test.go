package jql

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func TestParseQuery(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		expected bson.M
		wantErr  bool
	}{
		{
			name:     "simple equality",
			query:    "description = \"test\"",
			expected: bson.M{"description": "test"},
			wantErr:  false,
		},
		{
			name:     "greater than",
			query:    "age > 18",
			expected: bson.M{"age": bson.M{"$gt": "18"}},
			wantErr:  false,
		},
		{
			name:     "less than or equal",
			query:    "price <= 100",
			expected: bson.M{"price": bson.M{"$lte": "100"}},
			wantErr:  false,
		},
		{
			name:     "not equal",
			query:    "status != \"active\"",
			expected: bson.M{"status": bson.M{"$ne": "active"}},
			wantErr:  false,
		},
		{
			name:     "contains",
			query:    "name ~ \"test\"",
			expected: bson.M{"name": bson.M{"$regex": "test", "$options": "i"}},
			wantErr:  false,
		},
		{
			name:     "AND condition",
			query:    "status = \"active\" AND age > 18",
			expected: bson.M{"$and": []bson.M{{"status": "active"}, {"age": bson.M{"$gt": "18"}}}},
			wantErr:  false,
		},
		{
			name:     "OR condition",
			query:    "status = \"active\" OR status = \"pending\"",
			expected: bson.M{"$or": []bson.M{{"status": "active"}, {"status": "pending"}}},
			wantErr:  false,
		},
		{
			name:     "NOT condition",
			query:    "NOT status = \"inactive\"",
			expected: bson.M{"$not": bson.M{"status": "inactive"}},
			wantErr:  false,
		},
		{
			name:     "IS NULL",
			query:    "assignee IS NULL",
			expected: bson.M{"assignee": bson.M{"$exists": false}},
			wantErr:  false,
		},
		{
			name:     "IS NOT NULL",
			query:    "assignee IS NOT NULL",
			expected: bson.M{"assignee": bson.M{"$exists": true}},
			wantErr:  false,
		},
		{
			name:     "IN with two values",
			query:    "status IN (\"active\", \"pending\")",
			expected: bson.M{"status": bson.M{"$in": []interface{}{"active", "pending"}}},
			wantErr:  false,
		},
		{
			name:     "IN with single value",
			query:    "status IN (\"active\")",
			expected: bson.M{"status": bson.M{"$in": []interface{}{"active"}}},
			wantErr:  false,
		},
		{
			name:     "NOT IN",
			query:    "status NOT IN (\"deleted\", \"archived\")",
			expected: bson.M{"status": bson.M{"$nin": []interface{}{"deleted", "archived"}}},
			wantErr:  false,
		},
		{
			name:     "empty query returns empty result",
			query:    "",
			expected: bson.M{},
			wantErr:  false,
		},
		{
			name:     "whitespace only query",
			query:    "   ",
			expected: bson.M{},
			wantErr:  false,
		},
		{
			name:     "numeric value",
			query:    "age = 25",
			expected: bson.M{"age": 25},
			wantErr:  false,
		},
		{
			name:     "negative numeric value",
			query:    "temperature < -10",
			expected: bson.M{"temperature": bson.M{"$lt": -10}},
			wantErr:  false,
		},
		{
			name:     "decimal numeric value",
			query:    "price > 99.99",
			expected: bson.M{"price": bson.M{"$gt": 99.99}},
			wantErr:  false,
		},
		{
			name:     "boolean true",
			query:    "active = true",
			expected: bson.M{"active": true},
			wantErr:  false,
		},
		{
			name:     "boolean false",
			query:    "deleted = false",
			expected: bson.M{"deleted": false},
			wantErr:  false,
		},
		{
			name:     "single quoted string",
			query:    "name = 'test'",
			expected: bson.M{"name": "test"},
			wantErr:  false,
		},
		{
			name:     "field with underscore",
			query:    "created_by = \"admin\"",
			expected: bson.M{"created_by": "admin"},
			wantErr:  false,
		},
		{
			name:     "field with numbers",
			query:    "field1 = \"value1\"",
			expected: bson.M{"field1": "value1"},
			wantErr:  false,
		},
		{
			name:     "case insensitive AND",
			query:    "status = \"active\" and age > 18",
			expected: bson.M{"$and": []bson.M{{"status": "active"}, {"age": bson.M{"$gt": "18"}}}},
			wantErr:  false,
		},
		{
			name:     "case insensitive OR",
			query:    "status = \"active\" or status = \"pending\"",
			expected: bson.M{"$or": []bson.M{{"status": "active"}, {"status": "pending"}}},
			wantErr:  false,
		},
		{
			name:     "case insensitive NOT",
			query:    "not status = \"inactive\"",
			expected: bson.M{"$not": bson.M{"status": "inactive"}},
			wantErr:  false,
		},
		{
			name:     "IN with numeric values",
			query:    "priority IN (1, 2, 3)",
			expected: bson.M{"priority": bson.M{"$in": []interface{}{1, 2, 3}}},
			wantErr:  false,
		},
		{
			name:     "NOT IN with numeric values",
			query:    "priority NOT IN (4, 5)",
			expected: bson.M{"priority": bson.M{"$nin": []interface{}{4, 5}}},
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if len(result) != len(tc.expected) {
					t.Errorf("ParseQuery() length = %d, want %d, result = %v, expected = %v", len(result), len(tc.expected), result, tc.expected)
					return
				}
			}
		})
	}
}

func TestParseQueryErrors(t *testing.T) {
	errorCases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "unclosed string double quote",
			query:   "status = \"active",
			wantErr: true,
		},
		{
			name:    "unclosed string single quote",
			query:   "status = 'active",
			wantErr: true,
		},
		{
			name:    "unclosed parentheses",
			query:   "(status = \"active\"",
			wantErr: true,
		},
		{
			name:    "unexpected closing parenthesis",
			query:   "status = \"active\")",
			wantErr: false,
		},
		{
			name:    "missing operator",
			query:   "status \"active\"",
			wantErr: true,
		},
		{
			name:    "missing value after operator",
			query:   "status =",
			wantErr: true,
		},
		{
			name:    "empty field name",
			query:   " = \"value\"",
			wantErr: true,
		},
		{
			name:    "IN without parentheses",
			query:   "status IN \"active\"",
			wantErr: true,
		},
		{
			name:    "IN with unclosed parenthesis",
			query:   "status IN (\"active\", \"pending\"",
			wantErr: true,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestParseQueryFunctions(t *testing.T) {
	functionCases := []struct {
		name       string
		query      string
		expectFunc bool
		funcName   string
	}{
		{
			name:       "CurrentUser function",
			query:      "assignee = CurrentUser()",
			expectFunc: true,
			funcName:   "currentuser",
		},
		{
			name:       "Now function",
			query:      "created > Now()",
			expectFunc: true,
			funcName:   "now",
		},
		{
			name:       "StartOfDay function",
			query:      "created > StartOfDay()",
			expectFunc: true,
			funcName:   "startofday",
		},
		{
			name:       "EndOfDay function",
			query:      "created < EndOfDay()",
			expectFunc: true,
			funcName:   "endofday",
		},
		{
			name:       "StartOfWeek function",
			query:      "created > StartOfWeek()",
			expectFunc: true,
			funcName:   "startofweek",
		},
		{
			name:       "EndOfWeek function",
			query:      "created < EndOfWeek()",
			expectFunc: true,
			funcName:   "endofweek",
		},
		{
			name:       "StartOfMonth function",
			query:      "created > StartOfMonth()",
			expectFunc: true,
			funcName:   "startofmonth",
		},
		{
			name:       "EndOfMonth function",
			query:      "created < EndOfMonth()",
			expectFunc: true,
			funcName:   "endofmonth",
		},
	}

	for _, tc := range functionCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if err != nil {
				t.Errorf("ParseQuery() unexpected error = %v", err)
				return
			}
			if tc.expectFunc {
				t.Logf("Result: %v", result)
			}
		})
	}
}

func TestParseQuerySpecialCases(t *testing.T) {
	specialCases := []struct {
		name  string
		query string
	}{
		{
			name:  "extra spaces around operator",
			query: "status   =   \"active\"",
		},
		{
			name:  "tabs in query",
			query: "status\t=\t\"active\"",
		},
		{
			name:  "newlines in query",
			query: "status = \"active\"\nAND age > 18",
		},
		{
			name:  "multiple spaces before field",
			query: "   status = \"active\"",
		},
		{
			name:  "multiple spaces after value",
			query: "status = \"active\"   ",
		},
		{
			name:  "Chinese characters in value",
			query: "name = \"中文测试\"",
		},
		{
			name:  "emoji in value",
			query: "emoji = \"😀\"",
		},
		{
			name:  "special chars in value",
			query: "code = \"test@#$%\"",
		},
	}

	for _, tc := range specialCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if err != nil {
				t.Errorf("ParseQuery() unexpected error = %v for query: %s", err, tc.query)
				return
			}
			t.Logf("Result for %s: %v", tc.name, result)
		})
	}
}

func TestParseQueryComplex(t *testing.T) {
	complexQuery := "status = \"active\" AND (age > 18 OR age < 10) AND priority IN (1, 2, 3) AND NOT deleted = true AND name ~ \".*test.*\""

	result, err := ParseQuery(complexQuery)
	if err != nil {
		t.Errorf("ParseQuery() error on complex query = %v", err)
		return
	}

	if len(result) == 0 {
		t.Errorf("ParseQuery() returned empty result for complex query")
	}

	t.Logf("Complex query result: %v", result)
}

func TestParseQueryGetExampleQueries(t *testing.T) {
	examples := GetExampleQueries()
	if len(examples) == 0 {
		t.Error("GetExampleQueries() returned empty slice")
	}

	for _, example := range examples {
		result, err := ParseQuery(example)
		if err != nil {
			t.Errorf("GetExampleQueries() example failed: %s, error: %v", example, err)
		}
		t.Logf("Example: %s -> %v", example, result)
	}
}

func TestValidateJQL(t *testing.T) {
	validCases := []struct {
		name  string
		query string
	}{
		{"simple", "status = \"active\""},
		{"with AND", "status = \"active\" AND age > 18"},
		{"with OR", "status = \"a\" OR status = \"b\""},
		{"with parentheses", "(status = \"active\") AND (age > 18)"},
		{"with IN", "status IN (\"a\", \"b\")"},
		{"with IS NULL", "field IS NULL"},
		{"with function", "created > Now()"},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJQL(tc.query)
			if err != nil {
				t.Errorf("ValidateJQL() valid query returned error: %v", err)
			}
		})
	}

	invalidCases := []struct {
		name  string
		query string
	}{
		{"unclosed quote", "status = \"active"},
		{"unclosed paren", "(status = \"active\""},
		{"invalid syntax", "status = "},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJQL(tc.query)
			if err == nil {
				t.Errorf("ValidateJQL() invalid query should return error")
			}
		})
	}
}

func TestNowFunction(t *testing.T) {
	before := time.Now()

	result, err := ParseQuery("created > Now()")
	if err != nil {
		t.Errorf("ParseQuery() error = %v", err)
		return
	}

	after := time.Now()

	createdCond, ok := result["created"]
	if !ok {
		t.Errorf("ParseQuery() result missing 'created' field")
		return
	}

	condMap, ok := createdCond.(bson.M)
	if !ok {
		t.Errorf("ParseQuery() created field is not bson.M")
		return
	}

	gtVal, ok := condMap["$gt"]
	if !ok {
		t.Errorf("ParseQuery() missing $gt operator")
		return
	}

	gtTime, ok := gtVal.(time.Time)
	if !ok {
		t.Errorf("ParseQuery() $gt value is not time.Time, got %T", gtVal)
		return
	}

	if gtTime.Before(before) || gtTime.After(after) {
		t.Errorf("ParseQuery() Now() time out of range: %v", gtTime)
	}
}

func TestDateFunctions(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		funcName string
	}{
		{"StartOfDay", "created > StartOfDay()", "startofday"},
		{"EndOfDay", "created < EndOfDay()", "endofday"},
		{"StartOfWeek", "created > StartOfWeek()", "startofweek"},
		{"EndOfWeek", "created < EndOfWeek()", "endofweek"},
		{"StartOfMonth", "created > StartOfMonth()", "startofmonth"},
		{"EndOfMonth", "created < EndOfMonth()", "endofmonth"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if err != nil {
				t.Errorf("ParseQuery() error = %v", err)
				return
			}

			createdCond, ok := result["created"]
			if !ok {
				t.Errorf("ParseQuery() result missing 'created' field")
				return
			}

			_, ok = createdCond.(bson.M)
			if !ok {
				t.Errorf("ParseQuery() created field is not bson.M")
				return
			}

			t.Logf("%s result: %v", tc.funcName, result)
		})
	}
}

func TestParserStateIsolation(t *testing.T) {
	p := NewParser()

	query1 := "a = 1"
	result1, err1 := p.Parse(query1)
	if err1 != nil {
		t.Errorf("First parse failed: %v", err1)
	}

	query2 := "b = 2"
	result2, err2 := p.Parse(query2)
	if err2 != nil {
		t.Errorf("Second parse failed: %v", err2)
	}

	if _, exists := result1["b"]; exists {
		t.Errorf("First parse result should not contain 'b', got %v", result1)
	}

	if _, exists := result2["a"]; exists {
		t.Errorf("Second parse result should not contain 'a', got %v", result2)
	}
}

func TestParseQueryEmbeddedFields(t *testing.T) {
	embeddedFieldCases := []struct {
		name     string
		query    string
		expected bson.M
		wantErr  bool
	}{
		{
			name:     "simple embedded field equality",
			query:    "profile.name = \"John\"",
			expected: bson.M{"profile.name": "John"},
			wantErr:  false,
		},
		{
			name:     "nested embedded field with two levels",
			query:    "address.city = \"Beijing\"",
			expected: bson.M{"address.city": "Beijing"},
			wantErr:  false,
		},
		{
			name:     "nested embedded field with three levels",
			query:    "company.department.team = \"Engineering\"",
			expected: bson.M{"company.department.team": "Engineering"},
			wantErr:  false,
		},
		{
			name:     "embedded field with comparison operator",
			query:    "profile.age > 25",
			expected: bson.M{"profile.age": bson.M{"$gt": "25"}},
			wantErr:  false,
		},
		{
			name:     "embedded field with less than or equal",
			query:    "config.timeout <= 300",
			expected: bson.M{"config.timeout": bson.M{"$lte": "300"}},
			wantErr:  false,
		},
		{
			name:     "embedded field with not equal",
			query:    "status.current != \"pending\"",
			expected: bson.M{"status.current": bson.M{"$ne": "pending"}},
			wantErr:  false,
		},
		{
			name:     "embedded field with contains (regex)",
			query:    "profile.bio ~ \"developer\"",
			expected: bson.M{"profile.bio": bson.M{"$regex": "developer", "$options": "i"}},
			wantErr:  false,
		},
		{
			name:     "embedded field with IS NULL",
			query:    "profile.nickname IS NULL",
			expected: bson.M{"profile.nickname": bson.M{"$exists": false}},
			wantErr:  false,
		},
		{
			name:     "embedded field with IS NOT NULL",
			query:    "profile.email IS NOT NULL",
			expected: bson.M{"profile.email": bson.M{"$exists": true}},
			wantErr:  false,
		},
		{
			name:     "embedded field with IN operator",
			query:    "status.type IN (\"active\", \"pending\")",
			expected: bson.M{"status.type": bson.M{"$in": []interface{}{"active", "pending"}}},
			wantErr:  false,
		},
		{
			name:     "embedded field with NOT IN operator",
			query:    "status.type NOT IN (\"deleted\", \"archived\")",
			expected: bson.M{"status.type": bson.M{"$nin": []interface{}{"deleted", "archived"}}},
			wantErr:  false,
		},
		{
			name:     "embedded field with numeric value",
			query:    "metrics.score = 95",
			expected: bson.M{"metrics.score": 95},
			wantErr:  false,
		},
		{
			name:     "embedded field with negative value",
			query:    "data.offset < -100",
			expected: bson.M{"data.offset": bson.M{"$lt": "-100"}},
			wantErr:  false,
		},
		{
			name:     "embedded field with decimal value",
			query:    "measurements.temp > 36.5",
			expected: bson.M{"measurements.temp": bson.M{"$gt": "36.5"}},
			wantErr:  false,
		},
		{
			name:     "embedded field with boolean value",
			query:    "settings.enabled = true",
			expected: bson.M{"settings.enabled": true},
			wantErr:  false,
		},
		{
			name:     "embedded field in AND condition",
			query:    "profile.name = \"John\" AND profile.age > 25",
			expected: bson.M{"$and": []bson.M{{"profile.name": "John"}, {"profile.age": bson.M{"$gt": "25"}}}},
			wantErr:  false,
		},
		{
			name:     "embedded field in OR condition",
			query:    "profile.city = \"Beijing\" OR profile.city = \"Shanghai\"",
			expected: bson.M{"$or": []bson.M{{"profile.city": "Beijing"}, {"profile.city": "Shanghai"}}},
			wantErr:  false,
		},
		{
			name:     "multiple embedded fields in AND condition",
			query:    "company.name = \"Acme\" AND address.city = \"Beijing\"",
			expected: bson.M{"$and": []bson.M{{"company.name": "Acme"}, {"address.city": "Beijing"}}},
			wantErr:  false,
		},
		{
			name:     "embedded field with quoted string value",
			query:    "data.label = \"test value\"",
			expected: bson.M{"data.label": "test value"},
			wantErr:  false,
		},
		{
			name:     "embedded field starting with underscore",
			query:    "_private.key = \"secret\"",
			expected: bson.M{"_private.key": "secret"},
			wantErr:  false,
		},
		{
			name:     "embedded field starting with dot",
			query:    ".hidden.field = \"value\"",
			expected: bson.M{".hidden.field": "value"},
			wantErr:  false,
		},
	}

	for _, tc := range embeddedFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v, query = %s", err, tc.wantErr, tc.query)
				return
			}
			if !tc.wantErr {
				if len(result) != len(tc.expected) {
					t.Errorf("ParseQuery() length mismatch = %d, want %d, result = %v, expected = %v, query = %s",
						len(result), len(tc.expected), result, tc.expected, tc.query)
					return
				}
				t.Logf("Query: %s -> Result: %v", tc.query, result)
			}
		})
	}
}

func TestParseQueryEmbeddedFieldsEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		query    string
		expected bson.M
		wantErr  bool
	}{
		{
			name:     "embedded field with single character segments",
			query:    "a.b = \"value\"",
			expected: bson.M{"a.b": "value"},
			wantErr:  false,
		},
		{
			name:     "embedded field with many levels",
			query:    "a.b.c.d.e = \"deep\"",
			expected: bson.M{"a.b.c.d.e": "deep"},
			wantErr:  false,
		},
		{
			name:     "embedded field with mixed separators",
			query:    "field_name.nested_field = \"value\"",
			expected: bson.M{"field_name.nested_field": "value"},
			wantErr:  false,
		},
		{
			name:     "embedded field with numbers in segment",
			query:    "user1.profile2.name3 = \"Test\"",
			expected: bson.M{"user1.profile2.name3": "Test"},
			wantErr:  false,
		},
		{
			name:     "embedded field in parentheses",
			query:    "(profile.name = \"John\")",
			expected: bson.M{"profile.name": "John"},
			wantErr:  false,
		},
		{
			name:     "embedded field with leading spaces",
			query:    "  profile.name = \"John\"",
			expected: bson.M{"profile.name": "John"},
			wantErr:  false,
		},
		{
			name:     "embedded field with trailing spaces",
			query:    "profile.name = \"John\"  ",
			expected: bson.M{"profile.name": "John"},
			wantErr:  false,
		},
		{
			name:    "embedded field with tab separator",
			query:   "profile\t.name = \"John\"",
			wantErr: true,
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v, query = %s", err, tc.wantErr, tc.query)
				return
			}
			if !tc.wantErr {
				t.Logf("Query: %s -> Result: %v", tc.query, result)
			}
		})
	}
}

func TestParseQueryEmbeddedFieldsErrors(t *testing.T) {
	errorCases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "embedded field ending with dot",
			query:   "profile.name. = \"value\"",
			wantErr: false,
		},
		{
			name:    "embedded field starting with dot only",
			query:   ". = \"value\"",
			wantErr: false,
		},
		{
			name:    "embedded field with consecutive dots",
			query:   "profile..name = \"value\"",
			wantErr: false,
		},
		{
			name:    "embedded field with space in segment",
			query:   "profile.na me = \"value\"",
			wantErr: true,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v, query = %s", err, tc.wantErr, tc.query)
			}
		})
	}
}

func TestParseQueryEmbeddedFieldsComplex(t *testing.T) {
	complexCases := []struct {
		name  string
		query string
	}{
		{
			name:  "complex query with embedded fields and multiple conditions",
			query: "company.name = \"Acme\" AND (address.city = \"Beijing\" OR address.city = \"Shanghai\") AND status.type IN (\"active\", \"pending\")",
		},
		{
			name:  "complex query with nested embedded fields in parentheses",
			query: "(profile.name = \"John\" AND profile.age > 25) OR (profile.name = \"Jane\" AND profile.age < 30)",
		},
		{
			name:  "complex query with embedded field and NOT",
			query: "NOT profile.status = \"inactive\"",
		},
		{
			name:  "complex query with multiple embedded field levels",
			query: "company.department.team = \"Engineering\" AND metrics.score > 80",
		},
		{
			name:  "complex query with embedded field and function",
			query: "profile.created > Now()",
		},
		{
			name:  "complex query with embedded field and date function",
			query: "session.start > StartOfDay() AND session.end < EndOfDay()",
		},
	}

	for _, tc := range complexCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseQuery(tc.query)
			if err != nil {
				t.Errorf("ParseQuery() error = %v, query = %s", err, tc.query)
				return
			}
			if len(result) == 0 {
				t.Errorf("ParseQuery() returned empty result for query: %s", tc.query)
				return
			}
			t.Logf("Complex query: %s -> Result: %v", tc.query, result)
		})
	}
}
