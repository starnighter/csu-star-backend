package repo

import (
	"strings"
	"testing"
)

func TestVisibleTeacherEvaluationCondition(t *testing.T) {
	condition := visibleTeacherEvaluationCondition("teacher_evaluations")

	requiredParts := []string{
		"teacher_evaluations.status = 'approved'",
		"teacher_evaluations.mode <> 'linked'",
		"FROM course_teachers ct",
		"ct.course_id = teacher_evaluations.course_id",
		"ct.teacher_id = teacher_evaluations.teacher_id",
		"ct.status = 'active'",
	}
	for _, part := range requiredParts {
		if !strings.Contains(condition, part) {
			t.Fatalf("expected teacher visibility condition to contain %q, got %q", part, condition)
		}
	}
}

func TestVisibleCourseEvaluationCondition(t *testing.T) {
	condition := visibleCourseEvaluationCondition("course_evaluations")

	requiredParts := []string{
		"course_evaluations.status = 'approved'",
		"course_evaluations.mode <> 'linked'",
		"FROM course_teachers ct",
		"ct.course_id = course_evaluations.course_id",
		"ct.teacher_id = course_evaluations.teacher_id",
		"ct.status = 'active'",
	}
	for _, part := range requiredParts {
		if !strings.Contains(condition, part) {
			t.Fatalf("expected course visibility condition to contain %q, got %q", part, condition)
		}
	}
}
