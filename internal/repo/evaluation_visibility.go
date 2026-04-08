package repo

import (
	"fmt"

	"gorm.io/gorm"
)

func visibleTeacherEvaluationCondition(table string) string {
	return fmt.Sprintf(`%s.status = 'approved'
		AND (
			%s.mode <> 'linked'
			OR EXISTS (
				SELECT 1
				FROM course_teachers ct
				WHERE ct.course_id = %s.course_id
					AND ct.teacher_id = %s.teacher_id
					AND ct.status = 'active'
			)
		)`, table, table, table, table)
}

func visibleCourseEvaluationCondition(table string) string {
	return fmt.Sprintf(`%s.status = 'approved'
		AND (
			%s.mode <> 'linked'
			OR EXISTS (
				SELECT 1
				FROM course_teachers ct
				WHERE ct.course_id = %s.course_id
					AND ct.teacher_id = %s.teacher_id
					AND ct.status = 'active'
			)
		)`, table, table, table, table)
}

func applyVisibleTeacherEvaluationFilter(db *gorm.DB, table string) *gorm.DB {
	return db.Where(visibleTeacherEvaluationCondition(table))
}

func applyVisibleCourseEvaluationFilter(db *gorm.DB, table string) *gorm.DB {
	return db.Where(visibleCourseEvaluationCondition(table))
}
