package util

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fengxxc/wechatmp2markdown/parse"
)

// BatchConvertHTMLFiles 批量转换公众号目录下所有子目录中的HTML文件为Markdown
// basePath: 公众号目录的路径，例如 D:\WechatDownload\浙江宣传\
// imagePolicy: 图片处理策略
// 返回成功转换的文件数量
func BatchConvertHTMLFiles(basePath string, imagePolicy parse.ImagePolicy) (int, error) {
	// 确保基础路径存在
	_, err := os.Stat(basePath)
	if err != nil {
		return 0, fmt.Errorf("基础路径不存在或无法访问: %v", err)
	}

	// 获取所有子目录
	var subdirectories []string

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0, fmt.Errorf("读取目录失败: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subdirectories = append(subdirectories, filepath.Join(basePath, entry.Name()))
		}
	}

	// 计数器
	count := 0

	// 处理每个子目录
	for _, dirPath := range subdirectories {
		// 检查是否有index.html或其他HTML文件
		htmlFiles, err := findHTMLFiles(dirPath)
		if err != nil {
			fmt.Printf("搜索目录 '%s' 中的HTML文件失败: %v\n", dirPath, err)
			continue
		}

		if len(htmlFiles) == 0 {
			fmt.Printf("目录 '%s' 中没有找到HTML文件，跳过\n", dirPath)
			continue
		}

		// 优先处理index.html，如果存在
		var htmlFile string
		for _, file := range htmlFiles {
			if strings.ToLower(filepath.Base(file)) == "index.html" {
				htmlFile = file
				break
			}
		}

		// 如果没有index.html，则使用第一个找到的HTML文件
		if htmlFile == "" {
			htmlFile = htmlFiles[0]
		}

		// 获取文章标题作为Markdown文件名
		fmt.Printf("开始处理: %s\n", htmlFile)
		articleStruct := parse.ParseFromHTMLFile(htmlFile, imagePolicy)
		title := strings.TrimSpace(articleStruct.Title.Val.(string))

		// 创建Markdown文件路径 - 将所有内容保存在同目录下
		mdFilePath := filepath.Join(dirPath, title+".md")

		// 格式化Markdown内容
		result, saveImageBytes := formatArticle(articleStruct)

		// 保存Markdown文件和图片
		err = saveMDFile(mdFilePath, result, saveImageBytes, dirPath)
		if err != nil {
			fmt.Printf("保存转换结果失败 '%s': %v\n", mdFilePath, err)
			continue
		}

		fmt.Printf("已转换: '%s' -> '%s'\n", htmlFile, mdFilePath)
		count++
	}

	return count, nil
}

// formatArticle 格式化文章为Markdown文本
func formatArticle(article parse.Article) (string, map[string][]byte) {
	var result string
	var titleMdStr string = formatTitle(article.Title)
	result += titleMdStr

	var saveImageBytes map[string][]byte = make(map[string][]byte)
	contentMdStr, contentImgBytes := formatContent(article.Content, 0)
	result += contentMdStr

	// 合并收集到的图片数据
	for k, v := range contentImgBytes {
		saveImageBytes[k] = v
	}

	return result, saveImageBytes
}

// formatTitle 格式化标题
func formatTitle(piece parse.Piece) string {
	var prefix string
	level, _ := strconv.Atoi(piece.Attrs["level"])
	for i := 0; i < level; i++ {
		prefix += "#"
	}
	return prefix + " " + piece.Val.(string) + "  \n"
}

// formatContent 格式化内容，正确处理图片
func formatContent(pieces []parse.Piece, depth int) (string, map[string][]byte) {
	var contentMdStr string
	var base64Imgs []string
	var saveImageBytes map[string][]byte = make(map[string][]byte)

	for _, piece := range pieces {
		var pieceMdStr string
		var patchSaveImageBytes map[string][]byte

		switch piece.Type {
		case parse.HEADER:
			pieceMdStr = formatTitle(piece)
		case parse.LINK:
			pieceMdStr = formatLink(piece)
		case parse.NORMAL_TEXT:
			if str, ok := piece.Val.(string); ok {
				pieceMdStr = str
			}
		case parse.BOLD_TEXT:
			if str, ok := piece.Val.(string); ok {
				pieceMdStr = "**" + str + "**"
			}
		case parse.ITALIC_TEXT:
			if str, ok := piece.Val.(string); ok {
				pieceMdStr = "*" + str + "*"
			}
		case parse.BOLD_ITALIC_TEXT:
			if str, ok := piece.Val.(string); ok {
				pieceMdStr = "***" + str + "***"
			}
		case parse.IMAGE:
			if piece.Val == nil {
				pieceMdStr = formatImageInline(piece)
			} else {
				// 将保存图片到本地
				src := piece.Attrs["src"]
				imgExt := parseImageExtFromSrc(src)
				var hashName string = calculateMD5(piece.Val.([]byte)) + "." + imgExt
				saveImageBytes[hashName] = piece.Val.([]byte)
				pieceMdStr = formatImageFileReferInline(piece.Attrs["alt"], hashName)
			}
		case parse.IMAGE_BASE64:
			pieceMdStr = formatImageRefer(piece, len(base64Imgs))
			base64Imgs = append(base64Imgs, piece.Val.(string))
		case parse.TABLE:
			pieceMdStr = formatTable(piece)
		case parse.CODE_BLOCK:
			pieceMdStr = formatCodeBlock(piece)
		case parse.BLOCK_QUOTES:
			pieceMdStr, patchSaveImageBytes = formatBlockQuote(piece, depth)
		case parse.O_LIST:
			pieceMdStr, patchSaveImageBytes = formatList(piece, depth)
		case parse.U_LIST:
			pieceMdStr, patchSaveImageBytes = formatList(piece, depth)
		case parse.BR:
			pieceMdStr = "  \n"
		case parse.NULL:
			continue
		}

		contentMdStr += pieceMdStr
		// 合并子元素处理中收集的图片数据
		if patchSaveImageBytes != nil {
			for k, v := range patchSaveImageBytes {
				saveImageBytes[k] = v
			}
		}
	}

	// 添加base64图片引用
	for i := 0; i < len(base64Imgs); i++ {
		contentMdStr += "\n[" + strconv.Itoa(i) + "]:" + "data:image/png;base64," + base64Imgs[i]
	}

	return contentMdStr, saveImageBytes
}

// formatLink 格式化链接
func formatLink(piece parse.Piece) string {
	return "[" + piece.Val.(string) + "](" + piece.Attrs["href"] + ")"
}

// formatImageInline 格式化内联图片引用
func formatImageInline(piece parse.Piece) string {
	return "![" + getOrEmpty(piece.Attrs, "alt") + "](" + getOrEmpty(piece.Attrs, "src") + ")"
}

// formatImageFileReferInline 格式化文件引用图片
func formatImageFileReferInline(alt string, filename string) string {
	if alt == "" {
		alt = filename
	}
	return "![" + alt + "](./" + filename + ")"
}

// formatImageRefer 格式化base64图片引用
func formatImageRefer(piece parse.Piece, index int) string {
	return "![" + getOrEmpty(piece.Attrs, "alt") + "][" + strconv.Itoa(index) + "]"
}

// getOrEmpty 获取map中的值，如果不存在则返回空字符串
func getOrEmpty(m map[string]string, key string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return ""
}

// formatTable 格式化表格
func formatTable(piece parse.Piece) string {
	var tableMdStr string
	if piece.Attrs != nil && piece.Attrs["type"] == "native" {
		tableMdStr = piece.Val.(string)
	}
	return tableMdStr
}

// formatCodeBlock 格式化代码块
func formatCodeBlock(piece parse.Piece) string {
	codeRows := piece.Val.([]string)
	return "```\n" + strings.Join(codeRows, "\n") + "\n```\n"
}

// formatBlockQuote 格式化引用块
func formatBlockQuote(piece parse.Piece, depth int) (string, map[string][]byte) {
	var bqMdString string
	var prefix string = ">"
	for i := 0; i < depth; i++ {
		prefix += ">"
	}
	prefix += " "
	var saveImageBytes map[string][]byte
	bqMdString, saveImageBytes = formatContent(piece.Val.([]parse.Piece), depth+1)
	return prefix + bqMdString + "  \n", saveImageBytes
}

// formatList 格式化列表
func formatList(li parse.Piece, depth int) (string, map[string][]byte) {
	var listMdString string
	var prefix string
	for j := 0; j < depth; j++ {
		prefix += "    "
	}
	if li.Type == parse.U_LIST {
		prefix += "- "
	} else if li.Type == parse.O_LIST {
		prefix += "1. " // 写死成1也没关系，markdown会自动累加序号
	}

	subContent, saveImageBytes := formatContent(li.Val.([]parse.Piece), depth+1)
	listMdString = prefix + subContent + "  \n"
	return listMdString, saveImageBytes
}

// parseImageExtFromSrc 从图片URL解析扩展名
func parseImageExtFromSrc(src string) string {
	if src == "" {
		return "png" // 默认为png
	}

	// 从URL中提取文件名部分
	lastSlashIdx := strings.LastIndex(src, "/")
	var fileName string
	if lastSlashIdx >= 0 && lastSlashIdx < len(src)-1 {
		fileName = src[lastSlashIdx+1:]
	} else {
		fileName = src
	}

	// 移除URL中的查询参数
	questionMarkIdx := strings.LastIndex(fileName, "?")
	if questionMarkIdx > 0 {
		fileName = fileName[:questionMarkIdx]
	}

	// 获取文件扩展名
	lastDotIdx := strings.LastIndex(fileName, ".")
	if lastDotIdx > 0 && lastDotIdx < len(fileName)-1 {
		return fileName[lastDotIdx+1:]
	}

	// 如果无法确定扩展名，返回默认值
	return "png"
}

// calculateMD5 计算字节数组的MD5值，避免导入循环
func calculateMD5(data []byte) string {
	h := md5.New()
	io.WriteString(h, string(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// saveMDFile 保存markdown文件和图片
func saveMDFile(mdFilePath string, content string, images map[string][]byte, basePath string) error {
	// 保存markdown文件
	err := os.WriteFile(mdFilePath, []byte(content), 0o644)
	if err != nil {
		return err
	}

	// 保存图片
	if len(images) > 0 {
		for imgName, imgData := range images {
			imgPath := filepath.Join(basePath, imgName)
			err := os.WriteFile(imgPath, imgData, 0o644)
			if err != nil {
				fmt.Printf("保存图片 '%s' 失败: %v\n", imgPath, err)
			}
		}
	}

	return nil
}

// findHTMLFiles 查找目录中的所有HTML文件
func findHTMLFiles(dirPath string) ([]string, error) {
	var htmlFiles []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if strings.HasSuffix(strings.ToLower(name), ".html") ||
				strings.HasSuffix(strings.ToLower(name), ".htm") {
				htmlFiles = append(htmlFiles, filepath.Join(dirPath, name))
			}
		}
	}

	return htmlFiles, nil
}

// BatchConvertHTMLFilesToTxt 批量转换公众号目录下所有子目录中的HTML文件为纯文本TXT
// basePath: 公众号目录的路径，例如 D:\WechatDownload\浙江宣传\
// 返回成功转换的文件数量
func BatchConvertHTMLFilesToTxt(basePath string) (int, error) {
	// 确保基础路径存在
	_, err := os.Stat(basePath)
	if err != nil {
		return 0, fmt.Errorf("基础路径不存在或无法访问: %v", err)
	}

	// 获取所有子目录
	var subdirectories []string

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0, fmt.Errorf("读取目录失败: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subdirectories = append(subdirectories, filepath.Join(basePath, entry.Name()))
		}
	}

	// 计数器
	count := 0

	// 处理每个子目录
	for _, dirPath := range subdirectories {
		// 检查是否有index.html或其他HTML文件
		htmlFiles, err := findHTMLFiles(dirPath)
		if err != nil {
			fmt.Printf("搜索目录 '%s' 中的HTML文件失败: %v\n", dirPath, err)
			continue
		}

		if len(htmlFiles) == 0 {
			fmt.Printf("目录 '%s' 中没有找到HTML文件，跳过\n", dirPath)
			continue
		}

		// 优先处理index.html，如果存在
		var htmlFile string
		for _, file := range htmlFiles {
			if strings.ToLower(filepath.Base(file)) == "index.html" {
				htmlFile = file
				break
			}
		}

		// 如果没有index.html，则使用第一个找到的HTML文件
		if htmlFile == "" {
			htmlFile = htmlFiles[0]
		}

		// 获取文章标题作为TXT文件名
		fmt.Printf("开始处理: %s\n", htmlFile)
		// 使用任意图片策略，因为我们只需要获取文本内容
		articleStruct := parse.ParseFromHTMLFile(htmlFile, parse.IMAGE_POLICY_URL)
		title := strings.TrimSpace(articleStruct.Title.Val.(string))

		// 创建TXT文件路径 - 将所有内容保存在同目录下
		txtFilePath := filepath.Join(dirPath, title+".txt")

		// 提取纯文本内容
		result := extractTextContent(articleStruct)

		// 保存TXT文件
		err = saveTxtFile(txtFilePath, result)
		if err != nil {
			fmt.Printf("保存转换结果失败 '%s': %v\n", txtFilePath, err)
			continue
		}

		fmt.Printf("已转换: '%s' -> '%s'\n", htmlFile, txtFilePath)
		count++
	}

	return count, nil
}

// extractTextContent 从文章结构体中提取纯文本内容
func extractTextContent(article parse.Article) string {
	var textContent strings.Builder

	// 添加标题
	title := article.Title.Val.(string)
	textContent.WriteString(title)
	textContent.WriteString("\n\n")

	// 添加内容正文（仅文本）
	content := extractTextFromPieces(article.Content)
	textContent.WriteString(content)

	return textContent.String()
}

// extractTextFromPieces 从Piece列表中提取纯文本
func extractTextFromPieces(pieces []parse.Piece) string {
	var text strings.Builder

	for _, piece := range pieces {
		switch piece.Type {
		case parse.HEADER:
			// 添加标题文本
			text.WriteString(piece.Val.(string))
			text.WriteString("\n\n")
		case parse.NORMAL_TEXT, parse.BOLD_TEXT, parse.ITALIC_TEXT, parse.BOLD_ITALIC_TEXT:
			// 添加普通文本
			if str, ok := piece.Val.(string); ok {
				text.WriteString(str)
			}
		case parse.LINK:
			// 只添加链接的文本部分
			if str, ok := piece.Val.(string); ok {
				text.WriteString(str)
			}
		case parse.BLOCK_QUOTES:
			// 递归处理引用块
			if subPieces, ok := piece.Val.([]parse.Piece); ok {
				quoteText := extractTextFromPieces(subPieces)
				text.WriteString(quoteText)
				text.WriteString("\n")
			}
		case parse.O_LIST, parse.U_LIST:
			// 递归处理列表
			if subPieces, ok := piece.Val.([]parse.Piece); ok {
				listText := extractTextFromPieces(subPieces)
				text.WriteString(listText)
				text.WriteString("\n")
			}
		case parse.CODE_BLOCK:
			// 添加代码块内容
			if codeRows, ok := piece.Val.([]string); ok {
				for _, row := range codeRows {
					text.WriteString(row)
					text.WriteString("\n")
				}
				text.WriteString("\n")
			}
		case parse.BR:
			// 处理换行
			text.WriteString("\n")
		}
	}

	return text.String()
}

// saveTxtFile 保存TXT文件
func saveTxtFile(txtFilePath string, content string) error {
	return os.WriteFile(txtFilePath, []byte(content), 0o644)
}
