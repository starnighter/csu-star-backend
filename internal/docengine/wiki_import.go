package docengine

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"fmt"
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

func wikiCatKey(id int64) string { return fmt.Sprintf("wiki_cat:%d", id) }
func wikiDocKey(id int64) string { return fmt.Sprintf("wiki_doc:%d", id) }

// isWikiRootIntro identifies space-level intro docs that must rank before college folders.
func isWikiRootIntro(title, slug string) bool {
	switch title {
	case "简介", "专业简介", "专业指北简介", "指南简介":
		return true
	}
	switch slug {
	case "index", "简介", "intro":
		return true
	}
	return false
}

// RepairAndImportPublishedWiki removes the broken first-generation import
// (pages that reused wiki row IDs and cross-linked majors → guides), then
// re-imports with stable external_key mapping and fresh snowflake page IDs.
func RepairAndImportPublishedWiki(db *gorm.DB) (imported int, err error) {
	if db == nil {
		return 0, nil
	}
	if err := purgeBrokenWikiImport(db); err != nil {
		return 0, err
	}
	return ImportPublishedWiki(db)
}

// purgeBrokenWikiImport deletes guides/majors pages from the bad import and
// any page whose external_key is a wiki_* source (safe re-import).
func purgeBrokenWikiImport(db *gorm.DB) error {
	// Collect ids to purge
	var ids []int64
	if err := db.Unscoped().Model(&model.CompassPage{}).
		Where("space_key IN ? OR external_key LIKE ? OR external_key LIKE ?",
			[]string{model.CompassSpaceGuides, model.CompassSpaceMajors},
			"wiki_cat:%", "wiki_doc:%").
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	// Legacy broken import used wiki serial IDs as page PKs with no external_key;
	// space_key filter above already covers those.
	if len(ids) == 0 {
		// Also catch legacy broken rows: space still guides/majors after partial purge
		return nil
	}
	if err := db.Unscoped().Where("page_id IN ?", ids).Delete(&model.CompassPageHistory{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("page_id IN ?", ids).Delete(&model.CompassPageWriter{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("page_id IN ?", ids).Delete(&model.CompassComment{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("page_id IN ?", ids).Delete(&model.CompassEditRequest{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("id IN ?", ids).Delete(&model.CompassPage{}).Error; err != nil {
		return err
	}
	if logger.Log != nil {
		logger.Log.Info("purged broken compass wiki import pages", zap.Int("count", len(ids)))
	}
	return nil
}

// compassCategorySortBase keeps wiki category folders after space-level root
// docs (e.g. 「简介」). Same sort_order on cat+intro previously put 计算机学院 first.
const compassCategorySortBase = 100_000

// ImportPublishedWiki copies published wiki_documents (+ category folders) into
// compass_pages. Uses external_key for idempotency; never reuses wiki row IDs
// as compass page primary keys.
func ImportPublishedWiki(db *gorm.DB) (imported int, err error) {
	if db == nil {
		return 0, nil
	}

	ownerID := systemOwnerID(db)
	catPageID := make(map[int64]int64) // wiki category id → compass page id

	var categories []model.WikiCategories
	if err := db.Order("sort_order ASC, id ASC").Find(&categories).Error; err != nil {
		return 0, err
	}

	for _, c := range categories {
		mapping, ok := wikiSectionToCompass[string(c.Section)]
		if !ok {
			continue
		}
		key := wikiCatKey(c.ID)
		pageID, err := upsertByExternalKey(db, key, model.CompassPage{
			SpaceKey:    mapping.Space,
			OwnerID:     ownerID,
			ContentType: mapping.CType,
			Title:       c.Name,
			Body:        "",
			// Folders always sort after root docs in the same space.
			SortOrder:   compassCategorySortBase + c.SortOrder,
			PublishedAt: time.Now(),
		})
		if err != nil {
			return imported, err
		}
		catPageID[c.ID] = pageID
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
			if pid, exists := catPageID[*d.CategoryID]; exists {
				parentID = &pid
			}
		}
		// Root intro docs must stay at the top of the space tree.
		docSort := d.SortOrder
		if d.CategoryID == nil && isWikiRootIntro(d.Title, d.Slug) {
			docSort = 0
		}
		published := time.Now()
		if d.PublishedAt != nil {
			published = *d.PublishedAt
		}
		key := wikiDocKey(d.ID)
		if _, err := upsertByExternalKey(db, key, model.CompassPage{
			SpaceKey:    mapping.Space,
			ParentID:    parentID,
			OwnerID:     ownerID,
			ContentType: mapping.CType,
			Title:       d.Title,
			Body:        d.Content,
			SortOrder:   docSort,
			PublishedAt: published,
		}); err != nil {
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

// upsertByExternalKey creates or updates a page keyed by external_key.
// Returns the compass page id.
func upsertByExternalKey(db *gorm.DB, externalKey string, page model.CompassPage) (int64, error) {
	now := time.Now()
	if page.PublishedAt.IsZero() {
		page.PublishedAt = now
	}
	ek := externalKey
	page.ExternalKey = &ek

	var existing model.CompassPage
	err := db.Unscoped().Where("external_key = ?", externalKey).Limit(1).Find(&existing).Error
	if err != nil {
		return 0, err
	}
	if existing.ID != 0 {
		if err := db.Unscoped().Model(&model.CompassPage{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"space_key":    page.SpaceKey,
			"parent_id":    page.ParentID,
			"owner_id":     page.OwnerID,
			"content_type": page.ContentType,
			"title":        page.Title,
			"body":         page.Body,
			"sort_order":   page.SortOrder,
			"published_at": page.PublishedAt,
			"external_key": ek,
			"deleted_at":   nil,
			"updated_at":   now,
		}).Error; err != nil {
			return 0, err
		}
		return existing.ID, nil
	}

	if page.ID == 0 {
		page.ID = utils.GenerateID()
	}
	page.CreatedAt = now
	page.UpdatedAt = now
	page.HotScore = 0
	if err := db.Create(&page).Error; err != nil {
		return 0, err
	}

	var histCount int64
	_ = db.Model(&model.CompassPageHistory{}).Where("page_id = ?", page.ID).Count(&histCount).Error
	if histCount == 0 {
		h := model.CompassPageHistory{
			ID: utils.GenerateID(), PageID: page.ID, EditorID: page.OwnerID,
			Title: page.Title, Body: page.Body,
		}
		if err := db.Create(&h).Error; err != nil {
			return page.ID, err
		}
	}
	return page.ID, nil
}
