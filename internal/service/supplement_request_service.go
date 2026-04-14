package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrSupplementRequestNotFound          = errors.New("supplement request not found")
	ErrSupplementRequestAlreadyReviewed   = errors.New("supplement request already reviewed")
	ErrSupplementRequestInvalidPayload    = errors.New("supplement request invalid payload")
	ErrSupplementRequestReviewNoteMissing = errors.New("supplement request review note missing")
)

func (s *MiscService) CreateSupplementRequest(
	userID int64,
	requestType, contact, teacherName string,
	departmentID *int16,
	courseName, courseType, remark string,
) (*repo.SupplementRequestItem, error) {
	contact = strings.TrimSpace(contact)
	remark = strings.TrimSpace(remark)
	switch strings.TrimSpace(requestType) {
	case "teacher":
		teacherName = strings.TrimSpace(teacherName)
		if contact == "" || teacherName == "" || departmentID == nil || *departmentID <= 0 {
			return nil, ErrSupplementRequestInvalidPayload
		}
		if !s.departmentExists(s.db, *departmentID) {
			return nil, ErrSupplementRequestInvalidPayload
		}
		request := &model.TeacherSupplementRequests{
			UserID:       userID,
			Status:       model.SupplementRequestStatusPending,
			Contact:      contact,
			TeacherName:  teacherName,
			DepartmentID: *departmentID,
			Remark:       remark,
		}
		if err := s.miscRepo.CreateTeacherSupplementRequest(request); err != nil {
			return nil, err
		}
		return s.GetSupplementRequest(request.ID)
	case "course":
		courseName = strings.TrimSpace(courseName)
		normalizedCourseType := normalizeSupplementCourseType(courseType)
		if contact == "" || courseName == "" || normalizedCourseType == "" {
			return nil, ErrSupplementRequestInvalidPayload
		}
		request := &model.CourseSupplementRequests{
			UserID:     userID,
			Status:     model.SupplementRequestStatusPending,
			Contact:    contact,
			CourseName: courseName,
			CourseType: normalizedCourseType,
			Remark:     remark,
		}
		if err := s.miscRepo.CreateCourseSupplementRequest(request); err != nil {
			return nil, err
		}
		return s.GetSupplementRequest(request.ID)
	default:
		return nil, ErrSupplementRequestInvalidPayload
	}
}

func (s *MiscService) ListSupplementRequests(query repo.SupplementRequestListQuery) ([]repo.SupplementRequestItem, int64, error) {
	fillPagination(&query.Page, &query.Size)

	requestType := strings.TrimSpace(query.RequestType)
	loadAll := repo.SupplementRequestListQuery{
		Status:      query.Status,
		RequestType: requestType,
		Keyword:     query.Keyword,
		Page:        1,
		Size:        0,
	}

	items := make([]repo.SupplementRequestItem, 0)
	if requestType == "" || requestType == "teacher" {
		teacherItems, _, err := s.miscRepo.ListTeacherSupplementRequests(loadAll)
		if err != nil {
			return nil, 0, err
		}
		for _, item := range teacherItems {
			items = append(items, toSupplementRequestItemFromTeacher(item))
		}
	}
	if requestType == "" || requestType == "course" {
		courseItems, _, err := s.miscRepo.ListCourseSupplementRequests(loadAll)
		if err != nil {
			return nil, 0, err
		}
		for _, item := range courseItems {
			items = append(items, toSupplementRequestItemFromCourse(item))
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	total := int64(len(items))
	start := (query.Page - 1) * query.Size
	if start >= len(items) {
		return []repo.SupplementRequestItem{}, total, nil
	}
	end := start + query.Size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, nil
}

func (s *MiscService) GetSupplementRequest(id int64) (*repo.SupplementRequestItem, error) {
	teacherItem, err := s.miscRepo.GetTeacherSupplementRequestByID(id)
	if err == nil {
		item := toSupplementRequestItemFromTeacher(*teacherItem)
		return &item, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	courseItem, err := s.miscRepo.GetCourseSupplementRequestByID(id)
	if err == nil {
		item := toSupplementRequestItemFromCourse(*courseItem)
		return &item, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSupplementRequestNotFound
	}
	return nil, err
}

func (s *MiscService) ApproveSupplementRequest(operatorID, id int64, reviewNote string) (*repo.SupplementRequestItem, error) {
	reviewNote = strings.TrimSpace(reviewNote)
	var result *repo.SupplementRequestItem
	err := s.withSupplementWriteTx(func(tx *gorm.DB) error {
		teacherRequest, err := s.miscRepo.GetTeacherSupplementRequestByID(id)
		switch {
		case err == nil:
			item, approveErr := s.approveTeacherSupplementRequest(tx, operatorID, teacherRequest, reviewNote)
			if approveErr != nil {
				return approveErr
			}
			result = item
			return nil
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		courseRequest, err := s.miscRepo.GetCourseSupplementRequestByID(id)
		switch {
		case err == nil:
			item, approveErr := s.approveCourseSupplementRequest(tx, operatorID, courseRequest, reviewNote)
			if approveErr != nil {
				return approveErr
			}
			result = item
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			return ErrSupplementRequestNotFound
		default:
			return err
		}
	})
	return result, err
}

func (s *MiscService) RejectSupplementRequest(operatorID, id int64, reviewNote string) (*repo.SupplementRequestItem, error) {
	reviewNote = strings.TrimSpace(reviewNote)
	if reviewNote == "" {
		return nil, ErrSupplementRequestReviewNoteMissing
	}

	var result *repo.SupplementRequestItem
	err := s.withSupplementWriteTx(func(tx *gorm.DB) error {
		teacherRequest, err := s.miscRepo.GetTeacherSupplementRequestByID(id)
		switch {
		case err == nil:
			item, rejectErr := s.rejectTeacherSupplementRequest(tx, operatorID, teacherRequest, reviewNote)
			if rejectErr != nil {
				return rejectErr
			}
			result = item
			return nil
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		courseRequest, err := s.miscRepo.GetCourseSupplementRequestByID(id)
		switch {
		case err == nil:
			item, rejectErr := s.rejectCourseSupplementRequest(tx, operatorID, courseRequest, reviewNote)
			if rejectErr != nil {
				return rejectErr
			}
			result = item
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			return ErrSupplementRequestNotFound
		default:
			return err
		}
	})
	return result, err
}

func (s *MiscService) approveTeacherSupplementRequest(tx *gorm.DB, operatorID int64, request *repo.TeacherSupplementRequestItem, reviewNote string) (*repo.SupplementRequestItem, error) {
	if request.Status != string(model.SupplementRequestStatusPending) {
		return nil, ErrSupplementRequestAlreadyReviewed
	}
	if request.DepartmentID <= 0 || !s.departmentExists(tx, request.DepartmentID) {
		return nil, ErrSupplementRequestInvalidPayload
	}

	teacher := &model.Teachers{
		Name:         strings.TrimSpace(request.TeacherName),
		DepartmentID: request.DepartmentID,
		Status:       string(model.CourseStatusActive),
	}
	if teacher.Name == "" {
		return nil, ErrSupplementRequestInvalidPayload
	}
	if err := tx.Create(teacher).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":              model.SupplementRequestStatusApproved,
		"reviewed_by":         operatorID,
		"reviewed_at":         now,
		"review_note":         reviewNote,
		"approved_teacher_id": teacher.ID,
		"updated_at":          now,
	}
	if err := tx.Model(&model.TeacherSupplementRequests{}).Where("id = ?", request.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	notification := buildTeacherSupplementReviewNotification(request.UserID, &model.TeacherSupplementRequests{ID: request.ID}, true, reviewNote)
	if err := tx.Create(notification).Error; err != nil {
		return nil, err
	}

	return &repo.SupplementRequestItem{
		ID:                 request.ID,
		UserID:             request.UserID,
		User:               request.User,
		RequestType:        "teacher",
		Status:             string(model.SupplementRequestStatusApproved),
		Contact:            request.Contact,
		TeacherName:        request.TeacherName,
		DepartmentID:       &request.DepartmentID,
		DepartmentName:     request.DepartmentName,
		Remark:             request.Remark,
		ReviewedBy:         &operatorID,
		ReviewedAt:         &now,
		ReviewNote:         reviewNote,
		ApprovedTargetType: "teacher",
		ApprovedTargetID:   &teacher.ID,
		CreatedAt:          request.CreatedAt,
		UpdatedAt:          now,
	}, nil
}

func (s *MiscService) approveCourseSupplementRequest(tx *gorm.DB, operatorID int64, request *repo.CourseSupplementRequestItem, reviewNote string) (*repo.SupplementRequestItem, error) {
	if request.Status != string(model.SupplementRequestStatusPending) {
		return nil, ErrSupplementRequestAlreadyReviewed
	}
	if strings.TrimSpace(request.CourseName) == "" || normalizeSupplementCourseType(request.CourseType) == "" {
		return nil, ErrSupplementRequestInvalidPayload
	}

	course := &model.Courses{
		Name:       strings.TrimSpace(request.CourseName),
		CourseType: normalizeSupplementCourseType(request.CourseType),
		Status:     model.CourseStatusActive,
	}
	if err := tx.Create(course).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":             model.SupplementRequestStatusApproved,
		"reviewed_by":        operatorID,
		"reviewed_at":        now,
		"review_note":        reviewNote,
		"approved_course_id": course.ID,
		"updated_at":         now,
	}
	if err := tx.Model(&model.CourseSupplementRequests{}).Where("id = ?", request.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	notification := buildCourseSupplementReviewNotification(request.UserID, &model.CourseSupplementRequests{ID: request.ID}, true, reviewNote)
	if err := tx.Create(notification).Error; err != nil {
		return nil, err
	}

	return &repo.SupplementRequestItem{
		ID:                 request.ID,
		UserID:             request.UserID,
		User:               request.User,
		RequestType:        "course",
		Status:             string(model.SupplementRequestStatusApproved),
		Contact:            request.Contact,
		CourseName:         request.CourseName,
		CourseType:         string(normalizeSupplementCourseType(request.CourseType)),
		Remark:             request.Remark,
		ReviewedBy:         &operatorID,
		ReviewedAt:         &now,
		ReviewNote:         reviewNote,
		ApprovedTargetType: "course",
		ApprovedTargetID:   &course.ID,
		CreatedAt:          request.CreatedAt,
		UpdatedAt:          now,
	}, nil
}

func (s *MiscService) rejectTeacherSupplementRequest(tx *gorm.DB, operatorID int64, request *repo.TeacherSupplementRequestItem, reviewNote string) (*repo.SupplementRequestItem, error) {
	if request.Status != string(model.SupplementRequestStatusPending) {
		return nil, ErrSupplementRequestAlreadyReviewed
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      model.SupplementRequestStatusRejected,
		"reviewed_by": operatorID,
		"reviewed_at": now,
		"review_note": reviewNote,
		"updated_at":  now,
	}
	if err := tx.Model(&model.TeacherSupplementRequests{}).Where("id = ?", request.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	notification := buildTeacherSupplementReviewNotification(request.UserID, &model.TeacherSupplementRequests{ID: request.ID}, false, reviewNote)
	if err := tx.Create(notification).Error; err != nil {
		return nil, err
	}

	return &repo.SupplementRequestItem{
		ID:             request.ID,
		UserID:         request.UserID,
		User:           request.User,
		RequestType:    "teacher",
		Status:         string(model.SupplementRequestStatusRejected),
		Contact:        request.Contact,
		TeacherName:    request.TeacherName,
		DepartmentID:   &request.DepartmentID,
		DepartmentName: request.DepartmentName,
		Remark:         request.Remark,
		ReviewedBy:     &operatorID,
		ReviewedAt:     &now,
		ReviewNote:     reviewNote,
		CreatedAt:      request.CreatedAt,
		UpdatedAt:      now,
	}, nil
}

func (s *MiscService) rejectCourseSupplementRequest(tx *gorm.DB, operatorID int64, request *repo.CourseSupplementRequestItem, reviewNote string) (*repo.SupplementRequestItem, error) {
	if request.Status != string(model.SupplementRequestStatusPending) {
		return nil, ErrSupplementRequestAlreadyReviewed
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      model.SupplementRequestStatusRejected,
		"reviewed_by": operatorID,
		"reviewed_at": now,
		"review_note": reviewNote,
		"updated_at":  now,
	}
	if err := tx.Model(&model.CourseSupplementRequests{}).Where("id = ?", request.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	notification := buildCourseSupplementReviewNotification(request.UserID, &model.CourseSupplementRequests{ID: request.ID}, false, reviewNote)
	if err := tx.Create(notification).Error; err != nil {
		return nil, err
	}

	return &repo.SupplementRequestItem{
		ID:          request.ID,
		UserID:      request.UserID,
		User:        request.User,
		RequestType: "course",
		Status:      string(model.SupplementRequestStatusRejected),
		Contact:     request.Contact,
		CourseName:  request.CourseName,
		CourseType:  request.CourseType,
		Remark:      request.Remark,
		ReviewedBy:  &operatorID,
		ReviewedAt:  &now,
		ReviewNote:  reviewNote,
		CreatedAt:   request.CreatedAt,
		UpdatedAt:   now,
	}, nil
}

func (s *MiscService) withSupplementWriteTx(fn func(*gorm.DB) error) error {
	if s.db == nil {
		return fn(nil)
	}
	return s.db.Transaction(fn)
}

func (s *MiscService) departmentExists(db *gorm.DB, id int16) bool {
	if db == nil {
		return true
	}
	var count int64
	if err := db.Model(&model.Departments{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func normalizeSupplementCourseType(value string) model.CourseType {
	switch strings.TrimSpace(value) {
	case "public", "公选课":
		return model.CourseTypePublic
	case "non_public", "非公选课":
		return model.CourseTypeNonPublic
	default:
		return ""
	}
}

func toSupplementRequestItemFromTeacher(item repo.TeacherSupplementRequestItem) repo.SupplementRequestItem {
	return repo.SupplementRequestItem{
		ID:                 item.ID,
		UserID:             item.UserID,
		User:               item.User,
		RequestType:        "teacher",
		Status:             item.Status,
		Contact:            item.Contact,
		TeacherName:        item.TeacherName,
		DepartmentID:       &item.DepartmentID,
		DepartmentName:     item.DepartmentName,
		Remark:             item.Remark,
		ReviewedBy:         item.ReviewedBy,
		ReviewedAt:         item.ReviewedAt,
		ReviewNote:         item.ReviewNote,
		ApprovedTargetType: approvedType(item.ApprovedTeacherID, "teacher"),
		ApprovedTargetID:   item.ApprovedTeacherID,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func toSupplementRequestItemFromCourse(item repo.CourseSupplementRequestItem) repo.SupplementRequestItem {
	return repo.SupplementRequestItem{
		ID:                 item.ID,
		UserID:             item.UserID,
		User:               item.User,
		RequestType:        "course",
		Status:             item.Status,
		Contact:            item.Contact,
		CourseName:         item.CourseName,
		CourseType:         item.CourseType,
		Remark:             item.Remark,
		ReviewedBy:         item.ReviewedBy,
		ReviewedAt:         item.ReviewedAt,
		ReviewNote:         item.ReviewNote,
		ApprovedTargetType: approvedType(item.ApprovedCourseID, "course"),
		ApprovedTargetID:   item.ApprovedCourseID,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func approvedType(id *int64, requestType string) string {
	if id == nil || *id == 0 {
		return ""
	}
	return requestType
}
