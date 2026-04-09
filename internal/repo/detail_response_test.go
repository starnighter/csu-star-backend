package repo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTeacherDetailJSONOmitsLegacyFields(t *testing.T) {
	payload, err := json.Marshal(TeacherDetail{
		ID:            456,
		Name:          "任叶庆",
		EvalCount:     1,
		FavoriteCount: 0,
		Courses: []TeacherCourseItem{
			{ID: 28, Name: "高等数学A（一）"},
		},
	})
	if err != nil {
		t.Fatalf("marshal teacher detail failed: %v", err)
	}

	jsonText := string(payload)
	legacyFields := []string{
		`"evaluation_count"`,
		`"evaluation_anchor"`,
		`"detail_path"`,
		`"resource_collection_path"`,
	}
	for _, field := range legacyFields {
		if strings.Contains(jsonText, field) {
			t.Fatalf("expected teacher detail json to omit %s, got %s", field, jsonText)
		}
	}
}

func TestCourseDetailJSONOmitsLegacyFields(t *testing.T) {
	payload, err := json.Marshal(CourseDetail{
		ID:            28,
		Name:          "高等数学A（一）",
		EvalCount:     3,
		ResourceCount: 2,
		Teachers: []CourseTeacherItem{
			{ID: 456, Name: "任叶庆", Title: "教授"},
		},
	})
	if err != nil {
		t.Fatalf("marshal course detail failed: %v", err)
	}

	jsonText := string(payload)
	legacyFields := []string{
		`"evaluation_count"`,
		`"resource_collection_path"`,
		`"evaluation_anchor"`,
		`"resources_preview"`,
		`"download_total"`,
		`"detail_path"`,
	}
	for _, field := range legacyFields {
		if strings.Contains(jsonText, field) {
			t.Fatalf("expected course detail json to omit %s, got %s", field, jsonText)
		}
	}
}
