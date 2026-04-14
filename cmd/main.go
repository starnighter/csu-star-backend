package main

import (
	"context"
	"csu-star-backend/config"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/task"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"csu-star-backend/router"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 初始化日志配置
	logger.Init()

	// 初始化配置文件
	if err := config.Init(); err != nil {
		logger.Log.Fatal("配置文件初始化失败，服务退出", zap.Error(err))
	}
	globalCfg := config.GlobalConfig

	// 初始化雪花算法及Redis
	utils.InitSnowflake(globalCfg.Snowflake.NodeID)
	utils.InitRedis()

	// 初始化数据库配置
	db, err := gorm.Open(postgres.Open(globalCfg.Database.DSN), &gorm.Config{})
	if err != nil {
		logger.Log.Fatal("数据库连接失败，服务退出", zap.Error(err))
	}
	if err := configureDatabasePool(db, globalCfg.Database); err != nil {
		logger.Log.Fatal("数据库连接池配置失败，服务退出", zap.Error(err))
	}
	if err := relaxResourceTypeConstraint(db); err != nil {
		logger.Log.Error("放开资源类型数据库限制失败：", zap.Error(err))
	}
	if err := ensureResourceDeletedStatus(db); err != nil {
		logger.Log.Error("补齐资源删除状态失败：", zap.Error(err))
	}
	if err := ensureCourseAggregateColumns(db); err != nil {
		logger.Log.Error("补齐课程聚合字段失败：", zap.Error(err))
	}
	if err := ensureTeacherFavoriteAggregateColumn(db); err != nil {
		logger.Log.Error("补齐教师收藏聚合字段失败：", zap.Error(err))
	}
	if err := ensureSocialIndexes(db); err != nil {
		logger.Log.Error("补齐互动表索引失败：", zap.Error(err))
	}
	if err := ensureSearchPerformanceIndexes(db); err != nil {
		logger.Log.Error("补齐搜索性能索引失败：", zap.Error(err))
	}
	if err := ensureEvaluationReplyTables(db); err != nil {
		logger.Log.Error("补齐评价回复表失败：", zap.Error(err))
	}
	if err := removeDeprecatedSemesterColumns(db); err != nil {
		logger.Log.Error("移除废弃学期字段失败：", zap.Error(err))
	}
	if err := ensureTeacherSupplementRequestTable(db); err != nil {
		logger.Log.Error("补齐教师补录申请表失败：", zap.Error(err))
	}
	if err := ensureCourseSupplementRequestTable(db); err != nil {
		logger.Log.Error("补齐课程补录申请表失败：", zap.Error(err))
	}
	if err := ensureCourseTeacherStatusColumns(db); err != nil {
		logger.Log.Error("补齐课程教师状态字段失败：", zap.Error(err))
	}
	if err := ensureTeacherMetadataColumn(db); err != nil {
		logger.Log.Error("补齐教师元数据字段失败：", zap.Error(err))
	}
	if err := ensureEvaluationVisibilityIndexes(db); err != nil {
		logger.Log.Error("补齐评价可见性索引失败：", zap.Error(err))
	}
	if err := ensureAnnouncementSoftDelete(db); err != nil {
		logger.Log.Error("补齐公告软删除字段失败：", zap.Error(err))
	}
	if err := ensureNotificationInfrastructure(db); err != nil {
		logger.Log.Error("补齐通知基础设施失败：", zap.Error(err))
	}
	if err := ensureUserSecurityInfrastructure(db); err != nil {
		logger.Log.Error("补齐风控与封禁基础设施失败：", zap.Error(err))
	}
	if err := ensureAuditLogActions(db); err != nil {
		logger.Log.Error("补齐审计动作枚举失败：", zap.Error(err))
	}

	// 初始化OAuth Client
	oauthClient := utils.NewHttpClient(10*time.Second, 100)

	// 初始化腾讯云相关组件
	err = utils.InitTencentCos()
	if err != nil {
		logger.Log.Fatal("腾讯云COS客户端初始化失败，服务退出", zap.Error(err))
	}
	if !utils.HasVerificationEmailFallbackProvider() {
		logger.Log.Error("验证码邮件SMTP通道不可用：未配置可用的SMTP provider")
	}

	// 初始化路由及依赖配置
	trustedProxies := resolveTrustedProxies(globalCfg.Server.TrustedProxies)
	r, err := router.SetUpRouter(db, oauthClient, trustedProxies)
	if err != nil {
		logger.Log.Fatal("路由初始化失败，服务退出", zap.Error(err))
	}

	// 初始化定时任务
	aggregateRepo := repo.NewAggregateRepository(db)
	courseRepo := repo.NewCourseRepository(db)
	teacherRepo := repo.NewTeacherRepository(db)
	miscRepo := repo.NewMiscRepository(db)
	appCtx, cancelBackgroundTasks := context.WithCancel(context.Background())
	scheduler := task.NewScheduler(db, aggregateRepo, courseRepo, teacherRepo, miscRepo)
	scheduler.Start(appCtx)

	// 配置HTTP Sever
	addr := fmt.Sprintf("%s:%v", resolveBindHost(globalCfg.Server.BindHost), globalCfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       durationOrDefault(globalCfg.Server.ReadTimeoutSec, 10*time.Second),
		WriteTimeout:      durationOrDefault(globalCfg.Server.WriteTimeoutSec, 30*time.Second),
		IdleTimeout:       durationOrDefault(globalCfg.Server.IdleTimeoutSec, 60*time.Second),
		ReadHeaderTimeout: durationOrDefault(globalCfg.Server.ReadHeaderTimeout, 5*time.Second),
	}

	// 启动服务
	go func() {
		logger.Log.Info("CSU-Star后端服务启动成功，监听端口：" + addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("监听出现错误：", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill) 信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	<-quit
	logger.Log.Info("正在关闭服务......")
	cancelBackgroundTasks()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("关闭超时，服务强制关闭：", zap.Error(err))
	}

	logger.Log.Info("服务关闭成功")
}

func configureDatabasePool(db *gorm.DB, cfg config.DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 40
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 20
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}

	connMaxLifetime := time.Duration(cfg.ConnMaxLifetimeMin) * time.Minute
	if connMaxLifetime <= 0 {
		connMaxLifetime = 30 * time.Minute
	}
	connMaxIdleTime := time.Duration(cfg.ConnMaxIdleTimeMin) * time.Minute
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 10 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	return nil
}

func durationOrDefault(seconds int, fallback time.Duration) time.Duration {
	if seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func resolveBindHost(bindHost string) string {
	bindHost = strings.TrimSpace(bindHost)
	if bindHost != "" {
		return bindHost
	}
	if ginMode := strings.TrimSpace(strings.ToLower(os.Getenv("GIN_MODE"))); ginMode == "release" {
		return "127.0.0.1"
	}
	return "0.0.0.0"
}

func resolveTrustedProxies(configured []string) []string {
	result := make([]string, 0, len(configured))
	for _, item := range configured {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	if len(result) > 0 {
		return result
	}
	return []string{"127.0.0.1", "::1"}
}

func relaxResourceTypeConstraint(db *gorm.DB) error {
	return db.Exec(`
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
					AND table_name = 'resources'
					AND column_name = 'type'
					AND udt_name = 'resource_type'
			) THEN
				ALTER TABLE resources
				ALTER COLUMN type TYPE varchar(64)
				USING type::text;
			END IF;
		END
		$$;
	`).Error
}

func ensureCourseAggregateColumns(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE courses
			ADD COLUMN IF NOT EXISTS download_total INTEGER DEFAULT 0,
			ADD COLUMN IF NOT EXISTS view_total INTEGER DEFAULT 0,
			ADD COLUMN IF NOT EXISTS like_total INTEGER DEFAULT 0,
			ADD COLUMN IF NOT EXISTS favorite_count INTEGER DEFAULT 0,
			ADD COLUMN IF NOT EXISTS resource_favorite_count INTEGER DEFAULT 0;
	`).Error
}

func ensureTeacherFavoriteAggregateColumn(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE teachers
			ADD COLUMN IF NOT EXISTS favorite_count INTEGER DEFAULT 0;
	`).Error
}

func ensureSocialIndexes(db *gorm.DB) error {
	return db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_favorites_target_type_target_id
			ON favorites (target_type, target_id);
		CREATE INDEX IF NOT EXISTS idx_favorites_user_target_type_target_id
			ON favorites (user_id, target_type, target_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_user_target_type_target_id_unique
			ON favorites (user_id, target_type, target_id);
		CREATE INDEX IF NOT EXISTS idx_likes_user_target_type_target_id
			ON likes (user_id, target_type, target_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_user_target_type_target_id_unique
			ON likes (user_id, target_type, target_id);
	`).Error
}

func ensureResourceDeletedStatus(db *gorm.DB) error {
	return db.Exec(`
		DO $$
		BEGIN
			ALTER TYPE resource_status ADD VALUE IF NOT EXISTS 'deleted';
		EXCEPTION
			WHEN duplicate_object THEN NULL;
		END
		$$;
	`).Error
}

func ensureSearchPerformanceIndexes(db *gorm.DB) error {
	return db.Exec(`
		CREATE EXTENSION IF NOT EXISTS pg_trgm;

		CREATE INDEX IF NOT EXISTS idx_courses_name_trgm
			ON courses USING gin(name gin_trgm_ops);

		CREATE INDEX IF NOT EXISTS idx_teachers_name_trgm
			ON teachers USING gin(name gin_trgm_ops);
	`).Error
}

func ensureEvaluationReplyTables(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS teacher_evaluation_replies (
			id BIGINT PRIMARY KEY,
			evaluation_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			content TEXT NOT NULL,
			reply_to_reply_id BIGINT,
			reply_to_user_id BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_teacher_evaluation_replies_evaluation_id
			ON teacher_evaluation_replies (evaluation_id);
		CREATE INDEX IF NOT EXISTS idx_teacher_evaluation_replies_user_id
			ON teacher_evaluation_replies (user_id);
		CREATE INDEX IF NOT EXISTS idx_teacher_evaluation_replies_reply_to_reply_id
			ON teacher_evaluation_replies (reply_to_reply_id);
		CREATE INDEX IF NOT EXISTS idx_teacher_evaluation_replies_reply_to_user_id
			ON teacher_evaluation_replies (reply_to_user_id);
		CREATE INDEX IF NOT EXISTS idx_teacher_evaluation_replies_created_at
			ON teacher_evaluation_replies (created_at);

		CREATE TABLE IF NOT EXISTS course_evaluation_replies (
			id BIGINT PRIMARY KEY,
			evaluation_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			content TEXT NOT NULL,
			reply_to_reply_id BIGINT,
			reply_to_user_id BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_course_evaluation_replies_evaluation_id
			ON course_evaluation_replies (evaluation_id);
		CREATE INDEX IF NOT EXISTS idx_course_evaluation_replies_user_id
			ON course_evaluation_replies (user_id);
		CREATE INDEX IF NOT EXISTS idx_course_evaluation_replies_reply_to_reply_id
			ON course_evaluation_replies (reply_to_reply_id);
		CREATE INDEX IF NOT EXISTS idx_course_evaluation_replies_reply_to_user_id
			ON course_evaluation_replies (reply_to_user_id);
		CREATE INDEX IF NOT EXISTS idx_course_evaluation_replies_created_at
			ON course_evaluation_replies (created_at);
	`).Error
}

func ensureUserSecurityInfrastructure(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE users
			ADD COLUMN IF NOT EXISTS ban_until TIMESTAMPTZ,
			ADD COLUMN IF NOT EXISTS ban_reason VARCHAR(255) DEFAULT '',
			ADD COLUMN IF NOT EXISTS ban_source VARCHAR(32) DEFAULT '',
			ADD COLUMN IF NOT EXISTS violation_count INTEGER DEFAULT 0,
			ADD COLUMN IF NOT EXISTS last_violation_at TIMESTAMPTZ;

		CREATE TABLE IF NOT EXISTS user_violations (
			id BIGINT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			scope VARCHAR(64) NOT NULL,
			trigger_key VARCHAR(255) DEFAULT '',
			reason TEXT NOT NULL,
			evidence JSONB DEFAULT '{}'::jsonb,
			penalty_level INTEGER DEFAULT 0,
			ban_duration_seconds BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_user_violations_user_id_created_at
			ON user_violations (user_id, created_at DESC);
	`).Error
}

func ensureAuditLogActions(db *gorm.DB) error {
	return db.Exec(`
		DO $$
		BEGIN
			ALTER TYPE audit_action ADD VALUE IF NOT EXISTS 'create';
			ALTER TYPE audit_action ADD VALUE IF NOT EXISTS 'update';
			ALTER TYPE audit_action ADD VALUE IF NOT EXISTS 'auto_violation';
			ALTER TYPE audit_action ADD VALUE IF NOT EXISTS 'auto_ban';
			ALTER TYPE audit_action ADD VALUE IF NOT EXISTS 'auto_unban';
		EXCEPTION
			WHEN duplicate_object THEN NULL;
		END
		$$;
	`).Error
}

func removeDeprecatedSemesterColumns(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE IF EXISTS course_teachers
			DROP COLUMN IF EXISTS semester;

		ALTER TABLE IF EXISTS resources
			DROP COLUMN IF EXISTS semester_start,
			DROP COLUMN IF EXISTS semester_end;
	`).Error
}

func ensureTeacherSupplementRequestTable(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS teacher_supplement_requests (
			id BIGINT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			contact VARCHAR(128) NOT NULL,
			teacher_name VARCHAR(128) NOT NULL,
			department_id SMALLINT NOT NULL,
			remark TEXT,
			reviewed_by BIGINT,
			reviewed_at TIMESTAMPTZ,
			review_note TEXT,
			approved_teacher_id BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT chk_teacher_supplement_request_status CHECK (status IN ('pending', 'approved', 'rejected'))
		);

		ALTER TABLE teacher_supplement_requests
			ADD COLUMN IF NOT EXISTS teacher_name VARCHAR(128),
			ADD COLUMN IF NOT EXISTS department_id SMALLINT,
			ADD COLUMN IF NOT EXISTS remark TEXT,
			ADD COLUMN IF NOT EXISTS reviewed_by BIGINT,
			ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
			ADD COLUMN IF NOT EXISTS review_note TEXT,
			ADD COLUMN IF NOT EXISTS approved_teacher_id BIGINT,
			ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

		CREATE INDEX IF NOT EXISTS idx_teacher_supplement_requests_status_created_at
			ON teacher_supplement_requests (status, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_teacher_supplement_requests_user_id_created_at
			ON teacher_supplement_requests (user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_teacher_supplement_requests_department_status_created_at
			ON teacher_supplement_requests (department_id, status, created_at DESC);
	`).Error
}

func ensureCourseSupplementRequestTable(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS course_supplement_requests (
			id BIGINT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			contact VARCHAR(128) NOT NULL,
			course_name VARCHAR(128) NOT NULL,
			course_type VARCHAR(16) NOT NULL,
			remark TEXT,
			reviewed_by BIGINT,
			reviewed_at TIMESTAMPTZ,
			review_note TEXT,
			approved_course_id BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT chk_course_supplement_request_status CHECK (status IN ('pending', 'approved', 'rejected')),
			CONSTRAINT chk_course_supplement_request_course_type CHECK (course_type IN ('public', 'non_public'))
		);

		ALTER TABLE course_supplement_requests
			ADD COLUMN IF NOT EXISTS course_name VARCHAR(128),
			ADD COLUMN IF NOT EXISTS course_type VARCHAR(16),
			ADD COLUMN IF NOT EXISTS remark TEXT,
			ADD COLUMN IF NOT EXISTS reviewed_by BIGINT,
			ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
			ADD COLUMN IF NOT EXISTS review_note TEXT,
			ADD COLUMN IF NOT EXISTS approved_course_id BIGINT,
			ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

		ALTER TABLE course_supplement_requests
			DROP CONSTRAINT IF EXISTS chk_course_supplement_request_course_type;

		ALTER TABLE course_supplement_requests
			ADD CONSTRAINT chk_course_supplement_request_course_type CHECK (
				course_type IN ('public', 'non_public')
			);

		CREATE INDEX IF NOT EXISTS idx_course_supplement_requests_status_created_at
			ON course_supplement_requests (status, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_course_supplement_requests_user_id_created_at
			ON course_supplement_requests (user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_course_supplement_requests_course_type_status_created_at
			ON course_supplement_requests (course_type, status, created_at DESC);
	`).Error
}

func ensureCourseTeacherStatusColumns(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE courses
			ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'active';

		ALTER TABLE teachers
			ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'active';

		ALTER TABLE course_teachers
			ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'active',
			ADD COLUMN IF NOT EXISTS canceled_at TIMESTAMPTZ,
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

		UPDATE courses SET status = 'active' WHERE status IS NULL OR status = '';
		UPDATE teachers SET status = 'active' WHERE status IS NULL OR status = '';
		UPDATE course_teachers
		SET
			status = 'active',
			updated_at = COALESCE(updated_at, NOW())
		WHERE status IS NULL OR status = '';

		CREATE INDEX IF NOT EXISTS idx_courses_status ON courses (status);
		CREATE INDEX IF NOT EXISTS idx_teachers_status ON teachers (status);
		CREATE INDEX IF NOT EXISTS idx_course_teachers_course_status ON course_teachers (course_id, status);
		CREATE INDEX IF NOT EXISTS idx_course_teachers_teacher_status ON course_teachers (teacher_id, status);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_course_teachers_course_teacher_unique
			ON course_teachers (course_id, teacher_id);
	`).Error
}

func ensureTeacherMetadataColumn(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE teachers
			ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

		UPDATE teachers
		SET metadata = '{}'::jsonb
		WHERE metadata IS NULL;
	`).Error
}

func ensureEvaluationVisibilityIndexes(db *gorm.DB) error {
	return db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_course_teachers_course_teacher_status
			ON course_teachers (course_id, teacher_id, status);
		CREATE INDEX IF NOT EXISTS idx_teacher_evaluations_teacher_status_mode_course
			ON teacher_evaluations (teacher_id, status, mode, course_id);
		CREATE INDEX IF NOT EXISTS idx_course_evaluations_course_status_mode_teacher
			ON course_evaluations (course_id, status, mode, teacher_id);
		CREATE INDEX IF NOT EXISTS idx_teacher_evaluations_user_mode_course_teacher
			ON teacher_evaluations (user_id, mode, course_id, teacher_id);
		CREATE INDEX IF NOT EXISTS idx_course_evaluations_user_mode_course_teacher
			ON course_evaluations (user_id, mode, course_id, teacher_id);
	`).Error
}

func ensureAnnouncementSoftDelete(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE announcements
			ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

		CREATE INDEX IF NOT EXISTS idx_announcements_deleted_at
			ON announcements (deleted_at);
	`).Error
}

func ensureNotificationInfrastructure(db *gorm.DB) error {
	return db.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_type') THEN
				ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'system';
			END IF;
		EXCEPTION
			WHEN duplicate_object THEN NULL;
		END
		$$;

		ALTER TABLE notifications
			ADD COLUMN IF NOT EXISTS category VARCHAR(32),
			ADD COLUMN IF NOT EXISTS result VARCHAR(16),
			ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

		UPDATE notifications
		SET
			category = CASE
				WHEN is_global = TRUE THEN 'announcement'
				WHEN type IN ('liked', 'commented') THEN 'interaction'
				WHEN type = 'report_handled' THEN 'report'
				WHEN type = 'correction_handled' THEN 'correction'
				WHEN type = 'points_changed' THEN 'points'
				ELSE 'admin_message'
			END,
			result = CASE
				WHEN is_global = TRUE THEN 'inform'
				WHEN type IN ('liked', 'commented', 'points_changed') THEN 'inform'
				WHEN type = 'report_handled' AND content ILIKE '%dismissed%' THEN 'rejected'
				WHEN type = 'report_handled' THEN 'approved'
				WHEN type = 'correction_handled' AND content ILIKE '%rejected%' THEN 'rejected'
				WHEN type = 'correction_handled' THEN 'approved'
				WHEN title ILIKE '%未通过%' OR content ILIKE '%未通过%' THEN 'rejected'
				WHEN title ILIKE '%通过%' OR content ILIKE '%通过%' THEN 'approved'
				ELSE 'inform'
			END,
			metadata = COALESCE(metadata, '{}'::jsonb)
		WHERE
			category IS NULL OR category = ''
			OR result IS NULL OR result = ''
			OR metadata IS NULL;

		ALTER TABLE notifications
			ALTER COLUMN category SET DEFAULT 'admin_message',
			ALTER COLUMN result SET DEFAULT 'inform',
			ALTER COLUMN metadata SET DEFAULT '{}'::jsonb;

		UPDATE notifications
		SET category = COALESCE(NULLIF(category, ''), 'admin_message'),
			result = COALESCE(NULLIF(result, ''), 'inform'),
			metadata = COALESCE(metadata, '{}'::jsonb);

		ALTER TABLE notifications
			ALTER COLUMN category SET NOT NULL,
			ALTER COLUMN result SET NOT NULL;

		CREATE TABLE IF NOT EXISTS global_notification_reads (
			notification_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (notification_id, user_id)
		);

		CREATE INDEX IF NOT EXISTS idx_notifications_user_created_at
			ON notifications (user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_notifications_global_created_at
			ON notifications (is_global, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_notifications_category_created_at
			ON notifications (category, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_global_notification_reads_user_id
			ON global_notification_reads (user_id, created_at DESC);
	`).Error
}
