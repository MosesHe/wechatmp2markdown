package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fengxxc/wechatmp2markdown/format"
	"github.com/fengxxc/wechatmp2markdown/parse"
	"github.com/fengxxc/wechatmp2markdown/server"
	"github.com/fengxxc/wechatmp2markdown/util"
)

func main() {
	// test.Test1()
	// test.Test2()
	args := os.Args
	if len(args) <= 1 {
		printUsage()
		return
	}

	// 处理批量重命名目录命令
	if args[1] == "rename" {
		if len(args) <= 2 {
			fmt.Println("错误: 缺少目录路径参数")
			fmt.Println("用法: wechatmp2markdown rename [公众号目录路径]")
			return
		}

		// 处理路径中可能包含的引号
		dirPath := args[2]
		dirPath = strings.ReplaceAll(dirPath, "\"", "")

		count, err := util.BatchRenameDirectories(dirPath)
		if err != nil {
			fmt.Printf("批量重命名目录失败: %v\n", err)
			return
		}

		fmt.Printf("成功重命名 %d 个目录\n", count)
		return
	}

	// 处理批量转换命令
	if args[1] == "batch" {
		if len(args) <= 2 {
			fmt.Println("错误: 缺少目录路径参数")
			fmt.Println("用法: wechatmp2markdown batch [公众号目录路径] [--image=选项]")
			return
		}

		// 处理路径中可能包含的引号
		dirPath := args[2]
		dirPath = strings.ReplaceAll(dirPath, "\"", "")

		// 获取图片处理方式
		imageArgValue := "base64"
		if len(args) > 3 && args[3] != "" {
			if strings.HasPrefix(args[3], "--image=") {
				imageArgValue = args[3][len("--image="):]
			} else if strings.HasPrefix(args[3], "-i") {
				imageArgVal := args[3][len("-i"):]
				switch imageArgVal {
				case "u":
					imageArgValue = "url"
				case "s":
					imageArgValue = "save"
				case "b":
					fallthrough
				default:
					imageArgValue = "base64"
				}
			}
		}

		imagePolicy := parse.ImageArgValue2ImagePolicy(imageArgValue)

		count, err := util.BatchConvertHTMLFiles(dirPath, imagePolicy)
		if err != nil {
			fmt.Printf("批量转换HTML文件失败: %v\n", err)
			return
		}

		fmt.Printf("成功转换 %d 个HTML文件\n", count)
		return
	}

	// 处理批量转换HTML到TXT的命令
	if args[1] == "batchTxt" {
		if len(args) <= 2 {
			fmt.Println("错误: 缺少目录路径参数")
			fmt.Println("用法: wechatmp2markdown batchTxt [公众号目录路径]")
			return
		}

		// 处理路径中可能包含的引号
		dirPath := args[2]
		dirPath = strings.ReplaceAll(dirPath, "\"", "")

		count, err := util.BatchConvertHTMLFilesToTxt(dirPath)
		if err != nil {
			fmt.Printf("批量转换HTML文件到TXT失败: %v\n", err)
			return
		}

		fmt.Printf("成功转换 %d 个HTML文件到TXT\n", count)
		return
	}

	if len(args) <= 2 {
		fmt.Println("错误: 参数不足")
		printUsage()
		return
	}

	args1 := args[1]
	args2 := args[2]

	// 处理命令行参数中可能包含的引号
	args1 = strings.ReplaceAll(args1, "\"", "")
	args2 = strings.ReplaceAll(args2, "\"", "")

	if args1 == "server" {
		// server pattern
		port := args2
		if port == "" {
			port = "8964"
		}
		server.Start(":" + port)
		return
	}

	// 添加对file参数的支持，用于指定要解析的本地HTML文件
	isLocalFile := false
	if args1 == "file" {
		isLocalFile = true
		args1 = args2 // HTML文件路径
		if len(args) > 3 {
			args2 = args[3] // 输出路径
		} else {
			// 如果没有提供输出路径，使用当前目录
			args2 = "./"
		}
	}

	// 将单个HTML文件转换为TXT文件
	if args1 == "fileTxt" {
		if len(args) <= 2 {
			fmt.Println("错误: 缺少HTML文件路径参数")
			fmt.Println("用法: wechatmp2markdown fileTxt [HTML文件路径] [输出路径]")
			return
		}

		htmlFilePath := args[2]
		// 处理路径中可能包含的引号
		htmlFilePath = strings.ReplaceAll(htmlFilePath, "\"", "")

		// 设置输出路径，如果未提供则使用当前目录
		outputPath := "./"
		if len(args) > 3 {
			outputPath = args[3]
			outputPath = strings.ReplaceAll(outputPath, "\"", "")
		}

		txtFilePath, err := util.ConvertHTMLFileToTxt(htmlFilePath, outputPath)
		if err != nil {
			fmt.Printf("转换HTML文件到TXT失败: %v\n", err)
			return
		}

		fmt.Printf("已转换: '%s' -> '%s'\n", htmlFilePath, txtFilePath)
		return
	}

	// --image=base64 	-ib 保存图片，base64格式，在md文件中（默认为此选项）
	// --image=url 		-iu 只保留图片链接
	// --image=save 	-is 保存图片，最终输出到文件夹
	// --save=zip -sz 		最终打包输出到zip
	imageArgValue := "base64"
	imageArgIdx := 3
	if isLocalFile {
		imageArgIdx = 4 // 如果是本地文件模式，参数位置后移
	}

	if len(args) > imageArgIdx && args[imageArgIdx] != "" {
		if strings.HasPrefix(args[imageArgIdx], "--image=") {
			imageArgValue = args[imageArgIdx][len("--image="):]
		} else if strings.HasPrefix(args[imageArgIdx], "-i") {
			imageArgVal := args[imageArgIdx][len("-i"):]
			switch imageArgVal {
			case "u":
				imageArgValue = "url"
			case "s":
				imageArgValue = "save"
			case "b":
				fallthrough
			default:
				imageArgValue = "base64"
			}
		}
	}

	var imagePolicy parse.ImagePolicy = parse.ImageArgValue2ImagePolicy(imageArgValue)

	var articleStruct parse.Article

	if isLocalFile {
		// 从本地HTML文件解析
		htmlFilePath := args1
		fmt.Printf("HTML file: %s, output: %s\n", htmlFilePath, args2)
		articleStruct = parse.ParseFromHTMLFile(htmlFilePath, imagePolicy)
	} else {
		// cli pattern - 从URL解析
		url := args1
		filename := args2
		fmt.Printf("url: %s, filename: %s\n", url, filename)
		articleStruct = parse.ParseFromURL(url, imagePolicy)
	}

	format.FormatAndSave(articleStruct, args2)
}

// 打印使用说明
func printUsage() {
	fmt.Println("wechatmp2markdown - 微信公众号文章转Markdown工具")
	fmt.Println("\n用法:")
	fmt.Println("  1. 从URL转换:")
	fmt.Println("     wechatmp2markdown [url] [输出路径] [--image=选项]")
	fmt.Println("     例如: wechatmp2markdown https://mp.weixin.qq.com/s/xxx ./output --image=save")
	fmt.Println("\n  2. 从本地HTML文件转换:")
	fmt.Println("     wechatmp2markdown file [HTML文件路径] [输出路径] [--image=选项]")
	fmt.Println("     例如: wechatmp2markdown file ./article.html ./output --image=save")
	fmt.Println("\n  3. 启动Web服务:")
	fmt.Println("     wechatmp2markdown server [端口号]")
	fmt.Println("     例如: wechatmp2markdown server 8964")
	fmt.Println("\n  4. 批量重命名目录:")
	fmt.Println("     wechatmp2markdown rename [公众号目录路径]")
	fmt.Println("     例如: wechatmp2markdown rename D:\\WechatDownload\\浙江宣传")
	fmt.Println("\n  5. 批量转换HTML文件:")
	fmt.Println("     wechatmp2markdown batch [公众号目录路径] [--image=选项]")
	fmt.Println("     例如: wechatmp2markdown batch D:\\WechatDownload\\浙江宣传 --image=save")
	fmt.Println("\n  6. 批量转换HTML文件为TXT:")
	fmt.Println("     wechatmp2markdown batchTxt [公众号目录路径]")
	fmt.Println("     例如: wechatmp2markdown batchTxt D:\\WechatDownload\\浙江宣传")
	fmt.Println("\n  7. 从本地HTML文件转换为TXT:")
	fmt.Println("     wechatmp2markdown fileTxt [HTML文件路径] [输出路径]")
	fmt.Println("     例如: wechatmp2markdown fileTxt ./article.html ./output")
	fmt.Println("\n图片选项:")
	fmt.Println("  --image=url    只保留图片URL链接")
	fmt.Println("  --image=save   保存图片到本地")
	fmt.Println("  --image=base64 将图片转换为base64编码嵌入Markdown (默认)")
}
