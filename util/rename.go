package util

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// BatchRenameDirectories 批量重命名目录，去除空格和引号，只保留日期和纯汉字
// basePath: 公众号目录的路径，例如 D:\WechatDownload\浙江宣传\
// 返回修改的目录数量
func BatchRenameDirectories(basePath string) (int, error) {
	// 确保基础路径存在
	_, err := os.Stat(basePath)
	if err != nil {
		return 0, fmt.Errorf("基础路径不存在或无法访问: %v", err)
	}

	// 先获取所有目录信息，避免遍历过程中目录名变化导致的问题
	var directories []string

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0, fmt.Errorf("读取目录失败: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, filepath.Join(basePath, entry.Name()))
		}
	}

	// 计数器
	count := 0

	// 处理每个目录
	for _, dirPath := range directories {
		// 获取目录名
		oldName := filepath.Base(dirPath)

		// 检查目录名是否符合格式："日期 文章名"，通常日期为 YYYY-MM-DD 格式
		datePattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)
		matches := datePattern.FindStringSubmatch(oldName)
		if len(matches) == 0 {
			fmt.Printf("跳过目录 '%s': 不符合日期格式\n", oldName)
			continue
		}

		// 提取日期部分
		date := matches[1]

		// 剩余部分为文章名，去除日期和空格
		title := strings.TrimPrefix(oldName, date)
		title = strings.TrimSpace(title)

		// 去除引号
		title = strings.ReplaceAll(title, "\"", "")
		title = strings.ReplaceAll(title, "'", "")

		// 只保留汉字
		var chineseOnly strings.Builder
		for _, r := range title {
			if unicode.Is(unicode.Han, r) {
				chineseOnly.WriteRune(r)
			}
		}
		title = chineseOnly.String()

		// 新的目录名：日期+文章名（无空格）
		newName := date + title

		// 检查新名称是否与原名称相同
		if newName == oldName {
			continue
		}

		// 新的路径
		parent := filepath.Dir(dirPath)
		newPath := filepath.Join(parent, newName)

		// 检查新路径是否已存在
		if _, err := os.Stat(newPath); err == nil {
			fmt.Printf("无法重命名 '%s': 目标目录 '%s' 已存在\n", oldName, newName)
			continue
		}

		// 重命名目录
		err = os.Rename(dirPath, newPath)
		if err != nil {
			fmt.Printf("重命名 '%s' 到 '%s' 失败: %v\n", oldName, newName, err)
			continue
		}

		fmt.Printf("已重命名: '%s' -> '%s'\n", oldName, newName)
		count++
	}

	return count, nil
}
