-- CSU-Star 南极星数据库结构 (GORM 适配版)
-- 版本：V2.0
-- 数据库：PostgreSQL 16

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 1. departments — 学院表
CREATE TABLE departments (
  id         SMALLSERIAL  PRIMARY KEY,
  name       VARCHAR(64)  NOT NULL UNIQUE,
  code       VARCHAR(16)  UNIQUE,
  created_at TIMESTAMPTZ    DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ    DEFAULT CURRENT_TIMESTAMP
);

-- 2. users — 用户表
CREATE TYPE user_status AS ENUM ('active', 'banned');
CREATE TYPE user_role AS ENUM ('user', 'admin', 'auditor');

CREATE TABLE users (
  id                BIGINT      PRIMARY KEY,
  email             VARCHAR(255)   UNIQUE,
  -- 学号。邮箱域名放开后不再能从邮箱前缀推导，故独立成列。
  -- student_id_source: campus_email(从存量校园邮箱回填) | manual | sso
  student_id        VARCHAR(32),
  student_id_source VARCHAR(16)    DEFAULT '',
  password          VARCHAR(255),
  nickname          VARCHAR(64),
  avatar_url        VARCHAR(500),
  role              user_role      DEFAULT 'user',
  status            user_status    DEFAULT 'active',
  email_verified    BOOLEAN        DEFAULT FALSE,
  points            INTEGER        DEFAULT 5,
  free_download_count INTEGER DEFAULT 3,
  inviter_id        BIGINT         REFERENCES users(id),
  last_login_at     TIMESTAMPTZ,
  metadata          JSONB,
  created_at        TIMESTAMPTZ      DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMPTZ      DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uq_users_student_id ON users (student_id) WHERE student_id IS NOT NULL;
-- email 的 UNIQUE 约束是大小写敏感的，这条补上大小写无关的唯一性
CREATE UNIQUE INDEX uq_users_email_lower ON users (lower(email)) WHERE email IS NOT NULL;

-- 3. user_oauth_bindings — OAuth 绑定表
CREATE TYPE oauth_provider AS ENUM ('wechat', 'qq', 'github', 'google');

CREATE TABLE user_oauth_bindings (
  id         BIGSERIAL     PRIMARY KEY,
  user_id    BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider   oauth_provider NOT NULL,
  openid     VARCHAR(255)  NOT NULL,
  unionid    VARCHAR(255),
  bound_at   TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
  metadata   JSONB,
  created_at TIMESTAMPTZ     DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ     DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, provider),
  UNIQUE(provider, openid)
);

-- 4. teachers — 教师表
CREATE TABLE teachers (
  id            BIGSERIAL    PRIMARY KEY,
  name          VARCHAR(64)  NOT NULL,
  title         VARCHAR(32),
  department_id SMALLINT     NOT NULL REFERENCES departments(id),
  avatar_url    VARCHAR(500),
  metadata      JSONB,
  avg_teaching_score    NUMERIC(3,2),
  avg_grading_score     NUMERIC(3,2),
  avg_attendance_score  NUMERIC(3,2),
  approval_rate         NUMERIC(5,2),
  resource_count        INTEGER DEFAULT 0,
  favorite_count        INTEGER DEFAULT 0,
  eval_count            INTEGER DEFAULT 0,
  created_at            TIMESTAMPTZ     DEFAULT CURRENT_TIMESTAMP,
  updated_at            TIMESTAMPTZ     DEFAULT CURRENT_TIMESTAMP
);

-- 5. courses — 课程表
CREATE TYPE course_type AS ENUM ('public', 'non_public');

CREATE TABLE courses (
  id            BIGSERIAL    PRIMARY KEY,
  code          VARCHAR(32),
  name          VARCHAR(128) NOT NULL,
  credits       NUMERIC(3,1),
  course_type   course_type,
  description   TEXT,
  metadata      JSONB,
  avg_workload_score   NUMERIC(3,2),
  avg_gain_score       NUMERIC(3,2),
  avg_difficulty_score NUMERIC(3,2),
  resource_count       INTEGER DEFAULT 0,
  download_total       INTEGER DEFAULT 0,
  like_total           INTEGER DEFAULT 0,
  favorite_count       INTEGER DEFAULT 0,
  eval_count           INTEGER DEFAULT 0,
  hot_score            INTEGER DEFAULT 0,
  created_at           TIMESTAMPTZ     DEFAULT CURRENT_TIMESTAMP,
  updated_at           TIMESTAMPTZ     DEFAULT CURRENT_TIMESTAMP
);

-- 6. course_teachers — 课程教师关联表
CREATE TABLE course_teachers (
  id         BIGSERIAL PRIMARY KEY,
  course_id  BIGINT    NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  teacher_id BIGINT    NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
  semester   VARCHAR(16),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(course_id, teacher_id, semester)
);

-- 7. resources — 资源表
CREATE TYPE resource_status AS ENUM ('draft', 'pending', 'approved', 'rejected', 'deleted');

CREATE TABLE resources (
  id          BIGSERIAL     PRIMARY KEY,
  title       VARCHAR(128)  NOT NULL,
  description TEXT,
  uploader_id BIGINT        NOT NULL REFERENCES users(id),
  course_id   BIGINT        NOT NULL REFERENCES courses(id),
  type        VARCHAR(64)   NOT NULL,
  status      resource_status DEFAULT 'approved',
  download_count INTEGER DEFAULT 0,
  view_count     INTEGER DEFAULT 0,
  like_count     INTEGER DEFAULT 0,
  comment_count  INTEGER DEFAULT 0,
  metadata       JSONB,
  created_at     TIMESTAMP      DEFAULT CURRENT_TIMESTAMP,
  updated_at     TIMESTAMP      DEFAULT CURRENT_TIMESTAMP
);

-- 8. resource_files — 资源文件表
CREATE TABLE resource_files (
  id           BIGSERIAL    PRIMARY KEY,
  resource_id  BIGINT       NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
  filename     VARCHAR(255) NOT NULL,
  file_key     VARCHAR(500) NOT NULL,
  file_url     VARCHAR(255) NOT NULL,
  file_size    BIGINT,
  file_hash    VARCHAR(128),
  mime_type    VARCHAR(100),
  created_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- 9. teacher_evaluations — 教师评价表
CREATE TABLE teacher_evaluations (
  id                 BIGSERIAL PRIMARY KEY,
  user_id            BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  teacher_id         BIGINT    NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
  course_id          BIGINT    REFERENCES courses(id) ON DELETE CASCADE,
  mode               VARCHAR(16) DEFAULT 'single',
  mirror_evaluation_id BIGINT,
  mirror_entity_type VARCHAR(16),
  teaching_score     INTEGER   NOT NULL CHECK (teaching_score BETWEEN 1 AND 5),
  grading_score      INTEGER   NOT NULL CHECK (grading_score BETWEEN 1 AND 5),
  attendance_score   INTEGER   NOT NULL CHECK (attendance_score BETWEEN 1 AND 5),
  workload_score     INTEGER   CHECK (workload_score BETWEEN 1 AND 5),
  gain_score         INTEGER   CHECK (gain_score BETWEEN 1 AND 5),
  difficulty_score   INTEGER   CHECK (difficulty_score BETWEEN 1 AND 5),
  comment            TEXT,
  is_anonymous       BOOLEAN   DEFAULT FALSE,
  status             resource_status DEFAULT 'approved',
  created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 10. course_evaluations — 课程评价表
CREATE TABLE course_evaluations (
  id               BIGSERIAL PRIMARY KEY,
  user_id          BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_id        BIGINT    NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  teacher_id       BIGINT    REFERENCES teachers(id) ON DELETE CASCADE,
  mode             VARCHAR(16) DEFAULT 'single',
  mirror_evaluation_id BIGINT,
  mirror_entity_type VARCHAR(16),
  workload_score   INTEGER   NOT NULL CHECK (workload_score BETWEEN 1 AND 5),
  gain_score       INTEGER   NOT NULL CHECK (gain_score BETWEEN 1 AND 5),
  difficulty_score INTEGER   NOT NULL CHECK (difficulty_score BETWEEN 1 AND 5),
  teaching_score   INTEGER   CHECK (teaching_score BETWEEN 1 AND 5),
  grading_score    INTEGER   CHECK (grading_score BETWEEN 1 AND 5),
  attendance_score INTEGER   CHECK (attendance_score BETWEEN 1 AND 5),
  comment          TEXT,
  is_anonymous     BOOLEAN   DEFAULT FALSE,
  status           resource_status DEFAULT 'approved',
  created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE teacher_evaluation_replies (
  id                BIGSERIAL PRIMARY KEY,
  evaluation_id     BIGINT    NOT NULL REFERENCES teacher_evaluations(id) ON DELETE CASCADE,
  user_id           BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content           TEXT      NOT NULL,
  is_anonymous      BOOLEAN   DEFAULT FALSE,
  reply_to_reply_id BIGINT    REFERENCES teacher_evaluation_replies(id) ON DELETE SET NULL,
  reply_to_user_id  BIGINT    REFERENCES users(id) ON DELETE SET NULL,
  created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE course_evaluation_replies (
  id                BIGSERIAL PRIMARY KEY,
  evaluation_id     BIGINT    NOT NULL REFERENCES course_evaluations(id) ON DELETE CASCADE,
  user_id           BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content           TEXT      NOT NULL,
  is_anonymous      BOOLEAN   DEFAULT FALSE,
  reply_to_reply_id BIGINT    REFERENCES course_evaluation_replies(id) ON DELETE SET NULL,
  reply_to_user_id  BIGINT    REFERENCES users(id) ON DELETE SET NULL,
  created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 11. tags — 标签表
CREATE TABLE tags (
  id         BIGSERIAL    PRIMARY KEY,
  name       VARCHAR(64)  NOT NULL UNIQUE,
  use_count  INTEGER      DEFAULT 0,
  created_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);

-- 12. resource_tags — 资源标签关联表
CREATE TABLE resource_tags (
  id         BIGSERIAL PRIMARY KEY,
  resource_id BIGINT    NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
  tag_id      BIGINT    NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(resource_id, tag_id)
);

-- 13. download_records — 下载记录表
CREATE TABLE download_records (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  resource_id BIGINT   NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
  points_cost INTEGER DEFAULT 0,
  ip_address INET,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 14. points_records — 积分流水表
CREATE TYPE points_type AS ENUM ('initial', 'checkin', 'upload', 'download', 'invite', 'manual');

CREATE TABLE points_records (
  id          BIGSERIAL   PRIMARY KEY,
  user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type        points_type NOT NULL,
  delta       INTEGER     NOT NULL,
  balance     INTEGER     NOT NULL,
  reason      TEXT,
  related_id  BIGINT,
  created_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP
);

-- 15. favorites — 收藏表
CREATE TYPE favorite_target_type AS ENUM ('resource', 'course', 'teacher');

CREATE TABLE favorites (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_type favorite_target_type NOT NULL,
  target_id  BIGINT    NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, target_type, target_id)
);

-- 16. likes — 点赞表
CREATE TYPE like_target_type AS ENUM ('resource', 'teacher_evaluation', 'course_evaluation', 'teacher_evaluation_reply', 'course_evaluation_reply', 'comment');

CREATE TABLE likes (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_type like_target_type NOT NULL,
  target_id  BIGINT    NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, target_type, target_id)
);

-- 17. comments — 评论表
CREATE TYPE comment_status AS ENUM ('active', 'deleted');
CREATE TYPE comment_target_type AS ENUM ('teacher', 'course', 'resource');

CREATE TABLE comments (
  id               BIGSERIAL    PRIMARY KEY,
  target_type      comment_target_type NOT NULL,
  target_id        BIGINT       NOT NULL,
  user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  parent_id        BIGINT       REFERENCES comments(id),
  reply_to_comment_id BIGINT    REFERENCES comments(id),
  content          TEXT         NOT NULL,
  like_count       INTEGER      DEFAULT 0,
  status           comment_status DEFAULT 'active',
  created_at       TIMESTAMPTZ  DEFAULT now(),
  updated_at       TIMESTAMPTZ  DEFAULT now(),
  deleted_at       TIMESTAMPTZ
);

-- 18. reports — 举报表
CREATE TYPE report_target_type AS ENUM ('resource', 'teacher_evaluation', 'course_evaluation', 'teacher_evaluation_reply', 'course_evaluation_reply', 'comment');
CREATE TYPE report_status AS ENUM ('pending', 'resolved', 'dismissed');

CREATE TABLE reports (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_type report_target_type NOT NULL,
  target_id   BIGINT    NOT NULL,
  reason      TEXT      NOT NULL,
  description TEXT,
  status      report_status DEFAULT 'pending',
  processor_id BIGINT   REFERENCES users(id),
  process_at  TIMESTAMP,
  process_note TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 19. corrections — 纠错表
CREATE TYPE correction_target_type AS ENUM ('course', 'teacher');
CREATE TYPE correction_status AS ENUM ('pending', 'accepted', 'rejected');

CREATE TABLE corrections (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_type correction_target_type NOT NULL,
  target_id   BIGINT    NOT NULL,
  field       VARCHAR(64) NOT NULL,
  suggested_value TEXT,
  status      correction_status DEFAULT 'pending',
  processor_id BIGINT   REFERENCES users(id),
  process_at  TIMESTAMP,
  process_note TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 20. notifications — 通知表
CREATE TYPE notification_type AS ENUM ('audit', 'liked', 'commented', 'report_handled', 'correction_handled', 'points_changed');

CREATE TABLE notifications (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type        notification_type NOT NULL,
  title       VARCHAR(255) NOT NULL,
  content     TEXT,
  related_id  BIGINT,
  is_read     BOOLEAN    DEFAULT FALSE,
  is_global   BOOLEAN    DEFAULT FALSE,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 21. search_histories — 搜索历史表
CREATE TABLE search_histories (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  keyword    VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, keyword)
);

-- 22. feedbacks — 用户反馈表
CREATE TYPE feedback_type AS ENUM ('bug', 'suggestion', 'complaint', 'other');
CREATE TYPE feedback_status AS ENUM ('pending', 'processing', 'resolved', 'closed');

CREATE TABLE feedbacks (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type        feedback_type NOT NULL,
  title       VARCHAR(255) NOT NULL,
  content     TEXT      NOT NULL,
  attachments JSONB,
  status      feedback_status DEFAULT 'pending',
  replied_by  BIGINT    REFERENCES users(id),
  replied_at  TIMESTAMP,
  reply       TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 23. invitations — 邀请表
create type invitation_status as enum ('pending', 'invited');

CREATE TABLE invitations (
  id         BIGSERIAL PRIMARY KEY,
  inviter_id BIGINT    NOT NULL REFERENCES users(id),
  invitee_id BIGINT    REFERENCES users(id),
  code       VARCHAR(32) NOT NULL UNIQUE,
  status     invitation_status DEFAULT 'pending',
  expires_at TIMESTAMP NOT NULL,
  used_at    TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 24. audit_logs — 审计日志表
CREATE TYPE audit_action AS ENUM ('approve', 'reject', 'delete', 'ban', 'unban', 'manual_adjust_points');

CREATE TABLE audit_logs (
  id          BIGSERIAL PRIMARY KEY,
  operator_id BIGINT    NOT NULL REFERENCES users(id),
  action      audit_action NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id   BIGINT,
  old_values  JSONB,
  new_values  JSONB,
  reason      TEXT,
  ip_address  INET,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 25. announcements — 系统公告表
CREATE TYPE announcement_type AS ENUM ('notice', 'maintenance', 'feature');

CREATE TABLE announcements (
  id          BIGSERIAL PRIMARY KEY,
  title       VARCHAR(255) NOT NULL,
  content     TEXT        NOT NULL,
  type        announcement_type NOT NULL,
  is_pinned   BOOLEAN     DEFAULT FALSE,
  is_published BOOLEAN    DEFAULT FALSE,
  published_at TIMESTAMP,
  expires_at  TIMESTAMP,
  created_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP
);

-- 26. teacher_rankings — 教师排行榜缓存表
CREATE TABLE teacher_rankings (
  id              BIGSERIAL PRIMARY KEY,
  teacher_id      BIGINT    NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
  department_id   SMALLINT  NOT NULL REFERENCES departments(id),
  period          VARCHAR(16) NOT NULL,
  dimension       VARCHAR(32) NOT NULL,
  rank            INTEGER   NOT NULL,
  score           NUMERIC(10,2),
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(teacher_id, period, dimension)
);

-- 27. course_rankings — 课程排行榜缓存表
CREATE TABLE course_rankings (
  id              BIGSERIAL PRIMARY KEY,
  course_id       BIGINT    NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  period          VARCHAR(16) NOT NULL,
  dimension       VARCHAR(32) NOT NULL,
  rank            INTEGER   NOT NULL,
  score           NUMERIC(10,2),
  updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(course_id, period, dimension)
);

-- 28. hot_keywords — 热搜词表
CREATE TABLE hot_keywords (
  id          BIGSERIAL PRIMARY KEY,
  keyword     VARCHAR(255) NOT NULL,
  period      VARCHAR(16) NOT NULL,
  count       INTEGER      NOT NULL DEFAULT 1,
  updated_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(keyword, period)
);

-- Performance indexes for startup aggregation and cache refresh
CREATE INDEX IF NOT EXISTS idx_teacher_evaluations_approved_teacher_created
  ON teacher_evaluations (teacher_id, created_at DESC)
  WHERE status = 'approved';

CREATE INDEX IF NOT EXISTS idx_course_evaluations_approved_course_created
  ON course_evaluations (course_id, created_at DESC)
  WHERE status = 'approved';

CREATE INDEX IF NOT EXISTS idx_resources_approved_course
  ON resources (course_id)
  WHERE status = 'approved';

CREATE INDEX IF NOT EXISTS idx_resources_uploader_created_at
  ON resources (uploader_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_teacher_evaluations_user_created_at
  ON teacher_evaluations (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_course_evaluations_user_created_at
  ON course_evaluations (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_points_records_user_type_created_at
  ON points_records (user_id, type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_comments_active_target
  ON comments (target_type, target_id)
  WHERE status = 'active';

-- N. mail_providers — 管理端可配置的出站邮件通道
-- kind: aliyun_dm(阿里云邮件推送 DirectMail) | tencent_ses(腾讯云邮件推送 SES) | custom_smtp(自填邮箱)
-- 云通道恒排在 custom_smtp 之前，tier 只在同一 kind 内部排序。
-- password 以 AES-GCM 加密存储，接口永不回传明文。
-- 本表为空时后端自动回落到 config.yaml 里的 mail.verification.providers。
CREATE TABLE IF NOT EXISTS mail_providers (
  id              BIGINT PRIMARY KEY,
  name            VARCHAR(64)  NOT NULL,
  kind            VARCHAR(32)  NOT NULL DEFAULT 'custom_smtp',
  host            VARCHAR(255) NOT NULL,
  port            INTEGER      NOT NULL,
  tls_mode        VARCHAR(16)  DEFAULT 'implicit',
  username        VARCHAR(255) NOT NULL,
  password        TEXT         NOT NULL,
  from_email_addr VARCHAR(255) NOT NULL,
  from_name       VARCHAR(64),
  tier            INTEGER      NOT NULL DEFAULT 0,
  enabled         BOOLEAN      NOT NULL DEFAULT TRUE,
  last_ok_at      TIMESTAMPTZ,
  last_err_at     TIMESTAMPTZ,
  last_err        TEXT,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mail_providers_deleted_at ON mail_providers (deleted_at);
CREATE INDEX IF NOT EXISTS idx_mail_providers_enabled_tier ON mail_providers (enabled, tier);

-- 29/30/31. wiki_sections / wiki_categories / wiki_documents — Wiki 文档服务端管理
-- 层级：「板块(wiki_sections) → 分组(可选,由 allow_categories 控制) → 文档」。
-- 板块可扩展：新增行即可，section 为 varchar FK，不再用 enum。
-- 「板块是否允许分组」「文档与分组板块一致」由 service 层校验。
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
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS wiki_categories (
  id          BIGSERIAL    PRIMARY KEY,
  section     VARCHAR(32)  NOT NULL DEFAULT 'major' REFERENCES wiki_sections(key),
  name        VARCHAR(64)  NOT NULL,
  sort_order  INTEGER      NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_wiki_categories_section_name
  ON wiki_categories (section, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_wiki_categories_section_sort
  ON wiki_categories (section, sort_order);
CREATE INDEX IF NOT EXISTS idx_wiki_categories_deleted_at
  ON wiki_categories (deleted_at);

CREATE TABLE IF NOT EXISTS wiki_documents (
  id           BIGSERIAL    PRIMARY KEY,
  section      VARCHAR(32)  NOT NULL REFERENCES wiki_sections(key),
  category_id  BIGINT       REFERENCES wiki_categories(id),
  slug         VARCHAR(128) NOT NULL,
  title        VARCHAR(128) NOT NULL,
  content      TEXT         NOT NULL DEFAULT '',
  sort_order   INTEGER      NOT NULL DEFAULT 0,
  is_published BOOLEAN      NOT NULL DEFAULT FALSE,
  published_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_wiki_documents_section_slug
  ON wiki_documents (section, slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_wiki_documents_tree
  ON wiki_documents (section, category_id, sort_order) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_wiki_documents_deleted_at
  ON wiki_documents (deleted_at);
