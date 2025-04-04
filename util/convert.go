package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fengxxc/wechatmp2markdown/parse"
)

// ConvertHTMLFileToTxt 将单个HTML文件转换为TXT
// htmlFilePath: HTML文件路径
// outputPath: 输出路径，可以是目录或文件路径
// 返回生成的TXT文件路径
func ConvertHTMLFileToTxt(htmlFilePath string, outputPath string) (string, error) {
	// 检查HTML文件是否存在
	if _, err := os.Stat(htmlFilePath); err != nil {
		return "", fmt.Errorf("HTML文件不存在或无法访问: %v", err)
	}

	// 解析HTML文件
	articleStruct := parse.ParseFromHTMLFile(htmlFilePath, parse.IMAGE_POLICY_URL)

	// 获取标题作为文件名
	title := strings.TrimSpace(articleStruct.Title.Val.(string))

	// 确定输出文件路径
	var txtFilePath string

	// 检查输出路径是目录还是文件
	fileInfo, err := os.Stat(outputPath)
	if err == nil && fileInfo.IsDir() {
		// 如果是目录，则在该目录下创建以标题命名的TXT文件
		txtFilePath = filepath.Join(outputPath, title+".txt")
	} else if strings.HasSuffix(strings.ToLower(outputPath), ".txt") {
		// 如果指定了TXT文件名，则直接使用
		txtFilePath = outputPath
	} else {
		// 否则，在指定路径下创建以标题命名的TXT文件
		txtFilePath = outputPath + "/" + title + ".txt"
	}

	// 提取文本内容
	result := extractTextContent(articleStruct)

	// 保存TXT文件
	err = saveTxtFile(txtFilePath, result)
	if err != nil {
		return "", fmt.Errorf("保存TXT文件失败: %v", err)
	}

	return txtFilePath, nil
}
