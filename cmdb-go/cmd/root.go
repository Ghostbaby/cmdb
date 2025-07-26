package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile  string
	verbose  bool
	logLevel string
	logger   *zap.Logger
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "cmdb-crawler",
	Short: "CMDB服务树数据爬取工具",
	Long: `CMDB服务树数据爬取工具

这个工具可以从CMDB系统中爬取服务树数据，支持多种输出格式：
- JSON格式：结构化数据，便于程序处理
- YAML格式：人类可读的配置格式
- CSV格式：便于Excel等工具处理

功能特性：
- 支持爬取所有服务树或指定服务树
- 并发爬取，提高效率
- 支持深度限制和统计信息
- 灵活的输出配置`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger()
	},
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认为 ./config/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "日志级别 (debug, info, warn, error)")

	// Viper绑定
	viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level"))
}

// initConfig 初始化配置
func initConfig() {
	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 搜索配置文件
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 环境变量前缀
	viper.SetEnvPrefix("CMDB_CRAWLER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("警告: 未找到配置文件，使用默认配置")
		} else {
			fmt.Printf("读取配置文件出错: %v\n", err)
			os.Exit(1)
		}
	} else {
		if verbose {
			fmt.Printf("使用配置文件: %s\n", viper.ConfigFileUsed())
		}
	}
}

// setDefaults 设置默认值
func setDefaults() {
	// CMDB配置默认值
	viper.SetDefault("cmdb.base_url", "http://localhost:8080")
	viper.SetDefault("cmdb.api_version", "v0.1")
	viper.SetDefault("cmdb.auth.username", "admin")
	viper.SetDefault("cmdb.auth.password", "admin")
	viper.SetDefault("cmdb.request.timeout", "30s")
	viper.SetDefault("cmdb.request.retry_count", 3)
	viper.SetDefault("cmdb.request.retry_wait_time", "1s")

	// 爬取配置默认值
	viper.SetDefault("crawler.service_tree.max_depth", -1)
	viper.SetDefault("crawler.service_tree.page_size", 1000)
	viper.SetDefault("crawler.service_tree.include_statistics", true)
	viper.SetDefault("crawler.concurrency.max_workers", 10)
	viper.SetDefault("crawler.concurrency.request_interval", "100ms")

	// 输出配置默认值
	viper.SetDefault("output.format", "json")
	viper.SetDefault("output.file_path", "./output/service_tree_data.json")
	viper.SetDefault("output.pretty_print", true)

	// 日志配置默认值
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.output", "console")
	viper.SetDefault("logging.file_path", "./logs/cmdb-crawler.log")
}

// initLogger 初始化日志
func initLogger() {
	level := viper.GetString("logging.level")
	if verbose {
		level = "debug"
	}

	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level.SetLevel(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

	// 根据配置选择输出方式
	logOutput := viper.GetString("logging.output")
	if logOutput == "file" {
		logPath := viper.GetString("logging.file_path")
		config.OutputPaths = []string{logPath}
		config.ErrorOutputPaths = []string{logPath}
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
}

// GetLogger 获取日志实例
func GetLogger() *zap.Logger {
	if logger == nil {
		initLogger()
	}
	return logger
}

// AuthConfig 认证配置
type AuthConfig struct {
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
}

// CMDBConfig CMDB配置
type CMDBConfig struct {
	BaseURL    string        `mapstructure:"base_url"`
	APIVersion string        `mapstructure:"api_version"`
	Auth       AuthConfig    `mapstructure:"auth"`
	Request    RequestConfig `mapstructure:"request"`
}

// GetConfig 获取配置
func GetConfig() *Config {
	return &Config{
		CMDB: CMDBConfig{
			BaseURL:    viper.GetString("cmdb.base_url"),
			APIVersion: viper.GetString("cmdb.api_version"),
			Auth: AuthConfig{
				APIKey:    viper.GetString("cmdb.auth.api_key"),
				APISecret: viper.GetString("cmdb.auth.api_secret"),
			},
			Request: RequestConfig{
				Timeout:       viper.GetDuration("cmdb.request.timeout"),
				RetryCount:    viper.GetInt("cmdb.request.retry_count"),
				RetryWaitTime: viper.GetDuration("cmdb.request.retry_wait_time"),
			},
		},
		Crawler: CrawlerConfig{
			ServiceTree: ServiceTreeConfig{
				TargetViews:       viper.GetStringSlice("crawler.service_tree.target_views"),
				MaxDepth:          viper.GetInt("crawler.service_tree.max_depth"),
				PageSize:          viper.GetInt("crawler.service_tree.page_size"),
				IncludeStatistics: viper.GetBool("crawler.service_tree.include_statistics"),
			},
			Concurrency: ConcurrencyConfig{
				MaxWorkers:      viper.GetInt("crawler.concurrency.max_workers"),
				RequestInterval: viper.GetDuration("crawler.concurrency.request_interval"),
			},
		},
		Output: OutputConfig{
			Format:      viper.GetString("output.format"),
			FilePath:    viper.GetString("output.file_path"),
			PrettyPrint: viper.GetBool("output.pretty_print"),
		},
		Logging: LoggingConfig{
			Level:    viper.GetString("logging.level"),
			Output:   viper.GetString("logging.output"),
			FilePath: viper.GetString("logging.file_path"),
		},
	}
}

// Config 配置结构
type Config struct {
	CMDB    CMDBConfig    `mapstructure:"cmdb"`
	Crawler CrawlerConfig `mapstructure:"crawler"`
	Output  OutputConfig  `mapstructure:"output"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type RequestConfig struct {
	Timeout       time.Duration `mapstructure:"timeout"`
	RetryCount    int           `mapstructure:"retry_count"`
	RetryWaitTime time.Duration `mapstructure:"retry_wait_time"`
}

type CrawlerConfig struct {
	ServiceTree ServiceTreeConfig `mapstructure:"service_tree"`
	Concurrency ConcurrencyConfig `mapstructure:"concurrency"`
}

type ServiceTreeConfig struct {
	TargetViews       []string `mapstructure:"target_views"`
	MaxDepth          int      `mapstructure:"max_depth"`
	PageSize          int      `mapstructure:"page_size"`
	IncludeStatistics bool     `mapstructure:"include_statistics"`
}

type ConcurrencyConfig struct {
	MaxWorkers      int           `mapstructure:"max_workers"`
	RequestInterval time.Duration `mapstructure:"request_interval"`
}

type OutputConfig struct {
	Format      string `mapstructure:"format"`
	FilePath    string `mapstructure:"file_path"`
	PrettyPrint bool   `mapstructure:"pretty_print"`
}

type LoggingConfig struct {
	Level    string `mapstructure:"level"`
	Output   string `mapstructure:"output"`
	FilePath string `mapstructure:"file_path"`
}
