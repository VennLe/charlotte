package cmd

import (
	"fmt"
	"runtime"
	"time"
)

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Version   string
	BuildTime string
	GoVersion string
	Platform  string
}

// GetVersionInfo 获取完整的版本信息
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// PrintVersion 打印版本信息
func PrintVersion() {
	info := GetVersionInfo()

	fmt.Println("Charlotte API - 企业级 Go API 服务")
	fmt.Println("=========================================")
	fmt.Printf("版本:     %s\n", info.Version)
	fmt.Printf("构建时间: %s\n", info.BuildTime)
	fmt.Printf("Go 版本:  %s\n", info.GoVersion)
	fmt.Printf("平台:     %s\n", info.Platform)
	fmt.Println("=========================================")
}

// PrintDetailedVersion 打印详细版本信息
func PrintDetailedVersion() {
	info := GetVersionInfo()

	fmt.Println("Charlotte API 详细版本信息")
	fmt.Println("==============================")
	fmt.Printf("版本号:       %s\n", info.Version)
	fmt.Printf("构建时间:     %s\n", info.BuildTime)

	// 解析构建时间
	if info.BuildTime != "unknown" {
		if buildTime, err := time.Parse(time.RFC3339, info.BuildTime); err == nil {
			fmt.Printf("构建时间:     %s\n", buildTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("运行时长:     %s\n", time.Since(buildTime).Round(time.Second))
		}
	}

	fmt.Printf("Go 版本:      %s\n", info.GoVersion)
	fmt.Printf("操作系统:     %s\n", runtime.GOOS)
	fmt.Printf("架构:         %s\n", runtime.GOARCH)
	fmt.Printf("CPU 核心数:   %d\n", runtime.NumCPU())
	fmt.Printf("Goroutine 数: %d\n", runtime.NumGoroutine())
	fmt.Println("==============================")
}

// GetVersionJSON 获取JSON格式的版本信息
func GetVersionJSON() string {
	info := GetVersionInfo()
	return fmt.Sprintf(`{
  "name": "Charlotte API",
  "version": "%s",
  "build_time": "%s", 
  "go_version": "%s",
  "platform": "%s"
}`, info.Version, info.BuildTime, info.GoVersion, info.Platform)
}
