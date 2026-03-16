package main

import (
	"context"
	"csu-star-backend/config"
	"csu-star-backend/pkg/utils"
	"csu-star-backend/router"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 初始化配置
	config.Init()
	globalCfg := config.GlobalConfig

	// 初始化雪花算法及Redis
	utils.InitSnowflake(globalCfg.Snowflake.NodeID)
	utils.InitRedis()

	// 初始化数据库配置
	db, err := gorm.Open(postgres.Open(globalCfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败：%v\n", err)
	}

	// 初始化路由及依赖配置
	r := router.SetUpRouter(db)

	// 配置HTTP Sever
	addr := fmt.Sprintf("CSU-star后端服务正在启动，监听端口：%d\n", globalCfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 启动服务
	go func() {
		log.Printf("CSU-Star后端服务启动成功，监听端口：%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("监听出现错误：%s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill) 信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	<-quit
	log.Println("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("关闭超时，服务强制关闭:", err)
	}

	log.Println("服务监听中......")
}
