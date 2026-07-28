-- 对齐中南大学官网二级学院命名（https://www.csu.edu.cn/xyxk1/ejxy.htm）
-- 幂等：可重复执行

BEGIN;

-- 1) departments：补全人工智能学院（历史 id 1–32 不动）
INSERT INTO departments (name, code)
SELECT '人工智能学院', NULL
WHERE NOT EXISTS (
  SELECT 1 FROM departments WHERE name = '人工智能学院'
);

-- 2) wiki 学院分类：旧名 → 官网名
-- 文学与新闻传播学院 → 人文学院
UPDATE wiki_categories c
SET name = '人文学院', updated_at = NOW()
WHERE c.section = 'major'
  AND c.name = '文学与新闻传播学院'
  AND c.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM wiki_categories x
    WHERE x.section = 'major' AND x.name = '人文学院' AND x.deleted_at IS NULL
  );

-- 若目标已存在：把文档迁过去后软删旧分类
UPDATE wiki_documents d
SET category_id = t.id, updated_at = NOW()
FROM wiki_categories o
JOIN wiki_categories t
  ON t.section = 'major' AND t.name = '人文学院' AND t.deleted_at IS NULL
WHERE o.section = 'major'
  AND o.name = '文学与新闻传播学院'
  AND o.deleted_at IS NULL
  AND d.category_id = o.id;

UPDATE wiki_categories
SET deleted_at = NOW(), updated_at = NOW()
WHERE section = 'major'
  AND name = '文学与新闻传播学院'
  AND deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM wiki_categories x
    WHERE x.section = 'major' AND x.name = '人文学院' AND x.deleted_at IS NULL
  );

-- 基础医学院 → 湘雅基础医学院
UPDATE wiki_categories c
SET name = '湘雅基础医学院', updated_at = NOW()
WHERE c.section = 'major'
  AND c.name = '基础医学院'
  AND c.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM wiki_categories x
    WHERE x.section = 'major' AND x.name = '湘雅基础医学院' AND x.deleted_at IS NULL
  );

UPDATE wiki_documents d
SET category_id = t.id, updated_at = NOW()
FROM wiki_categories o
JOIN wiki_categories t
  ON t.section = 'major' AND t.name = '湘雅基础医学院' AND t.deleted_at IS NULL
WHERE o.section = 'major'
  AND o.name = '基础医学院'
  AND o.deleted_at IS NULL
  AND d.category_id = o.id;

UPDATE wiki_categories
SET deleted_at = NOW(), updated_at = NOW()
WHERE section = 'major'
  AND name = '基础医学院'
  AND deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM wiki_categories x
    WHERE x.section = 'major' AND x.name = '湘雅基础医学院' AND x.deleted_at IS NULL
  );

-- 物理与电子学院 → 物理学院（电子类专业后续可迁到电子信息学院）
UPDATE wiki_categories c
SET name = '物理学院', updated_at = NOW()
WHERE c.section = 'major'
  AND c.name = '物理与电子学院'
  AND c.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM wiki_categories x
    WHERE x.section = 'major' AND x.name = '物理学院' AND x.deleted_at IS NULL
  );

UPDATE wiki_documents d
SET category_id = t.id, updated_at = NOW()
FROM wiki_categories o
JOIN wiki_categories t
  ON t.section = 'major' AND t.name = '物理学院' AND t.deleted_at IS NULL
WHERE o.section = 'major'
  AND o.name = '物理与电子学院'
  AND o.deleted_at IS NULL
  AND d.category_id = o.id;

UPDATE wiki_categories
SET deleted_at = NOW(), updated_at = NOW()
WHERE section = 'major'
  AND name = '物理与电子学院'
  AND deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM wiki_categories x
    WHERE x.section = 'major' AND x.name = '物理学院' AND x.deleted_at IS NULL
  );

-- 确保电子信息学院 / 人工智能学院 / 物理学院分类存在
INSERT INTO wiki_categories (section, name, sort_order)
SELECT 'major', '电子信息学院', 65
WHERE NOT EXISTS (
  SELECT 1 FROM wiki_categories
  WHERE section = 'major' AND name = '电子信息学院' AND deleted_at IS NULL
);

INSERT INTO wiki_categories (section, name, sort_order)
SELECT 'major', '人工智能学院', 55
WHERE NOT EXISTS (
  SELECT 1 FROM wiki_categories
  WHERE section = 'major' AND name = '人工智能学院' AND deleted_at IS NULL
);

INSERT INTO wiki_categories (section, name, sort_order)
SELECT 'major', '物理学院', 60
WHERE NOT EXISTS (
  SELECT 1 FROM wiki_categories
  WHERE section = 'major' AND name = '物理学院' AND deleted_at IS NULL
);

-- 文档归属校正（按 slug）
UPDATE wiki_documents d
SET category_id = c.id, updated_at = NOW()
FROM wiki_categories c
WHERE d.section = 'major'
  AND d.deleted_at IS NULL
  AND c.section = 'major'
  AND c.deleted_at IS NULL
  AND c.name = '电子信息学院'
  AND d.slug IN ('电子信息科学与技术', '光电信息科学与工程');

UPDATE wiki_documents d
SET category_id = c.id, updated_at = NOW()
FROM wiki_categories c
WHERE d.section = 'major'
  AND d.deleted_at IS NULL
  AND c.section = 'major'
  AND c.deleted_at IS NULL
  AND c.name = '物理学院'
  AND d.slug IN ('应用物理学');

UPDATE wiki_documents d
SET category_id = c.id, updated_at = NOW()
FROM wiki_categories c
WHERE d.section = 'major'
  AND d.deleted_at IS NULL
  AND c.section = 'major'
  AND c.deleted_at IS NULL
  AND c.name = '人工智能学院'
  AND d.slug IN ('人工智能');

-- 清理 wiki 树缓存（若使用 redis 需另清；此处无 DB 缓存）
COMMIT;
