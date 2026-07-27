package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrWikiInvalidPayload   = errors.New("wiki invalid payload")
	ErrWikiNotFound         = errors.New("wiki target not found")
	ErrWikiCategoryNotEmpty = errors.New("wiki category not empty")
	ErrWikiSlugConflict     = errors.New("wiki slug conflict")
)

const (
	wikiTreeCacheKey    = "cache:wiki:tree"
	wikiDocCacheKeyFmt  = "cache:wiki:doc:%s:%s"
	wikiCacheTTL        = time.Hour
	wikiCachePattern    = "cache:wiki:*"
	wikiAuditCategory   = "wiki_category"
	wikiAuditDocument   = "wiki_document"
)

type WikiService struct {
	db   *gorm.DB
	repo repo.WikiRepository
}

func NewWikiService(db *gorm.DB, r repo.WikiRepository) *WikiService {
	return &WikiService{db: db, repo: r}
}

// ---------- 公开读 ----------

type WikiTreeDoc struct {
	ID    int64  `json:"id,string"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type WikiTreeCategory struct {
	ID   int64         `json:"id,string"`
	Name string        `json:"name"`
	Docs []WikiTreeDoc `json:"docs"`
}

type WikiTreeSection struct {
	Section         string             `json:"section"`
	Title           string             `json:"title"`
	AllowCategories bool               `json:"allow_categories"`
	Docs            []WikiTreeDoc      `json:"docs"`
	Categories      []WikiTreeCategory `json:"categories"`
}

// WikiSectionMeta 管理端板块列表项。
type WikiSectionMeta struct {
	Key             string `json:"key"`
	Title           string `json:"title"`
	SortOrder       int    `json:"sort_order"`
	AllowCategories bool   `json:"allow_categories"`
}

type WikiTree struct {
	Sections []WikiTreeSection `json:"sections"`
}

type WikiDocDetail struct {
	ID           int64      `json:"id,string"`
	Section      string     `json:"section"`
	CategoryName string     `json:"category_name,omitempty"`
	Slug         string     `json:"slug"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	UpdatedAt    time.Time  `json:"updated_at"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

// GetTree 返回已发布文档组成的目录树(不含正文),供前端侧边栏与门户页。
func (s *WikiService) GetTree() (*WikiTree, error) {
	cached, err := utils.RDB.Get(utils.Ctx, wikiTreeCacheKey).Bytes()
	if err == nil {
		var tree WikiTree
		if json.Unmarshal(cached, &tree) == nil {
			return &tree, nil
		}
	} else if err != redis.Nil {
		logger.Log.Warn("wiki 目录树缓存读取失败", zap.Error(err))
	}

	tree, err := s.buildTree()
	if err != nil {
		return nil, err
	}

	if data, marshalErr := json.Marshal(tree); marshalErr == nil {
		if setErr := utils.RDB.Set(utils.Ctx, wikiTreeCacheKey, data, wikiCacheTTL).Err(); setErr != nil {
			logger.Log.Warn("wiki 目录树缓存写入失败", zap.Error(setErr))
		}
	}
	return tree, nil
}

func (s *WikiService) buildTree() (*WikiTree, error) {
	registry, err := s.repo.ListSections()
	if err != nil {
		return nil, err
	}
	categories, err := s.repo.ListCategories("")
	if err != nil {
		return nil, err
	}
	docs, err := s.repo.ListPublishedDocMetas()
	if err != nil {
		return nil, err
	}

	sections := make([]WikiTreeSection, 0, len(registry))
	sectionIndex := make(map[model.WikiSection]*WikiTreeSection, len(registry))
	for _, reg := range registry {
		sections = append(sections, WikiTreeSection{
			Section:         reg.Key,
			Title:           reg.Title,
			AllowCategories: reg.AllowCategories,
			Docs:            []WikiTreeDoc{},
			Categories:      []WikiTreeCategory{},
		})
	}
	for i := range sections {
		sectionIndex[model.WikiSection(sections[i].Section)] = &sections[i]
	}

	// 先按分类分桶,再组装分类列表;不可持有指向 sec.Categories 元素的指针,append 扩容会使其失效。
	docsByCategory := make(map[int64][]WikiTreeDoc, len(categories))
	for _, c := range categories {
		docsByCategory[c.ID] = []WikiTreeDoc{}
	}
	for _, d := range docs {
		node := WikiTreeDoc{ID: d.ID, Slug: d.Slug, Title: d.Title}
		if d.CategoryID != nil {
			if list, ok := docsByCategory[*d.CategoryID]; ok {
				docsByCategory[*d.CategoryID] = append(list, node)
				continue
			}
		}
		if sec, ok := sectionIndex[d.Section]; ok {
			sec.Docs = append(sec.Docs, node)
		}
	}

	for _, c := range categories {
		sec, ok := sectionIndex[c.Section]
		if !ok || !sec.AllowCategories {
			continue
		}
		sec.Categories = append(sec.Categories, WikiTreeCategory{ID: c.ID, Name: c.Name, Docs: docsByCategory[c.ID]})
	}

	return &WikiTree{Sections: sections}, nil
}

// ListSectionMetas 管理端 / 前端需要的板块注册表。
func (s *WikiService) ListSectionMetas() ([]WikiSectionMeta, error) {
	rows, err := s.repo.ListSections()
	if err != nil {
		return nil, err
	}
	out := make([]WikiSectionMeta, 0, len(rows))
	for _, r := range rows {
		out = append(out, WikiSectionMeta{
			Key:             r.Key,
			Title:           r.Title,
			SortOrder:       r.SortOrder,
			AllowCategories: r.AllowCategories,
		})
	}
	return out, nil
}

func (s *WikiService) requireSection(key string) (*model.WikiSections, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrWikiInvalidPayload
	}
	sec, err := s.repo.GetSection(key)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiInvalidPayload
	}
	if err != nil {
		return nil, err
	}
	return sec, nil
}

// GetDoc 返回已发布文档详情。未发布/不存在统一返回 ErrWikiNotFound,不暴露草稿存在性。
func (s *WikiService) GetDoc(section, slug string) (*WikiDocDetail, error) {
	if _, err := s.requireSection(section); err != nil {
		return nil, ErrWikiNotFound
	}
	sec := model.WikiSection(section)
	if strings.TrimSpace(slug) == "" {
		return nil, ErrWikiNotFound
	}

	cacheKey := fmt.Sprintf(wikiDocCacheKeyFmt, section, slug)
	cached, err := utils.RDB.Get(utils.Ctx, cacheKey).Bytes()
	if err == nil {
		var detail WikiDocDetail
		if json.Unmarshal(cached, &detail) == nil {
			return &detail, nil
		}
	} else if err != redis.Nil {
		logger.Log.Warn("wiki 文档缓存读取失败", zap.Error(err))
	}

	doc, err := s.repo.GetPublishedDoc(sec, slug)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiNotFound
	}
	if err != nil {
		return nil, err
	}

	detail := &WikiDocDetail{
		ID:          doc.ID,
		Section:     string(doc.Section),
		Slug:        doc.Slug,
		Title:       doc.Title,
		Content:     doc.Content,
		UpdatedAt:   doc.UpdatedAt,
		PublishedAt: doc.PublishedAt,
	}
	if doc.CategoryID != nil {
		if cat, catErr := s.repo.GetCategoryByID(*doc.CategoryID); catErr == nil {
			detail.CategoryName = cat.Name
		}
	}

	if data, marshalErr := json.Marshal(detail); marshalErr == nil {
		if setErr := utils.RDB.Set(utils.Ctx, cacheKey, data, wikiCacheTTL).Err(); setErr != nil {
			logger.Log.Warn("wiki 文档缓存写入失败", zap.Error(setErr))
		}
	}
	return detail, nil
}

// ---------- admin ----------

func (s *WikiService) invalidateCache() {
	if err := utils.DeleteKeysByPattern(wikiCachePattern); err != nil {
		logger.Log.Warn("wiki 缓存失效失败", zap.Error(err))
	}
}

func (s *WikiService) withWriteTx(fn func(r repo.WikiRepository) error) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		return fn(s.repo.WithTx(tx))
	})
	if err == nil {
		s.invalidateCache()
	}
	return err
}

func (s *WikiService) ListCategories(section string) ([]repo.WikiCategoryItem, error) {
	return s.repo.ListCategories(section)
}

func (s *WikiService) CreateCategory(operatorID int64, section, name string, sortOrder *int, ip net.IP) (*model.WikiCategories, error) {
	if section == "" {
		section = string(model.WikiSectionMajor)
	}
	reg, err := s.requireSection(section)
	if err != nil {
		return nil, err
	}
	if !reg.AllowCategories {
		return nil, ErrWikiInvalidPayload
	}
	sec := model.WikiSection(section)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrWikiInvalidPayload
	}
	category := &model.WikiCategories{Section: sec, Name: name}
	if sortOrder != nil {
		category.SortOrder = *sortOrder
	}
	err = s.withWriteTx(func(r repo.WikiRepository) error {
		if err := r.CreateCategory(category); err != nil {
			return err
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionCreate,
			TargetType: wikiAuditCategory,
			TargetID:   category.ID,
			NewValues:  mustJSON(category),
			Reason:     "create wiki category",
			IpAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (s *WikiService) UpdateCategory(operatorID, id int64, name string, sortOrder *int, ip net.IP) (*model.WikiCategories, error) {
	name = strings.TrimSpace(name)
	if name == "" && sortOrder == nil {
		return nil, ErrWikiInvalidPayload
	}
	var updated *model.WikiCategories
	err := s.withWriteTx(func(r repo.WikiRepository) error {
		category, err := r.GetCategoryByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWikiNotFound
		}
		if err != nil {
			return err
		}
		oldValues := mustJSON(category)
		fields := map[string]any{}
		if name != "" {
			fields["name"] = name
			category.Name = name
		}
		if sortOrder != nil {
			fields["sort_order"] = *sortOrder
			category.SortOrder = *sortOrder
		}
		if err := r.UpdateCategory(id, fields); err != nil {
			return err
		}
		updated = category
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionUpdate,
			TargetType: wikiAuditCategory,
			TargetID:   id,
			OldValues:  oldValues,
			NewValues:  mustJSON(category),
			Reason:     "update wiki category",
			IpAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *WikiService) DeleteCategory(operatorID, id int64, ip net.IP) error {
	return s.withWriteTx(func(r repo.WikiRepository) error {
		category, err := r.GetCategoryByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWikiNotFound
		}
		if err != nil {
			return err
		}
		count, err := r.CountDocsInCategory(id)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrWikiCategoryNotEmpty
		}
		if err := r.DeleteCategory(id); err != nil {
			return err
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: wikiAuditCategory,
			TargetID:   id,
			OldValues:  mustJSON(category),
			Reason:     "delete wiki category",
			IpAddress:  ip,
		})
	})
}

func (s *WikiService) ReorderCategories(operatorID int64, ids []int64, ip net.IP) error {
	if len(ids) == 0 {
		return ErrWikiInvalidPayload
	}
	return s.withWriteTx(func(r repo.WikiRepository) error {
		if err := r.UpdateSortOrders("wiki_categories", ids); err != nil {
			return err
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionUpdate,
			TargetType: wikiAuditCategory,
			NewValues:  mustJSON(map[string]any{"ordered_ids": ids}),
			Reason:     "reorder wiki categories",
			IpAddress:  ip,
		})
	})
}

func (s *WikiService) ListDocs(q *repo.WikiDocListQuery) ([]repo.WikiDocMeta, int64, error) {
	// 管理端目录树需要一次拉全量（req 上限 200）；勿用全局 fillPagination 的 50 硬顶，
	// 否则 admin 侧 size=200 会静默截断，超出部分文档从树里消失。
	fillWikiAdminPagination(&q.Page, &q.Size)
	return s.repo.ListDocs(*q)
}

const wikiAdminListMaxSize = 200

func fillWikiAdminPagination(page, size *int) {
	if *page <= 0 {
		*page = 1
	}
	if *size <= 0 {
		*size = 50
	}
	if *size > wikiAdminListMaxSize {
		*size = wikiAdminListMaxSize
	}
}

func (s *WikiService) GetDocByID(id int64) (*model.WikiDocuments, error) {
	doc, err := s.repo.GetDocByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiNotFound
	}
	return doc, err
}

// validateDocPlacement 校验「板块是否允许分组、分组板块一致」。
func validateDocPlacement(r repo.WikiRepository, section model.WikiSection, categoryID *int64) error {
	reg, err := r.GetSection(string(section))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrWikiInvalidPayload
	}
	if err != nil {
		return err
	}
	if categoryID == nil {
		return nil
	}
	if !reg.AllowCategories {
		return ErrWikiInvalidPayload
	}
	category, err := r.GetCategoryByID(*categoryID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrWikiInvalidPayload
	}
	if err != nil {
		return err
	}
	if category.Section != section {
		return ErrWikiInvalidPayload
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "uq_wiki_documents_section_slug")
}

func (s *WikiService) CreateDoc(operatorID int64, section string, categoryID *int64, slug, title, content string, sortOrder *int, ip net.IP) (*model.WikiDocuments, error) {
	if _, err := s.requireSection(section); err != nil {
		return nil, err
	}
	sec := model.WikiSection(section)
	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)
	if title == "" {
		return nil, ErrWikiInvalidPayload
	}
	if slug == "" {
		slug = title
	}
	if strings.Contains(slug, "/") {
		return nil, ErrWikiInvalidPayload
	}
	doc := &model.WikiDocuments{
		Section:    sec,
		CategoryID: categoryID,
		Slug:       slug,
		Title:      title,
		Content:    content,
	}
	if sortOrder != nil {
		doc.SortOrder = *sortOrder
	}
	err := s.withWriteTx(func(r repo.WikiRepository) error {
		if err := validateDocPlacement(r, sec, categoryID); err != nil {
			return err
		}
		if err := r.CreateDoc(doc); err != nil {
			if isUniqueViolation(err) {
				return ErrWikiSlugConflict
			}
			return err
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionCreate,
			TargetType: wikiAuditDocument,
			TargetID:   doc.ID,
			NewValues:  mustJSON(docAuditView(doc)),
			Reason:     "create wiki document",
			IpAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return doc, nil
}

type WikiDocUpdate struct {
	CategoryID    *int64
	ClearCategory bool
	Slug          *string
	Title         *string
	Content       *string
	SortOrder     *int
}

func (s *WikiService) UpdateDoc(operatorID, id int64, u WikiDocUpdate, ip net.IP) (*model.WikiDocuments, error) {
	var updated *model.WikiDocuments
	err := s.withWriteTx(func(r repo.WikiRepository) error {
		doc, err := r.GetDocByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWikiNotFound
		}
		if err != nil {
			return err
		}
		oldValues := mustJSON(docAuditView(doc))

		fields := map[string]any{}
		if u.ClearCategory {
			fields["category_id"] = nil
			doc.CategoryID = nil
		} else if u.CategoryID != nil {
			fields["category_id"] = *u.CategoryID
			doc.CategoryID = u.CategoryID
		}
		if u.Slug != nil {
			slug := strings.TrimSpace(*u.Slug)
			if slug == "" || strings.Contains(slug, "/") {
				return ErrWikiInvalidPayload
			}
			fields["slug"] = slug
			doc.Slug = slug
		}
		if u.Title != nil {
			title := strings.TrimSpace(*u.Title)
			if title == "" {
				return ErrWikiInvalidPayload
			}
			fields["title"] = title
			doc.Title = title
		}
		if u.Content != nil {
			fields["content"] = *u.Content
			doc.Content = *u.Content
		}
		if u.SortOrder != nil {
			fields["sort_order"] = *u.SortOrder
			doc.SortOrder = *u.SortOrder
		}
		if len(fields) == 0 {
			return ErrWikiInvalidPayload
		}

		if err := validateDocPlacement(r, doc.Section, doc.CategoryID); err != nil {
			return err
		}
		if err := r.UpdateDoc(id, fields); err != nil {
			if isUniqueViolation(err) {
				return ErrWikiSlugConflict
			}
			return err
		}
		updated = doc
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionUpdate,
			TargetType: wikiAuditDocument,
			TargetID:   id,
			OldValues:  oldValues,
			NewValues:  mustJSON(docAuditView(doc)),
			Reason:     "update wiki document",
			IpAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// SetDocPublished 发布/下架。独立于 UpdateDoc,便于审计里区分内容修改与可见性变更。
func (s *WikiService) SetDocPublished(operatorID, id int64, published bool, ip net.IP) (*model.WikiDocuments, error) {
	var updated *model.WikiDocuments
	err := s.withWriteTx(func(r repo.WikiRepository) error {
		doc, err := r.GetDocByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWikiNotFound
		}
		if err != nil {
			return err
		}
		oldValues := mustJSON(docAuditView(doc))
		fields := map[string]any{"is_published": published}
		doc.IsPublished = published
		if published && doc.PublishedAt == nil {
			now := time.Now()
			fields["published_at"] = now
			doc.PublishedAt = &now
		}
		if err := r.UpdateDoc(id, fields); err != nil {
			return err
		}
		updated = doc
		reason := "publish wiki document"
		if !published {
			reason = "unpublish wiki document"
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionUpdate,
			TargetType: wikiAuditDocument,
			TargetID:   id,
			OldValues:  oldValues,
			NewValues:  mustJSON(docAuditView(doc)),
			Reason:     reason,
			IpAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *WikiService) ReorderDocs(operatorID int64, ids []int64, ip net.IP) error {
	if len(ids) == 0 {
		return ErrWikiInvalidPayload
	}
	return s.withWriteTx(func(r repo.WikiRepository) error {
		if err := r.UpdateSortOrders("wiki_documents", ids); err != nil {
			return err
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionUpdate,
			TargetType: wikiAuditDocument,
			NewValues:  mustJSON(map[string]any{"ordered_ids": ids}),
			Reason:     "reorder wiki documents",
			IpAddress:  ip,
		})
	})
}

func (s *WikiService) DeleteDoc(operatorID, id int64, ip net.IP) error {
	return s.withWriteTx(func(r repo.WikiRepository) error {
		doc, err := r.GetDocByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWikiNotFound
		}
		if err != nil {
			return err
		}
		if err := r.DeleteDoc(id); err != nil {
			return err
		}
		return r.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: wikiAuditDocument,
			TargetID:   id,
			OldValues:  mustJSON(docAuditView(doc)),
			Reason:     "delete wiki document",
			IpAddress:  ip,
		})
	})
}

// docAuditView 审计里不落全文,避免 audit_logs 因长文膨胀;正文变更看内容长度即可。
func docAuditView(d *model.WikiDocuments) map[string]any {
	return map[string]any{
		"id":            d.ID,
		"section":       d.Section,
		"category_id":   d.CategoryID,
		"slug":          d.Slug,
		"title":         d.Title,
		"content_bytes": len(d.Content),
		"sort_order":    d.SortOrder,
		"is_published":  d.IsPublished,
	}
}
