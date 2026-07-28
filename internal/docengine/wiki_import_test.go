package docengine

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/pkg/utils"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestImportPublishedWikiUsesExternalKeysNotWikiIDs(t *testing.T) {
	utils.InitSnowflake(1)

	db, err := gorm.Open(sqlite.Open("file:compass_import2?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC().Truncate(time.Second)
		},
	})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}

	if err := db.Exec(`
CREATE TABLE users (id INTEGER PRIMARY KEY, role TEXT, status TEXT);
CREATE TABLE wiki_categories (
  id INTEGER PRIMARY KEY, section TEXT NOT NULL, name TEXT NOT NULL,
  sort_order INTEGER DEFAULT 0, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
);
CREATE TABLE wiki_documents (
  id INTEGER PRIMARY KEY, section TEXT NOT NULL, category_id INTEGER, slug TEXT NOT NULL,
  title TEXT NOT NULL, content TEXT NOT NULL, sort_order INTEGER DEFAULT 0,
  is_published INTEGER DEFAULT 0, published_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
);
`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.CompassPage{},
		&model.CompassPageHistory{},
		&model.CompassPageWriter{},
		&model.CompassComment{},
		&model.CompassEditRequest{},
	); err != nil {
		t.Fatal(err)
	}
	_ = db.Exec(`INSERT INTO users (id, role, status) VALUES (9, 'admin', 'active')`)

	// Simulate broken first import: wiki cat id=1 and guide doc id=1 collide
	_ = db.Exec(`INSERT INTO wiki_categories (id, section, name, sort_order) VALUES (1, 'major', '计算机学院', 1)`)
	_ = db.Exec(`INSERT INTO wiki_documents (id, section, slug, title, content, sort_order, is_published, published_at)
		VALUES (1, 'compass', 'intro', '简介', '指南简介', 1, 1, CURRENT_TIMESTAMP)`)
	_ = db.Exec(`INSERT INTO wiki_documents (id, section, category_id, slug, title, content, sort_order, is_published, published_at)
		VALUES (13, 'major', 1, 'se', '软件工程', '# 软工', 1, 1, CURRENT_TIMESTAMP)`)

	// Seed broken rows like production
	_ = db.Create(&model.CompassPage{ID: 1, SpaceKey: model.CompassSpaceGuides, OwnerID: 9,
		ContentType: model.CompassContentGuide, Title: "简介", Body: "指南简介"}).Error
	_ = db.Create(&model.CompassPage{ID: 13, SpaceKey: model.CompassSpaceMajors, OwnerID: 9,
		ContentType: model.CompassContentMajor, Title: "软件工程", Body: "# 软工", ParentID: int64Ptr(1)}).Error

	n, err := RepairAndImportPublishedWiki(db)
	if err != nil {
		t.Fatalf("repair+import: %v", err)
	}
	if n < 3 {
		t.Fatalf("want >=3 upserts, got %d", n)
	}

	// No page should still use wiki id 1 for two different concepts
	var pages []model.CompassPage
	if err := db.Find(&pages).Error; err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("want 3 pages (cat+guide+major), got %d", len(pages))
	}

	var se model.CompassPage
	if err := db.Where("title = ?", "软件工程").First(&se).Error; err != nil {
		t.Fatal(err)
	}
	if se.ParentID == nil {
		t.Fatal("software eng must have parent college")
	}
	var parent model.CompassPage
	if err := db.First(&parent, *se.ParentID).Error; err != nil {
		t.Fatal(err)
	}
	if parent.Title != "计算机学院" || parent.SpaceKey != model.CompassSpaceMajors {
		t.Fatalf("parent wrong: %+v", parent)
	}
	if se.ExternalKey == nil || *se.ExternalKey != "wiki_doc:13" {
		t.Fatalf("external key: %v", se.ExternalKey)
	}
	// parent must NOT be the guide page
	if parent.Title == "简介" {
		t.Fatal("major still parented under guide 简介")
	}
}

func int64Ptr(v int64) *int64 { return &v }
