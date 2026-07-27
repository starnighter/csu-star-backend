package docengine

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/logger"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// wikiSectionToCompass maps wiki_sections.key → compass space + content type.
var wikiSectionToCompass = map[string]struct {
	Space string
	CType string
}{
	string(model.WikiSectionCompass): {Space: model.CompassSpaceGuides, CType: model.CompassContentGuide},
	string(model.WikiSectionMajor):   {Space: model.CompassSpaceMajors, CType: model.CompassContentMajor},
}

// ImportPublishedWiki copies published wiki_documents (+ category folders) into
// compass_pages so guides/majors use the same engine as essays/courses.
// Idempotent: re-run updates title/body/tree; preserves view counters.
func ImportPublishedWiki(db *gorm.DB) (imported int, err error) {
	if db == nil {
		return 0, nil
	}

	ownerID := systemOwnerID(db)

	var categories []model.WikiCategories
	if err := db.Order("sort_order ASC, id ASC").Find(&categories).Error; err != nil {
		return 0, err
	}
	catByID := make(map[int64]model.WikiCategories, len(categories))
	for _, c := range categories {
		catByID[c.ID] = c
	}

	for _, c := range categories {
		mapping, ok := wikiSectionToCompass[string(c.Section)]
		if !ok {
			continue
		}
		page := model.CompassPage{
			ID:          c.ID,
			SpaceKey:    mapping.Space,
			OwnerID:     ownerID,
			ContentType: mapping.CType,
			Title:       c.Name,
			Body:        "",
			SortOrder:   c.SortOrder,
			PublishedAt: time.Now(),
		}
		if err := upsertCompassPage(db, &page); err != nil {
			return imported, err
		}
		imported++
	}

	var docs []model.WikiDocuments
	if err := db.Where("is_published = TRUE").
		Order("section ASC, category_id ASC NULLS FIRST, sort_order ASC, id ASC").
		Find(&docs).Error; err != nil {
		return imported, err
	}

	for _, d := range docs {
		mapping, ok := wikiSectionToCompass[string(d.Section)]
		if !ok {
			continue
		}
		var parentID *int64
		if d.CategoryID != nil {
			if _, exists := catByID[*d.CategoryID]; exists {
				pid := *d.CategoryID
				parentID = &pid
			}
		}
		published := time.Now()
		if d.PublishedAt != nil {
			published = *d.PublishedAt
		}
		page := model.CompassPage{
			ID:          d.ID,
			SpaceKey:    mapping.Space,
			ParentID:    parentID,
			OwnerID:     ownerID,
			ContentType: mapping.CType,
			Title:       d.Title,
			Body:        d.Content,
			SortOrder:   d.SortOrder,
			PublishedAt: published,
		}
		if err := upsertCompassPage(db, &page); err != nil {
			return imported, err
		}
		imported++
	}

	if logger.Log != nil {
		logger.Log.Info("compass wiki import finished", zap.Int("pages_upserted", imported))
	}
	return imported, nil
}

func systemOwnerID(db *gorm.DB) int64 {
	var id int64
	_ = db.Model(&model.Users{}).
		Where("role IN ?", []string{string(model.UserRoleAdmin), string(model.UserRoleAuditor)}).
		Order("id ASC").
		Limit(1).
		Pluck("id", &id).Error
	if id != 0 {
		return id
	}
	_ = db.Model(&model.Users{}).Order("id ASC").Limit(1).Pluck("id", &id).Error
	if id != 0 {
		return id
	}
	return 1
}

func upsertCompassPage(db *gorm.DB, page *model.CompassPage) error {
	now := time.Now()
	if page.PublishedAt.IsZero() {
		page.PublishedAt = now
	}

	var existing model.CompassPage
	if err := db.Unscoped().Where("id = ?", page.ID).Limit(1).Find(&existing).Error; err != nil {
		return err
	}
	if existing.ID != 0 {
		return db.Unscoped().Model(&model.CompassPage{}).Where("id = ?", page.ID).Updates(map[string]any{
			"space_key":    page.SpaceKey,
			"parent_id":    page.ParentID,
			"owner_id":     page.OwnerID,
			"content_type": page.ContentType,
			"title":        page.Title,
			"body":         page.Body,
			"sort_order":   page.SortOrder,
			"published_at": page.PublishedAt,
			"deleted_at":   nil,
			"updated_at":   now,
		}).Error
	}

	page.CreatedAt = now
	page.UpdatedAt = now
	page.HotScore = 0
	if err := db.Create(page).Error; err != nil {
		// race: another worker created it
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
			strings.Contains(err.Error(), "UNIQUE") {
			return db.Model(&model.CompassPage{}).Where("id = ?", page.ID).Updates(map[string]any{
				"space_key": page.SpaceKey, "parent_id": page.ParentID, "title": page.Title,
				"body": page.Body, "sort_order": page.SortOrder, "content_type": page.ContentType,
				"updated_at": now,
			}).Error
		}
		return err
	}

	var histCount int64
	_ = db.Model(&model.CompassPageHistory{}).Where("page_id = ?", page.ID).Count(&histCount).Error
	if histCount == 0 {
		// history id: page.ID based offset avoids requiring snowflake in import path
		h := model.CompassPageHistory{
			ID: page.ID, // one initial snapshot per page; unique with page_id later entries use snowflake
			PageID: page.ID, EditorID: page.OwnerID, Title: page.Title, Body: page.Body,
		}
		return db.Create(&h).Error
	}
	return nil
}
