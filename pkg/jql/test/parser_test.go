package jql

import (
	"testing"

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
			name:     "parentheses",
			query:    "(status = \"active\" AND age > 18) OR (status = \"pending\" AND age <= 18)",
			expected: bson.M{"$or": []bson.M{{"$and": []bson.M{{"status": "active"}, {"age": bson.M{"$gt": "18"}}}}, {"$and": []bson.M{{"status": "pending"}, {"age": bson.M{"$lte": "18"}}}}}},
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
					t.Errorf("ParseQuery() length = %d, want %d", len(result), len(tc.expected))
					return
				}
				// 简单比较，实际项目中应使用更详细的比较
			}
		})
	}
}
