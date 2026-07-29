-- migrate_20260729_wiki_intro_before_colleges.sql
-- 彻底修复：专业/指南「简介」排在学院之前。
-- 1) wiki_documents：简介必须 category_id NULL、sort_order=0
-- 2) compass_pages：wiki_doc 根简介 sort=0；wiki_cat 学院文件夹 sort>=100000
-- 幂等，可重复执行。

BEGIN;

-- ---------- wiki 正式文档 ----------
UPDATE wiki_documents
SET category_id = NULL,
    sort_order = 0,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND section IN ('major', 'compass')
  AND (
    title IN ('简介', '专业简介', '专业指北简介', '指南简介')
    OR slug IN ('index', '简介', 'intro')
  )
  AND (category_id IS NOT NULL OR sort_order IS DISTINCT FROM 0);

-- ---------- compass 协作树（导入镜像）----------
-- 根级简介文档置顶
UPDATE compass_pages
SET sort_order = 0,
    parent_id = NULL,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND space_key IN ('majors', 'guides')
  AND (
    (external_key LIKE 'wiki_doc:%' AND parent_id IS NULL)
    OR title IN ('简介', '专业简介', '专业指北简介', '指南简介')
  )
  AND (
    title IN ('简介', '专业简介', '专业指北简介', '指南简介')
    OR external_key IN (
      SELECT 'wiki_doc:' || id::text
      FROM wiki_documents
      WHERE deleted_at IS NULL
        AND section IN ('major', 'compass')
        AND (
          title IN ('简介', '专业简介', '专业指北简介', '指南简介')
          OR slug IN ('index', '简介', 'intro')
        )
    )
  )
  AND (sort_order IS DISTINCT FROM 0 OR parent_id IS NOT NULL);

-- 学院文件夹统一抬到 100000+ 原序（避免与简介 sort=10 平手）
UPDATE compass_pages
SET sort_order = 100000 + (sort_order % 100000),
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND external_key LIKE 'wiki_cat:%'
  AND sort_order < 100000;

COMMIT;
