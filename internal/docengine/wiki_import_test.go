package docengine

import (
	"csu-star-backend/internal/model"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestImportPublishedWikiCreatesMajorsAndGuides(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:compass_import?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC().Truncate(time.Second)
		},
	})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}

	// Minimal tables (avoid full Users model column soup)
	if err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  role TEXT,
  status TEXT
);
CREATE TABLE wiki_categories (
  id INTEGER PRIMARY KEY,
  section TEXT NOT NULL,
  name TEXT NOT NULL,
  sort_order INTEGER DEFAULT 0,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME
);
CREATE TABLE wiki_documents (
  id INTEGER PRIMARY KEY,
  section TEXT NOT NULL,
  category_id INTEGER,
  slug TEXT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  sort_order INTEGER DEFAULT 0,
  is_published INTEGER DEFAULT 0,
  published_at DATETIME,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME
);
`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.CompassPage{}, &model.CompassPageHistory{}); err != nil {
		t.Fatal(err)
	}

	_ = db.Exec(`INSERT INTO users (id, role, status) VALUES (9, 'admin', 'active')`)

	catID := int64(1001)
	docID := int64(2002)
	guideID := int64(3003)
	_ = db.Exec(`INSERT INTO wiki_categories (id, section, name, sort_order) VALUES (?, 'major', '计算机学院', 1)`, catID)
	_ = db.Exec(`INSERT INTO wiki_documents (id, section, category_id, slug, title, content, sort_order, is_published, published_at)
		VALUES (?, 'major', ?, 'se', '软件工程', '# 软工
正文', 1, 1, CURRENT_TIMESTAMP)`, docID, catID)
	_ = db.Exec(`INSERT INTO wiki_documents (id, section, slug, title, content, sort_order, is_published, published_at)
		VALUES (?, 'compass', 'xuan-ke', '选课篇', '选课指南', 1, 1, CURRENT_TIMESTAMP)`, guideID)

	n, err := ImportPublishedWiki(db)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n < 3 {
		t.Fatalf("expected >=3 pages upserted, got %d", n)
	}

	var majorDoc model.CompassPage
	if err := db.First(&majorDoc, docID).Error; err != nil {
		t.Fatal(err)
	}
	if majorDoc.SpaceKey != model.CompassSpaceMajors || majorDoc.ContentType != model.CompassContentMajor {
		t.Fatalf("major doc mapping: %+v", majorDoc)
	}
	if majorDoc.ParentID == nil || *majorDoc.ParentID != catID {
		t.Fatalf("major doc parent want cat %d got %v", catID, majorDoc.ParentID)
	}
	if majorDoc.Body != "# 软工\n正文" {
		t.Fatalf("body not imported: %q", majorDoc.Body)
	}

	var guide model.CompassPage
	if err := db.First(&guide, guideID).Error; err != nil {
		t.Fatal(err)
	}
	if guide.SpaceKey != model.CompassSpaceGuides {
		t.Fatalf("guide space: %s", guide.SpaceKey)
	}

	// idempotent
	if _, err := ImportPublishedWiki(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.CompassPage{}).Count(&count)
	if count != 3 {
		t.Fatalf("want 3 pages after re-import, got %d", count)
	}
}
