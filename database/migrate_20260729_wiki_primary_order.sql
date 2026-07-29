-- migrate_20260729_wiki_primary_order.sql
-- 保证 major「简介」为根文档（category_id NULL）且 sort 靠前；学院 sort 可管理端再调。
-- 幂等。

BEGIN;

-- 专业指北「简介」必须挂在板块根，不能进学院
UPDATE wiki_documents
SET category_id = NULL,
    sort_order = 0,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND section = 'major'
  AND (title = '简介' OR slug IN ('index', '简介'))
  AND (category_id IS NOT NULL OR sort_order <> 0);

-- 入坑指南「简介」同样保证为根文档
UPDATE wiki_documents
SET category_id = NULL,
    sort_order = 0,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND section = 'compass'
  AND (title = '简介' OR slug IN ('index', '简介'))
  AND (category_id IS NOT NULL OR sort_order <> 0);

COMMIT;
