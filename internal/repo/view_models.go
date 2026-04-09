package repo

import (
	"fmt"
	"time"
)

type CourseBrief struct {
	ID                     int64   `json:"id,string"`
	Code                   string  `json:"code"`
	Name                   string  `json:"name"`
	Credits                float64 `json:"credits,omitempty"`
	CourseType             string  `json:"course_type,omitempty"`
	ResourceCollectionPath string  `json:"resource_collection_path,omitempty"`
	DetailPath             string  `json:"detail_path,omitempty"`
}

type TeacherBrief struct {
	ID         int64  `json:"id,string"`
	Name       string `json:"name"`
	Title      string `json:"title,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	DetailPath string `json:"detail_path,omitempty"`
}

type UserBrief struct {
	ID        int64  `json:"id,string"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role,omitempty"`
}

type ResourceCard struct {
	ID            int64                    `json:"id,string"`
	Title         string                   `json:"title"`
	Description   string                   `json:"-"`
	Type          string                   `json:"resource_type"`
	PointsCost    int                      `json:"-"`
	DownloadCount int                      `json:"downloads"`
	LikeCount     int                      `json:"likes"`
	CommentCount  int                      `json:"-"`
	ViewCount     int                      `json:"views"`
	FavoriteCount int                      `json:"favorite_count,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"-"`
	DetailPath    string                   `json:"detail_path,omitempty"`
	FileCount     int                      `json:"file_count,omitempty" gorm:"-"`
	FirstFile     *ResourceCardFilePreview `json:"first_file,omitempty" gorm:"-"`
}

type ResourceCardFilePreview struct {
	Filename  string `json:"filename"`
	Mime      string `json:"mime,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
}

type EvaluationReply struct {
	ID              int64      `json:"id,string"`
	EvaluationID    int64      `json:"evaluation_id,string"`
	UserID          int64      `json:"-"`
	User            *UserBrief `json:"user,omitempty"`
	Content         string     `json:"content"`
	IsAnonymous     bool       `json:"is_anonymous"`
	ReplyToReplyID  *int64     `json:"reply_to_reply_id,omitempty,string"`
	ReplyToUserID   *int64     `json:"-"`
	ReplyToUser     *UserBrief `json:"reply_to_user,omitempty"`
	Likes           int64      `json:"likes"`
	IsLiked         bool       `json:"is_liked"`
	AuthorName      string     `json:"-"`
	AuthorAvatar    string     `json:"-"`
	AuthorRole      string     `json:"-"`
	ReplyToUserName string     `json:"-"`
	ReplyToUserRole string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type CourseResourceCollectionCard struct {
	CourseID               int64          `json:"course_id,string"`
	CourseCode             string         `json:"course_code,omitempty"`
	CourseName             string         `json:"course_name"`
	ResourceCount          int            `json:"resource_count"`
	DownloadCount          int            `json:"download_total"`
	LikeCount              int            `json:"like_total"`
	FavoriteCount          int            `json:"favorite_count,omitempty"`
	ResourceCollectionPath string         `json:"detail_path,omitempty"`
	CourseDetailPath       string         `json:"-"`
	ResourcesPreview       []ResourceCard `json:"resources_preview,omitempty"`
}

type CourseResourceCollectionDetail struct {
	Course           CourseBrief                       `json:"course"`
	ResourceCount    int                               `json:"resource_count"`
	DownloadCount    int                               `json:"download_total"`
	LikeCount        int                               `json:"like_total"`
	FavoriteCount    int                               `json:"favorite_count,omitempty"`
	EvaluationAnchor string                            `json:"evaluation_anchor,omitempty"`
	Items            CourseResourceCollectionItemsPage `json:"items"`
}

type CourseResourceCollectionItemsPage struct {
	Items []ResourceCard `json:"items"`
	Total int64          `json:"total"`
}

type GlobalSearchItem struct {
	Type                   string                        `json:"type"`
	ID                     int64                         `json:"id,string"`
	Title                  string                        `json:"title,omitempty"`
	Name                   string                        `json:"name,omitempty"`
	CourseID               int64                         `json:"course_id,omitempty,string"`
	CourseName             string                        `json:"course_name,omitempty"`
	CourseType             string                        `json:"course_type,omitempty"`
	AvgScore               float64                       `json:"avg_score,omitempty"`
	AvgHomework            float64                       `json:"avg_homework,omitempty"`
	AvgGain                float64                       `json:"avg_gain,omitempty"`
	AvgExamDiff            float64                       `json:"avg_exam_diff,omitempty"`
	DepartmentID           int64                         `json:"department_id,omitempty"`
	DepartmentName         string                        `json:"department_name,omitempty"`
	AvatarURL              string                        `json:"avatar_url,omitempty"`
	AvgQuality             float64                       `json:"avg_quality,omitempty"`
	AvgGrading             float64                       `json:"avg_grading,omitempty"`
	AvgAttendance          float64                       `json:"avg_attendance,omitempty"`
	GoodRate               float64                       `json:"good_rate,omitempty"`
	EvalCount              int64                         `json:"eval_count,omitempty"`
	ResourceCount          int64                         `json:"resource_count,omitempty"`
	TeacherCount           int64                         `json:"teacher_count,omitempty"`
	FavoriteCount          int64                         `json:"favorite_count,omitempty"`
	DownloadTotal          int64                         `json:"download_total,omitempty"`
	LikeTotal              int64                         `json:"like_total,omitempty"`
	ResourcesPreview       []ResourceCard                `json:"resources_preview,omitempty" gorm:"-"`
	Subtitle               string                        `json:"subtitle,omitempty"`
	DetailPath             string                        `json:"detail_path,omitempty"`
	ResourceCollectionPath string                        `json:"resource_collection_path,omitempty"`
	Course                 *CourseBrief                  `json:"course,omitempty" gorm:"-"`
	Teacher                *TeacherBrief                 `json:"teacher,omitempty" gorm:"-"`
	Courses                []CourseBrief                 `json:"courses,omitempty" gorm:"-"`
	Teachers               []TeacherBrief                `json:"teachers,omitempty" gorm:"-"`
	ResourceCollection     *CourseResourceCollectionCard `json:"resource_collection,omitempty" gorm:"-"`
}

func CourseDetailPath(courseID int64) string {
	return fmt.Sprintf("/course/detail?id=%d", courseID)
}

func TeacherDetailPath(teacherID int64) string {
	return fmt.Sprintf("/teacher/detail?id=%d", teacherID)
}

func ResourceDetailPath(resourceID int64) string {
	return fmt.Sprintf("/resource/detail?id=%d", resourceID)
}

func CourseResourceCollectionPath(courseID int64) string {
	return fmt.Sprintf("/resource/course?courseId=%d", courseID)
}

func CourseEvaluationAnchorPath(courseID int64) string {
	return fmt.Sprintf("/course/detail?id=%d#evaluations", courseID)
}
