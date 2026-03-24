package repo

import (
	"csu-star-backend/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

type CommentListQuery struct {
	TargetType model.CommentTargetType
	TargetID   int64
	Sort       string
	Page       int
	Size       int
}

type CommentListItem struct {
	ID               int64             `json:"id"`
	TargetType       string            `json:"target_type"`
	TargetID         int64             `json:"target_id"`
	UserID           int64             `json:"user_id"`
	ParentID         int64             `json:"parent_id"`
	ReplyToCommentID int64             `json:"reply_to_comment_id"`
	Content          string            `json:"content"`
	LikeCount        int               `json:"like_count"`
	IsLiked          bool              `json:"is_liked"`
	Status           string            `json:"status"`
	AuthorName       string            `json:"author_name"`
	AuthorAvatarURL  string            `json:"author_avatar_url"`
	ReplyToUserID    int64             `json:"reply_to_user_id"`
	ReplyToUserName  string            `json:"reply_to_user_name"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Replies          []CommentListItem `json:"replies,omitempty"`
}

type CommentRepository interface {
	ListComments(query CommentListQuery) ([]CommentListItem, int64, error)
	CreateComment(comment *model.Comments) error
	CreateCommentWithEffects(comment *model.Comments, notification *model.Notifications) error
	GetCommentByID(id int64) (*model.Comments, error)
	GetCommentDetailByID(id int64) (*CommentListItem, error)
	SoftDeleteComment(id int64) error
	SoftDeleteCommentWithEffects(id int64, resourceID int64, updateResourceCount bool) error
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) ListComments(query CommentListQuery) ([]CommentListItem, int64, error) {
	var total int64
	base := r.db.Table("comments").Where(
		"comments.target_type = ? AND comments.target_id = ? AND comments.status = ?",
		query.TargetType, query.TargetID, model.CommentStatusActive,
	)

	topLevel := base.Where("comments.parent_id = 0 OR comments.parent_id IS NULL")
	if err := topLevel.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var parents []CommentListItem
	err := topLevel.
		Joins("JOIN users ON users.id = comments.user_id").
		Select(`
			comments.id,
			comments.target_type,
			comments.target_id,
			comments.user_id,
			comments.parent_id,
			comments.reply_to_comment_id,
			comments.content,
			comments.like_count,
			FALSE AS is_liked,
			comments.status,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			comments.created_at,
			comments.updated_at`).
		Order(commentSortExpr(query.Sort)).
		Order("comments.id DESC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&parents).Error
	if err != nil {
		return nil, 0, err
	}
	if len(parents) == 0 {
		return parents, total, nil
	}

	parentIDs := make([]int64, 0, len(parents))
	for _, item := range parents {
		parentIDs = append(parentIDs, item.ID)
	}

	var replies []CommentListItem
	err = base.Where("comments.parent_id IN ?", parentIDs).
		Joins("JOIN users ON users.id = comments.user_id").
		Joins("LEFT JOIN comments AS reply_to ON reply_to.id = comments.reply_to_comment_id").
		Joins("LEFT JOIN users AS reply_user ON reply_user.id = reply_to.user_id").
		Select(`
			comments.id,
			comments.target_type,
			comments.target_id,
			comments.user_id,
			comments.parent_id,
			comments.reply_to_comment_id,
			comments.content,
			comments.like_count,
			FALSE AS is_liked,
			comments.status,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			COALESCE(reply_user.id, 0) AS reply_to_user_id,
			COALESCE(reply_user.nickname, '') AS reply_to_user_name,
			comments.created_at,
			comments.updated_at`).
		Order(commentSortExpr(query.Sort)).
		Order("comments.id ASC").
		Scan(&replies).Error
	if err != nil {
		return nil, 0, err
	}

	replyMap := make(map[int64][]CommentListItem)
	for _, reply := range replies {
		replyMap[reply.ParentID] = append(replyMap[reply.ParentID], reply)
	}
	for i := range parents {
		parents[i].Replies = replyMap[parents[i].ID]
	}

	return parents, total, nil
}

func (r *commentRepository) CreateComment(comment *model.Comments) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) CreateCommentWithEffects(comment *model.Comments, notification *model.Notifications) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		if comment.TargetType == model.CommentTargetTypeResource {
			if err := tx.Model(&model.Resources{}).
				Where("id = ?", comment.TargetID).
				Update("comment_count", gorm.Expr(`(
					SELECT COUNT(*)
					FROM comments
					WHERE comments.target_type = ? AND comments.target_id = ? AND comments.status = ?
				)`, model.CommentTargetTypeResource, comment.TargetID, model.CommentStatusActive)).Error; err != nil {
				return err
			}
		}

		if notification != nil {
			if notification.RelatedID == 0 {
				notification.RelatedID = comment.ID
			}
			if err := tx.Create(notification).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *commentRepository) GetCommentByID(id int64) (*model.Comments, error) {
	var comment model.Comments
	err := r.db.First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *commentRepository) GetCommentDetailByID(id int64) (*CommentListItem, error) {
	var item CommentListItem
	err := r.db.Table("comments").
		Joins("JOIN users ON users.id = comments.user_id").
		Joins("LEFT JOIN comments AS reply_to ON reply_to.id = comments.reply_to_comment_id").
		Joins("LEFT JOIN users AS reply_user ON reply_user.id = reply_to.user_id").
		Select(`
			comments.id,
			comments.target_type,
			comments.target_id,
			comments.user_id,
			comments.parent_id,
			comments.reply_to_comment_id,
			comments.content,
			comments.like_count,
			FALSE AS is_liked,
			comments.status,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			COALESCE(reply_user.id, 0) AS reply_to_user_id,
			COALESCE(reply_user.nickname, '') AS reply_to_user_name,
			comments.created_at,
			comments.updated_at`).
		Where("comments.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func (r *commentRepository) SoftDeleteComment(id int64) error {
	now := time.Now()
	result := r.db.Model(&model.Comments{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     model.CommentStatusDeleted,
			"deleted_at": &now,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *commentRepository) SoftDeleteCommentWithEffects(id int64, resourceID int64, updateResourceCount bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&model.Comments{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     model.CommentStatusDeleted,
				"deleted_at": &now,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if updateResourceCount {
			if err := tx.Model(&model.Resources{}).
				Where("id = ?", resourceID).
				Update("comment_count", gorm.Expr(`(
					SELECT COUNT(*)
					FROM comments
					WHERE comments.target_type = ? AND comments.target_id = ? AND comments.status = ?
				)`, model.CommentTargetTypeResource, resourceID, model.CommentStatusActive)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func commentSortExpr(sort string) string {
	switch sort {
	case "likes":
		return "comments.like_count DESC"
	default:
		return "comments.created_at DESC"
	}
}

var ErrInvalidCommentReply = errors.New("invalid comment reply")
