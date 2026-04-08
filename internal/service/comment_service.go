package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"errors"

	"gorm.io/gorm"
)

var (
	ErrCommentTargetNotFound = errors.New("comment target not found")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrCommentForbidden      = errors.New("comment forbidden")
	ErrCommentReplyInvalid   = errors.New("comment reply invalid")
)

type CommentService struct {
	commentRepo  repo.CommentRepository
	teacherRepo  repo.TeacherRepository
	courseRepo   repo.CourseRepository
	resourceRepo repo.ResourceRepository
	socialRepo   repo.SocialRepository
}

func NewCommentService(cr repo.CommentRepository, tr repo.TeacherRepository, cor repo.CourseRepository, rr repo.ResourceRepository, sr repo.SocialRepository) *CommentService {
	return &CommentService{
		commentRepo:  cr,
		teacherRepo:  tr,
		courseRepo:   cor,
		resourceRepo: rr,
		socialRepo:   sr,
	}
}

func (s *CommentService) ListComments(targetType model.CommentTargetType, targetID int64, sort string, page, size int, userID int64) ([]repo.CommentListItem, int64, error) {
	fillPagination(&page, &size)
	if sort == "" {
		sort = "created_at"
	}

	ok, err := s.targetExists(targetType, targetID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, ErrCommentTargetNotFound
	}

	items, total, err := s.commentRepo.ListComments(repo.CommentListQuery{
		TargetType: targetType,
		TargetID:   targetID,
		Sort:       sort,
		Page:       page,
		Size:       size,
	})
	if err != nil {
		return nil, 0, err
	}
	if userID > 0 && s.socialRepo != nil {
		commentIDs := make([]int64, 0, len(items))
		replyIDs := make([]int64, 0, len(items))
		for _, item := range items {
			commentIDs = append(commentIDs, item.ID)
			for _, reply := range item.Replies {
				replyIDs = append(replyIDs, reply.ID)
			}
		}
		commentLikes, err := s.socialRepo.ListLikedTargetIDs(userID, model.LikeTargetTypeComment, commentIDs)
		if err != nil {
			return nil, 0, err
		}
		replyLikes, err := s.socialRepo.ListLikedTargetIDs(userID, model.LikeTargetTypeComment, replyIDs)
		if err != nil {
			return nil, 0, err
		}
		for i := range items {
			items[i].IsLiked = commentLikes[items[i].ID]
			for j := range items[i].Replies {
				items[i].Replies[j].IsLiked = replyLikes[items[i].Replies[j].ID]
			}
		}
	}
	s.normalizeCommentUsers(items)
	return items, total, nil
}

func (s *CommentService) CreateComment(userID int64, targetType model.CommentTargetType, targetID int64, content string, parentID, replyToCommentID int64) (*repo.CommentListItem, error) {
	ok, err := s.targetExists(targetType, targetID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCommentTargetNotFound
	}

	var parent *model.Comments
	if parentID > 0 {
		parent, err = s.commentRepo.GetCommentByID(parentID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentReplyInvalid
		}
		if err != nil {
			return nil, err
		}
		if parent.TargetType != targetType || parent.TargetID != targetID || parent.Status != model.CommentStatusActive {
			return nil, ErrCommentReplyInvalid
		}
		if commentIDValue(parent.ParentID) > 0 {
			return nil, ErrCommentReplyInvalid
		}
	}

	if replyToCommentID > 0 {
		replyTo, err := s.commentRepo.GetCommentByID(replyToCommentID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentReplyInvalid
		}
		if err != nil {
			return nil, err
		}
		if replyTo.TargetType != targetType || replyTo.TargetID != targetID || replyTo.Status != model.CommentStatusActive {
			return nil, ErrCommentReplyInvalid
		}
		if parent == nil {
			return nil, ErrCommentReplyInvalid
		}
		if replyTo.ID != parent.ID && commentIDValue(replyTo.ParentID) != parent.ID {
			return nil, ErrCommentReplyInvalid
		}
	} else if parent != nil {
		replyToCommentID = parent.ID
	}

	comment := &model.Comments{
		TargetType:       targetType,
		TargetID:         targetID,
		UserID:           userID,
		ParentID:         nullableCommentID(parentID),
		ReplyToCommentID: nullableCommentID(replyToCommentID),
		Content:          content,
		Status:           model.CommentStatusActive,
	}

	recipientID := int64(0)
	title := "收到新的评论"
	contentText := "你的内容收到了新的评论。"

	if replyToCommentID > 0 {
		replyTo, err := s.commentRepo.GetCommentByID(replyToCommentID)
		if err == nil {
			recipientID = replyTo.UserID
			title = "收到新的回复"
			contentText = "你的评论收到了新的回复。"
		}
	} else if targetType == model.CommentTargetTypeResource && s.socialRepo != nil {
		ownerID, err := s.socialRepo.GetResourceOwnerID(targetID)
		if err == nil {
			recipientID = ownerID
			contentText = "你的资源收到了新的评论。"
		}
	}

	var notification *model.Notifications
	if recipientID > 0 && recipientID != userID && s.socialRepo != nil {
		notification = &model.Notifications{
			UserID:    recipientID,
			Type:      model.NotificationTypeCommented,
			Category:  model.NotificationCategoryInteraction,
			Result:    model.NotificationResultInform,
			Title:     title,
			Content:   contentText,
			IsRead:    false,
			IsGlobal:  false,
			RelatedID: targetID,
			Metadata:  buildInteractionMetadata(string(targetType), targetID, buildResourceInteractionRoute(targetID)),
		}
	}

	if err := s.commentRepo.CreateCommentWithEffects(comment, notification); err != nil {
		return nil, err
	}
	if targetType == model.CommentTargetTypeResource {
	}

	item, err := s.commentRepo.GetCommentDetailByID(comment.ID)
	if err != nil {
		return nil, err
	}
	s.normalizeCommentItem(item)
	return item, nil
}

func (s *CommentService) UpdateComment(userID int64, commentID int64, content string) (*repo.CommentListItem, error) {
	comment, err := s.commentRepo.GetCommentByID(commentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	if comment.Status != model.CommentStatusActive {
		return nil, ErrCommentNotFound
	}
	if comment.UserID != userID {
		return nil, ErrCommentForbidden
	}
	if err := s.commentRepo.UpdateCommentContent(commentID, content); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	item, err := s.commentRepo.GetCommentDetailByID(commentID)
	if err != nil {
		return nil, err
	}
	s.normalizeCommentItem(item)
	return item, nil
}

func (s *CommentService) DeleteComment(userID int64, userRole string, commentID int64) error {
	comment, err := s.commentRepo.GetCommentByID(commentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCommentNotFound
	}
	if err != nil {
		return err
	}
	if comment.Status != model.CommentStatusActive {
		return ErrCommentNotFound
	}
	allowed := comment.UserID == userID || isPrivilegedRole(userRole)
	if !allowed && comment.TargetType == model.CommentTargetTypeResource && s.socialRepo != nil {
		ownerID, err := s.socialRepo.GetResourceOwnerID(comment.TargetID)
		if err == nil && ownerID == userID {
			allowed = true
		}
	}
	if !allowed {
		return ErrCommentForbidden
	}
	if err := s.commentRepo.SoftDeleteCommentWithEffects(commentID, comment.TargetID, comment.TargetType == model.CommentTargetTypeResource); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCommentNotFound
		}
		return err
	}
	if comment.TargetType == model.CommentTargetTypeResource {
	}
	return nil
}

func (s *CommentService) normalizeCommentUsers(items []repo.CommentListItem) {
	for i := range items {
		s.normalizeCommentItem(&items[i])
	}
}

func nullableCommentID(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	value := id
	return &value
}

func commentIDValue(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func (s *CommentService) normalizeCommentItem(item *repo.CommentListItem) {
	if item == nil {
		return
	}
	item.User = &repo.UserBrief{
		ID:        item.UserID,
		Nickname:  item.AuthorName,
		AvatarURL: item.AuthorAvatarURL,
	}
	if item.ReplyToUserID > 0 {
		item.ReplyToUser = &repo.UserBrief{
			ID:       item.ReplyToUserID,
			Nickname: item.ReplyToUserName,
		}
	}
	for i := range item.Replies {
		s.normalizeCommentItem(&item.Replies[i])
	}
}

func (s *CommentService) targetExists(targetType model.CommentTargetType, targetID int64) (bool, error) {
	switch targetType {
	case model.CommentTargetTypeTeacher:
		return s.teacherRepo.TeacherExists(targetID)
	case model.CommentTargetTypeCourse:
		return s.courseRepo.CourseExists(targetID)
	case model.CommentTargetTypeResource:
		return s.resourceRepo.ResourceExists(targetID)
	default:
		return false, nil
	}
}
