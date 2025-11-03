package src

import "fmt"

// 这些变量会在编译时通过 -ldflags 注入
var (
	Version   = "dev"     // 版本号
	Commit    = "none"    // Git提交哈希
	BuildTime = "unknown" // 构建时间
	GoVersion = "unknown" // Go版本
)

func VersionLong() string {
	return fmt.Sprintf(
		"Version: %s Git Commit: %s Build Time: %s Go Version: %s",
		Version, Commit, BuildTime, GoVersion,
	)
}
