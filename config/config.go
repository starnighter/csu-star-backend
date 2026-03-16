package config

import (
	"log"

	"github.com/spf13/viper"
)

var GlobalConfig *Config

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Snowflake SnowflakeConfig `mapstructure:"snowflake"`
	Tencent   TencentConfig   `mapstructure:"tencent"`
	Oauth     OauthConfig     `mapstructure:"oauth"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	AccessExpiration  int64  `mapstructure:"access_expiration"`
	RefreshExpiration int64  `mapstructure:"refresh_expiration"`
}

type SnowflakeConfig struct {
	NodeID int64 `mapstructure:"nodeID"`
}

type TencentConfig struct {
	SecretID      string `mapstructure:"secret_id"`
	SecretKey     string `mapstructure:"secret_key"`
	FromEmailAddr string `mapstructure:"from_email_addr"`
	Subject       string `mapstructure:"subject"`
	TemplateID    uint64 `mapstructure:"template_id"`
}

type OauthConfig struct {
	QQ     QQConfig     `mapstructure:"qq"`
	Wechat WechatConfig `mapstructure:"wechat"`
}

type QQConfig struct {
	AppID  string `mapstructure:"app_id"`
	AppKey string `mapstructure:"app_key"`
}

type WechatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

func Init() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败：%v", err)
	}
	viper.SetConfigName("config-secret")
	if err := viper.MergeInConfig(); err != nil {
		log.Fatalf("合并配置文件失败：%v", err)
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		log.Fatalf("解析配置文件失败：%v", err)
	}

	log.Println("配置加载成功")
}
