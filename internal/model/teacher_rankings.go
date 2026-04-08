package model

import "time"

type TeacherRankings struct {
	ID           int64     `gorm:"primary_key" json:"id,string"`
	TeacherID    int64     `gorm:"type:bigint;not null" json:"teacher_id,string"`
	DepartmentID int16     `gorm:"type:smallint;not null" json:"department_id"`
	Period       string    `gorm:"type:varchar(16);not null" json:"period"`
	Dimension    string    `gorm:"type:varchar(32);not null" json:"dimension"`
	Rank         int       `gorm:"type:integer;not null" json:"rank"`
	Score        float64   `gorm:"type:numeric(10,2)" json:"score"`
	UpdatedAt    time.Time `gorm:"type:autoUpdateTime" json:"updated_at"`
}
