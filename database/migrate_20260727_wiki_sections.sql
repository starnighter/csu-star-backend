-- Wiki 板块注册表：可扩展多板块，section 从 enum 改为 varchar
-- 幂等：可重复执行（关键步骤带 IF NOT EXISTS / 条件判断）

BEGIN;

-- 1. 板块注册表
CREATE TABLE IF NOT EXISTS wiki_sections (
  key               VARCHAR(32)  PRIMARY KEY,
  title             VARCHAR(64)  NOT NULL,
  sort_order        INTEGER      NOT NULL DEFAULT 0,
  allow_categories  BOOLEAN      NOT NULL DEFAULT FALSE,
  created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_wiki_sections_sort
  ON wiki_sections (sort_order) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_wiki_sections_deleted_at
  ON wiki_sections (deleted_at);

INSERT INTO wiki_sections (key, title, sort_order, allow_categories)
VALUES
  ('compass', '入坑指南', 10, FALSE),
  ('major',   '专业指北', 20, TRUE)
ON CONFLICT (key) DO UPDATE
SET
  title = EXCLUDED.title,
  sort_order = EXCLUDED.sort_order,
  allow_categories = EXCLUDED.allow_categories,
  updated_at = NOW();

-- 2. section 列从 enum 转为 varchar(32)
DO $$
BEGIN
  -- categories
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'wiki_categories' AND column_name = 'section'
      AND udt_name = 'wiki_section'
  ) THEN
    ALTER TABLE wiki_categories
      ALTER COLUMN section DROP DEFAULT,
      ALTER COLUMN section TYPE VARCHAR(32) USING section::text;
    ALTER TABLE wiki_categories
      ALTER COLUMN section SET DEFAULT 'major';
  END IF;

  -- documents
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'wiki_documents' AND column_name = 'section'
      AND udt_name = 'wiki_section'
  ) THEN
    ALTER TABLE wiki_documents
      ALTER COLUMN section TYPE VARCHAR(32) USING section::text;
  END IF;
END $$;

-- 3. 外键（若尚无）
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_wiki_categories_section'
  ) THEN
    ALTER TABLE wiki_categories
      ADD CONSTRAINT fk_wiki_categories_section
      FOREIGN KEY (section) REFERENCES wiki_sections(key);
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_wiki_documents_section'
  ) THEN
    ALTER TABLE wiki_documents
      ADD CONSTRAINT fk_wiki_documents_section
      FOREIGN KEY (section) REFERENCES wiki_sections(key);
  END IF;
END $$;

-- 4. 删除旧 enum（仅当无其它依赖）
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'wiki_section') THEN
    -- 确认无表仍使用该类型
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns WHERE udt_name = 'wiki_section'
    ) THEN
      DROP TYPE wiki_section;
    END IF;
  END IF;
END $$;

COMMIT;
