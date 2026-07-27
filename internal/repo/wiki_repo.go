package repo

import (
	"csu-star-backend/internal/model"

	"gorm.io/gorm"
)

type WikiCategoryItem struct {
	model.WikiCategories
	DocCount int64 `json:"doc_count"`
}

type WikiDocMeta struct {
	ID          int64             `json:"id,string"`
	Section     model.WikiSection `json:"section"`
	CategoryID  *int64            `json:"category_id,string"`
	Slug        string            `json:"slug"`
	Title       string            `json:"title"`
	SortOrder   int               `json:"sort_order"`
	IsPublished bool              `json:"is_published"`
	UpdatedAt   string            `json:"updated_at"`
}

type WikiDocListQuery struct {
	Section    string
	CategoryID *int64
	Keyword    string
	Status     string // published | draft | ""
	Page       int
	Size       int
}

type WikiRepository interface {
	WithTx(tx *gorm.DB) WikiRepository

	ListSections() ([]model.WikiSections, error)
	GetSection(key string) (*model.WikiSections, error)

	ListCategories(section string) ([]WikiCategoryItem, error)
	GetCategoryByID(id int64) (*model.WikiCategories, error)
	CreateCategory(c *model.WikiCategories) error
	UpdateCategory(id int64, fields map[string]any) error
	DeleteCategory(id int64) error
	CountDocsInCategory(id int64) (int64, error)

	ListDocs(q WikiDocListQuery) ([]WikiDocMeta, int64, error)
	ListPublishedDocMetas() ([]WikiDocMeta, error)
	GetDocByID(id int64) (*model.WikiDocuments, error)
	GetPublishedDoc(section model.WikiSection, slug string) (*model.WikiDocuments, error)
	CreateDoc(d *model.WikiDocuments) error
	UpdateDoc(id int64, fields map[string]any) error
	DeleteDoc(id int64) error

	UpdateSortOrders(table string, ids []int64) error
	CreateAuditLog(log *model.AuditLogs) error
}

type wikiRepository struct {
	db *gorm.DB
}

func NewWikiRepository(db *gorm.DB) WikiRepository {
	return &wikiRepository{db: db}
}

func (r *wikiRepository) WithTx(tx *gorm.DB) WikiRepository {
	return &wikiRepository{db: tx}
}

func (r *wikiRepository) ListSections() ([]model.WikiSections, error) {
	var out []model.WikiSections
	err := r.db.Model(&model.WikiSections{}).
		Order("sort_order ASC, key ASC").
		Find(&out).Error
	return out, err
}

func (r *wikiRepository) GetSection(key string) (*model.WikiSections, error) {
	var s model.WikiSections
	if err := r.db.Where("key = ?", key).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *wikiRepository) ListCategories(section string) ([]WikiCategoryItem, error) {
	var out []WikiCategoryItem
	query := r.db.Model(&model.WikiCategories{}).
		Select("wiki_categories.*, (SELECT COUNT(*) FROM wiki_documents d WHERE d.category_id = wiki_categories.id AND d.deleted_at IS NULL) AS doc_count")
	if section != "" {
		query = query.Where("section = ?", section)
	}
	err := query.Order("sort_order ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *wikiRepository) GetCategoryByID(id int64) (*model.WikiCategories, error) {
	var c model.WikiCategories
	if err := r.db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *wikiRepository) CreateCategory(c *model.WikiCategories) error {
	return r.db.Create(c).Error
}

func (r *wikiRepository) UpdateCategory(id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&model.WikiCategories{}).Where("id = ?", id).Updates(fields).Error
}

func (r *wikiRepository) DeleteCategory(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.WikiCategories{}).Error
}

func (r *wikiRepository) CountDocsInCategory(id int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.WikiDocuments{}).Where("category_id = ?", id).Count(&count).Error
	return count, err
}

const wikiDocMetaColumns = "id, section, category_id, slug, title, sort_order, is_published, updated_at"

func (r *wikiRepository) ListDocs(q WikiDocListQuery) ([]WikiDocMeta, int64, error) {
	query := r.db.Model(&model.WikiDocuments{})
	if q.Section != "" {
		query = query.Where("section = ?", q.Section)
	}
	if q.CategoryID != nil {
		if *q.CategoryID == 0 {
			query = query.Where("category_id IS NULL")
		} else {
			query = query.Where("category_id = ?", *q.CategoryID)
		}
	}
	if q.Keyword != "" {
		query = query.Where("title ILIKE ?", "%"+q.Keyword+"%")
	}
	switch q.Status {
	case "published":
		query = query.Where("is_published = TRUE")
	case "draft":
		query = query.Where("is_published = FALSE")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []WikiDocMeta
	err := query.Select(wikiDocMetaColumns).
		Order("section ASC, category_id ASC NULLS FIRST, sort_order ASC, id ASC").
		Offset((q.Page - 1) * q.Size).Limit(q.Size).
		Find(&out).Error
	return out, total, err
}

func (r *wikiRepository) ListPublishedDocMetas() ([]WikiDocMeta, error) {
	var out []WikiDocMeta
	err := r.db.Model(&model.WikiDocuments{}).
		Select(wikiDocMetaColumns).
		Where("is_published = TRUE").
		Order("section ASC, category_id ASC NULLS FIRST, sort_order ASC, id ASC").
		Find(&out).Error
	return out, err
}

func (r *wikiRepository) GetDocByID(id int64) (*model.WikiDocuments, error) {
	var d model.WikiDocuments
	if err := r.db.Where("id = ?", id).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *wikiRepository) GetPublishedDoc(section model.WikiSection, slug string) (*model.WikiDocuments, error) {
	var d model.WikiDocuments
	err := r.db.Where("section = ? AND slug = ? AND is_published = TRUE", section, slug).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *wikiRepository) CreateDoc(d *model.WikiDocuments) error {
	return r.db.Create(d).Error
}

func (r *wikiRepository) UpdateDoc(id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&model.WikiDocuments{}).Where("id = ?", id).Updates(fields).Error
}

func (r *wikiRepository) DeleteDoc(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.WikiDocuments{}).Error
}

// UpdateSortOrders 按 ids 的数组序重写 sort_order(10,20,30…留间隙便于手工插入)。
func (r *wikiRepository) UpdateSortOrders(table string, ids []int64) error {
	for i, id := range ids {
		if err := r.db.Table(table).Where("id = ? AND deleted_at IS NULL", id).
			Update("sort_order", (i+1)*10).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *wikiRepository) CreateAuditLog(log *model.AuditLogs) error {
	return r.db.Create(log).Error
}
