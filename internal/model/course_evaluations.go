package model

import (
	"time"

	"csu-star-backend/pkg/utils"
	"gorm.io/gorm"
)

type CourseEvaluations struct {
	ID                 int64            `gorm:"primary_key" json:"id,string"`
	UserID             int64            `gorm:"type:bigint;not null" json:"user_id,string"`
	CourseID           int64            `gorm:"type:bigint;not null" json:"course_id,string"`
	TeacherID          *int64           `gorm:"type:bigint" json:"teacher_id,string"`
	Mode               EvaluationMode   `gorm:"type:varchar(16);default:'single'" json:"mode"`
	MirrorEvaluationID *int64           `gorm:"type:bigint" json:"mirror_evaluation_id,string"`
	MirrorEntityType   MirrorEntityType `gorm:"type:varchar(16)" json:"mirror_entity_type"`
	TeacherName        string           `gorm:"-" json:"teacher_name"`
	WorkloadScore      int              `gorm:"type:integer;not null" json:"workload_score"`
	GainScore          int              `gorm:"type:integer;not null" json:"gain_score"`
	DifficultyScore    int              `gorm:"type:integer;not null" json:"difficulty_score"`
	TeachingScore      *int             `gorm:"type:integer" json:"teaching_score"`
	GradingScore       *int             `gorm:"type:integer" json:"grading_score"`
	AttendanceScore    *int             `gorm:"type:integer" json:"attendance_score"`
	Comment            string           `gorm:"type:text" json:"comment"`
	IsAnonymous        bool             `gorm:"type:boolean;default:false" json:"is_anonymous"`
	Status             ResourceStatus   `gorm:"type:resource_status" json:"status"`
	CreatedAt          time.Time        `gorm:"type:autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time        `gorm:"type:autoUpdateTime" json:"updated_at"`
}

type CourseEvaluationReplies struct {
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

func (r *CourseEvaluationReplies) BeforeCreate(tx *gorm.DB) error {
	if r.ID == 0 {
		r.ID = utils.GenerateID()
	}
	return nil
}
