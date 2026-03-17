package main

import (
	"context"
	"csu-star-backend/config"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"csu-star-backend/router"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	config.Init()
	globalCfg := config.GlobalConfig

	// 初始化雪花算法及Redis
	utils.InitSnowflake(globalCfg.Snowflake.NodeID)
	utils.InitRedis()

	// 初始化数据库配置
	db, err := gorm.Open(postgres.Open(globalCfg.Database.DSN), &gorm.Config{})
	if err != nil {
		logger.Log.Error("数据库连接失败：", zap.Error(err))
	}

	// 初始化OAuth Client
	oauthClient := utils.NewHttpClient(10*time.Second, 100)

	// 初始化路由及依赖配置
	r := router.SetUpRouter(db, oauthClient)

	// 配置HTTP Sever
	addr := fmt.Sprintf("0.0.0.0:%v", globalCfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("关闭超时，服务强制关闭：", zap.Error(err))
	}

	logger.Log.Info("服务关闭成功")
}
