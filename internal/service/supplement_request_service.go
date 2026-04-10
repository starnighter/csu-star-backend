package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	relatedCourseName string,
	relatedCourseIDs []string,
	relatedCourseNames []string,
	relatedTeacherIDs []string,
	relatedTeacherNames []string,
	courseName, courseType, remark string,
) (*repo.SupplementRequestItem, error) {
	normalizedCourseIDs, normalizedCourseNames, err := normalizeSupplementRelationPairs(relatedCourseIDs, relatedCourseNames)
	if err != nil {
		return nil, ErrSupplementRequestInvalidPayload
	}
	normalizedTeacherIDs, normalizedTeacherNames, err := normalizeSupplementRelationPairs(relatedTeacherIDs, relatedTeacherNames)
	if err != nil {
		return nil, ErrSupplementRequestInvalidPayload
	}

	request := &model.SupplementRequests{
		UserID:              userID,
		RequestType:         model.SupplementRequestType(strings.TrimSpace(requestType)),
		Status:              model.SupplementRequestStatusPending,
		Contact:             strings.TrimSpace(contact),
		TeacherName:         strings.TrimSpace(teacherName),
		DepartmentID:        departmentID,
		RelatedCourseName:   strings.TrimSpace(relatedCourseName),
		RelatedCourseIDs:    mustJSON(normalizedCourseIDs),
		RelatedCourseNames:  mustJSON(normalizedCourseNames),
		RelatedTeacherIDs:   mustJSON(normalizedTeacherIDs),
		RelatedTeacherNames: mustJSON(normalizedTeacherNames),
		CourseName:          strings.TrimSpace(courseName),
		CourseType:          strings.TrimSpace(courseType),
		Remark:              strings.TrimSpace(remark),
	}

	if err := s.validateSupplementRequest(request); err != nil {
		return nil, err
	}

	if err := s.miscRepo.CreateSupplementRequest(request); err != nil {
		return nil, err
	}

	return s.miscRepo.GetSupplementRequestByID(request.ID)
}

func (s *MiscService) ListSupplementRequests(query repo.SupplementRequestListQuery) ([]repo.SupplementRequestItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	return s.miscRepo.ListSupplementRequests(query)
}

func (s *MiscService) GetSupplementRequest(id int64) (*repo.SupplementRequestItem, error) {
	item, err := s.miscRepo.GetSupplementRequestByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSupplementRequestNotFound
	}
	return item, err
}

func (s *MiscService) ApproveSupplementRequest(operatorID, requestID int64, reviewNote string) (*repo.SupplementRequestItem, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var request model.SupplementRequests
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", requestID).
			First(&request).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSupplementRequestNotFound
			}
			return err
		}

		if request.Status != model.SupplementRequestStatusPending {
			return ErrSupplementRequestAlreadyReviewed
		}

		approvedTargetType := ""
		var approvedTargetID int64
		switch request.RequestType {
		case model.SupplementRequestTypeTeacher:
			targetID, err := s.resolveTeacherTarget(tx, request.TeacherName, request.DepartmentID)
			if err != nil {
				return err
			}
			approvedTargetType = string(model.SupplementRequestTypeTeacher)
			approvedTargetID = targetID
		case model.SupplementRequestTypeCourse:
			targetID, err := s.resolveCourseTarget(tx, request.CourseName, request.CourseType)
			if err != nil {
				return err
			}
			approvedTargetType = string(model.SupplementRequestTypeCourse)
			approvedTargetID = targetID
		default:
			return ErrSupplementRequestInvalidPayload
		}

		now := time.Now()
		request.Status = model.SupplementRequestStatusApproved
		request.ReviewedBy = &operatorID
		request.ReviewedAt = &now
		request.ReviewNote = strings.TrimSpace(reviewNote)
		request.ApprovedTargetType = approvedTargetType
		request.ApprovedTargetID = &approvedTargetID

		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "supplement_request",
			TargetID:   request.ID,
			OldValues:  mustJSON(map[string]interface{}{"status": model.SupplementRequestStatusPending}),
			NewValues: mustJSON(map[string]interface{}{
				"status":               model.SupplementRequestStatusApproved,
				"approved_target_type": approvedTargetType,
				"approved_target_id":   approvedTargetID,
			}),
			Reason: request.ReviewNote,
		}).Error; err != nil {
			return err
		}

		return tx.Create(buildSupplementReviewNotification(
			request.UserID,
			&request,
			true,
			s.buildSupplementReviewNotificationContent(&request, true),
		)).Error
	})
	if err != nil {
		return nil, err
	}

	return s.miscRepo.GetSupplementRequestByID(requestID)
}

func (s *MiscService) RejectSupplementRequest(operatorID, requestID int64, reviewNote string) (*repo.SupplementRequestItem, error) {
	reviewNote = strings.TrimSpace(reviewNote)
	if reviewNote == "" {
		return nil, ErrSupplementRequestReviewNoteMissing
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var request model.SupplementRequests
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", requestID).
			First(&request).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSupplementRequestNotFound
			}
			return err
		}

		if request.Status != model.SupplementRequestStatusPending {
			return ErrSupplementRequestAlreadyReviewed
		}

		now := time.Now()
		request.Status = model.SupplementRequestStatusRejected
		request.ReviewedBy = &operatorID
		request.ReviewedAt = &now
		request.ReviewNote = reviewNote

		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionReject,
			TargetType: "supplement_request",
			TargetID:   request.ID,
			OldValues:  mustJSON(map[string]interface{}{"status": model.SupplementRequestStatusPending}),
			NewValues:  mustJSON(map[string]interface{}{"status": model.SupplementRequestStatusRejected}),
			Reason:     request.ReviewNote,
		}).Error; err != nil {
			return err
		}

		return tx.Create(buildSupplementReviewNotification(
			request.UserID,
			&request,
			false,
			s.buildSupplementReviewNotificationContent(&request, false),
		)).Error
	})
	if err != nil {
		return nil, err
	}

	return s.miscRepo.GetSupplementRequestByID(requestID)
}

func (s *MiscService) validateSupplementRequest(request *model.SupplementRequests) error {
	if strings.TrimSpace(request.Contact) == "" {
		return ErrSupplementRequestInvalidPayload
	}

	switch request.RequestType {
	case model.SupplementRequestTypeTeacher:
		if strings.TrimSpace(request.TeacherName) == "" || request.DepartmentID == nil {
			return ErrSupplementRequestInvalidPayload
		}
		if !s.departmentExists(*request.DepartmentID) {
			return ErrSupplementRequestInvalidPayload
		}
		if strings.TrimSpace(request.RelatedCourseName) != "" {
			return ErrSupplementRequestInvalidPayload
		}
		normalizedCourseIDs, normalizedCourseNames, err := normalizeSupplementRelationPairsFromJSON(
			request.RelatedCourseIDs,
			request.RelatedCourseNames,
		)
		if err != nil {
			return ErrSupplementRequestInvalidPayload
		}
		request.RelatedCourseIDs = mustJSON(normalizedCourseIDs)
		request.RelatedCourseNames = mustJSON(normalizedCourseNames)
		request.RelatedCourseName = ""
		request.RelatedTeacherIDs = mustJSON([]int64{})
		request.RelatedTeacherNames = mustJSON([]string{})
		request.CourseName = ""
		request.CourseType = ""
	case model.SupplementRequestTypeCourse:
		if strings.TrimSpace(request.CourseName) == "" || normalizeSupplementCourseType(request.CourseType) == "" {
			return ErrSupplementRequestInvalidPayload
		}
		normalizedTeacherIDs, normalizedTeacherNames, err := normalizeSupplementRelationPairsFromJSON(
			request.RelatedTeacherIDs,
			request.RelatedTeacherNames,
		)
		if err != nil {
			return ErrSupplementRequestInvalidPayload
		}
		request.TeacherName = ""
		request.DepartmentID = nil
		request.RelatedCourseName = ""
		request.RelatedCourseIDs = mustJSON([]int64{})
		request.RelatedCourseNames = mustJSON([]string{})
		request.RelatedTeacherIDs = mustJSON(normalizedTeacherIDs)
		request.RelatedTeacherNames = mustJSON(normalizedTeacherNames)
		request.CourseType = normalizeSupplementCourseType(request.CourseType)
	default:
		return ErrSupplementRequestInvalidPayload
	}

	return nil
}

func (s *MiscService) resolveTeacherTarget(tx *gorm.DB, teacherName string, departmentID *int16) (int64, error) {
	if departmentID == nil || strings.TrimSpace(teacherName) == "" || !s.departmentExistsTx(tx, *departmentID) {
		return 0, ErrSupplementRequestInvalidPayload
	}

	var teacher model.Teachers
	err := tx.Where("LOWER(name) = LOWER(?) AND department_id = ? AND status = ?", strings.TrimSpace(teacherName), *departmentID, model.CourseStatusActive).
		First(&teacher).Error
	switch {
	case err == nil:
		return teacher.ID, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return 0, err
	}

	err = tx.Where("LOWER(name) = LOWER(?) AND department_id = ? AND status = ?", strings.TrimSpace(teacherName), *departmentID, model.CourseStatusDeleted).
		First(&teacher).Error
	switch {
	case err == nil:
		if err := tx.Model(&teacher).Updates(map[string]interface{}{
			"status":     model.CourseStatusActive,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return 0, err
		}
		return teacher.ID, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return 0, err
	}

	teacher = model.Teachers{
		Name:         strings.TrimSpace(teacherName),
		DepartmentID: *departmentID,
	}
	if err := tx.Create(&teacher).Error; err != nil {
		return 0, err
	}
	return teacher.ID, nil
}

func (s *MiscService) resolveCourseTarget(tx *gorm.DB, courseName, courseType string) (int64, error) {
	normalizedCourseType := normalizeSupplementCourseType(courseType)
	if strings.TrimSpace(courseName) == "" || normalizedCourseType == "" {
		return 0, ErrSupplementRequestInvalidPayload
	}

	var course model.Courses
	err := tx.Where("LOWER(name) = LOWER(?) AND course_type = ? AND status = ?", strings.TrimSpace(courseName), model.CourseType(normalizedCourseType), model.CourseStatusActive).
		First(&course).Error
	switch {
	case err == nil:
		return course.ID, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return 0, err
	}

	err = tx.Where("LOWER(name) = LOWER(?) AND course_type = ? AND status = ?", strings.TrimSpace(courseName), model.CourseType(normalizedCourseType), model.CourseStatusDeleted).
		First(&course).Error
	switch {
	case err == nil:
		if err := tx.Model(&course).Updates(map[string]interface{}{
			"status":     model.CourseStatusActive,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return 0, err
		}
		return course.ID, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return 0, err
	}

	course = model.Courses{
		Name:       strings.TrimSpace(courseName),
		CourseType: model.CourseType(normalizedCourseType),
	}
	if err := tx.Create(&course).Error; err != nil {
		return 0, err
	}
	return course.ID, nil
}

func (s *MiscService) departmentExists(id int16) bool {
	return s.departmentExistsTx(s.db, id)
}

func (s *MiscService) departmentExistsTx(db *gorm.DB, id int16) bool {
	if db == nil {
		return true
	}
	var count int64
	if err := db.Model(&model.Departments{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func normalizeSupplementCourseType(value string) string {
	switch strings.TrimSpace(value) {
	case "公选课", string(model.CourseTypePublic):
		return string(model.CourseTypePublic)
	case "非公选课", string(model.CourseTypeNonPublic):
		return string(model.CourseTypeNonPublic)
	default:
		return ""
	}
}

func (s *MiscService) buildSupplementReviewNotificationContent(request *model.SupplementRequests, approved bool) string {
	targetName := strings.TrimSpace(request.TeacherName)
	if request.RequestType == model.SupplementRequestTypeCourse {
		targetName = strings.TrimSpace(request.CourseName)
	}

	if approved {
		if targetName == "" {
			return "你提交的补录申请已通过审核。"
		}
		return "你提交的补录申请已通过审核：" + targetName
	}

	if targetName == "" {
		return "你提交的补录申请未通过审核。原因：" + request.ReviewNote
	}
	return "你提交的补录申请未通过审核：" + targetName + "。原因：" + request.ReviewNote
}

func mustJSON(value interface{}) datatypes.JSON {
	raw, _ := json.Marshal(value)
	return datatypes.JSON(raw)
}

func normalizeSupplementNames(names []string) ([]string, error) {
	filtered := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, item := range names {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if len([]rune(trimmed)) > 128 {
			return nil, ErrSupplementRequestInvalidPayload
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		filtered = append(filtered, trimmed)
	}
	if len(filtered) > 10 {
		return nil, ErrSupplementRequestInvalidPayload
	}
	return filtered, nil
}

func normalizeSupplementRelationPairs(ids []string, names []string) ([]int64, []string, error) {
	if len(ids) == 0 && len(names) == 0 {
		return []int64{}, []string{}, nil
	}
	if len(ids) == 0 || len(names) == 0 || len(ids) != len(names) {
		return nil, nil, ErrSupplementRequestInvalidPayload
	}

	normalizedIDs := make([]int64, 0, len(ids))
	normalizedNames := make([]string, 0, len(names))
	seen := make(map[int64]string, len(ids))

	for index := range ids {
		normalizedIDList, err := normalizeSupplementIDs([]string{ids[index]})
		if err != nil || len(normalizedIDList) != 1 {
			return nil, nil, ErrSupplementRequestInvalidPayload
		}
		normalizedNameList, err := normalizeSupplementNames([]string{names[index]})
		if err != nil || len(normalizedNameList) != 1 {
			return nil, nil, ErrSupplementRequestInvalidPayload
		}

		id := normalizedIDList[0]
		name := normalizedNameList[0]
		if existingName, ok := seen[id]; ok {
			if existingName != name {
				return nil, nil, ErrSupplementRequestInvalidPayload
			}
			continue
		}

		seen[id] = name
		normalizedIDs = append(normalizedIDs, id)
		normalizedNames = append(normalizedNames, name)
	}

	return normalizedIDs, normalizedNames, nil
}

func normalizeSupplementIDs(values []string) ([]int64, error) {
	filtered := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		id, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil || id <= 0 {
			return nil, ErrSupplementRequestInvalidPayload
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		filtered = append(filtered, id)
	}
	if len(filtered) > 10 {
		return nil, ErrSupplementRequestInvalidPayload
	}
	return filtered, nil
}

func normalizeSupplementNamesFromJSON(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}

	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, err
	}
	return normalizeSupplementNames(names)
}

func normalizeSupplementRelationPairsFromJSON(rawIDs, rawNames datatypes.JSON) ([]int64, []string, error) {
	ids, err := normalizeSupplementIDsFromJSON(rawIDs)
	if err != nil {
		return nil, nil, err
	}
	names, err := normalizeSupplementNamesFromJSON(rawNames)
	if err != nil {
		return nil, nil, err
	}

	stringIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		stringIDs = append(stringIDs, mustInt64String(id))
	}
	return normalizeSupplementRelationPairs(stringIDs, names)
}

func normalizeSupplementIDsFromJSON(raw datatypes.JSON) ([]int64, error) {
	if len(raw) == 0 {
		return []int64{}, nil
	}

	var ids []int64
	if err := json.Unmarshal(raw, &ids); err == nil {
		stringified := make([]string, 0, len(ids))
		for _, id := range ids {
			stringified = append(stringified, mustInt64String(id))
		}
		return normalizeSupplementIDs(stringified)
	}

	var stringIDs []string
	if err := json.Unmarshal(raw, &stringIDs); err != nil {
		return nil, err
	}
	return normalizeSupplementIDs(stringIDs)
}

func normalizeSupplementTeacherNames(names []string) ([]string, error) {
	return normalizeSupplementNames(names)
}

func normalizeSupplementTeacherNamesFromJSON(raw datatypes.JSON) ([]string, error) {
	return normalizeSupplementNamesFromJSON(raw)
}

func mustInt64String(value int64) string {
	return strconv.FormatInt(value, 10)
}
