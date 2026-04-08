package model

import (
	"time"

	"csu-star-backend/pkg/utils"
	"gorm.io/gorm"
)

type EvaluationMode string

type MirrorEntityType string

const (
	EvaluationModeSingle EvaluationMode = "single"
	EvaluationModeLinked EvaluationMode = "linked"

	MirrorEntityTypeTeacher MirrorEntityType = "teacher"
	MirrorEntityTypeCourse  MirrorEntityType = "course"
)

type TeacherEvaluations struct {
	ID                  int64            `gorm:"primary_key" json:"id,string"`
	UserID              int64            `gorm:"type:bigint;not null" json:"user_id,string"`
	TeacherID           int64            `gorm:"type:bigint;not null" json:"teacher_id,string"`
	CourseID            *int64           `gorm:"type:bigint" json:"course_id,string"`
	Mode                EvaluationMode   `gorm:"type:varchar(16);default:'single'" json:"mode"`
	MirrorEvaluationID  *int64           `gorm:"type:bigint" json:"mirror_evaluation_id,string"`
	MirrorEntityType    MirrorEntityType `gorm:"type:varchar(16)" json:"mirror_entity_type"`
	CourseName          string           `gorm:"-" json:"course_name"`
	TeachingScore       int              `gorm:"type:integer;not null" json:"teaching_score"`
	GradingScore        int              `gorm:"type:integer;not null" json:"grading_score"`
	AttendanceScore     int              `gorm:"type:integer;not null" json:"attendance_score"`
	HomeworkScore       *int             `gorm:"column:workload_score;type:integer" json:"workload_score"`
	GainScore           *int             `gorm:"type:integer" json:"gain_score"`
	ExamDifficultyScore *int             `gorm:"column:difficulty_score;type:integer" json:"difficulty_score"`
	Comment             string           `gorm:"type:text" json:"comment"`
	IsAnonymous         bool             `gorm:"type:boolean;default:false" json:"is_anonymous"`
	Status              ResourceStatus   `gorm:"type:resource_status" json:"status"`
	CreatedAt           time.Time        `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time        `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type TeacherEvaluationReplies struct {
	ID             int64      `gorm:"primary_key" json:"id,string"`
	EvaluationID   int64      `gorm:"type:bigint;not null" json:"evaluation_id,string"`
	UserID         int64      `gorm:"type:bigint;not null" json:"user_id,string"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	IsAnonymous    bool       `gorm:"type:boolean;default:false" json:"is_anonymous"`
	ReplyToReplyID *int64     `gorm:"type:bigint" json:"reply_to_reply_id,string"`
	ReplyToUserID  *int64     `gorm:"type:bigint" json:"reply_to_user_id,string"`
	CreatedAt      time.Time  `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"type:autoUpdateTime" json:"updated_at"`
}

func (r *TeacherEvaluationReplies) BeforeCreate(tx *gorm.DB) error {
	if r.ID == 0 {
		r.ID = utils.GenerateID()
	}
	return nil
}
