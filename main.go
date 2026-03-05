package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileNode 表示文件或目录节点
type FileNode struct {
	Path     string
	Name     string
	Size     uint64
	IsDir    bool
	Children []*FileNode
	Percent  float64 // 占总大小的百分比
}

// formatSize 格式化字节数为 B/KB/MB/GB
func formatSize(bytes uint64) string {
	size := float64(bytes)
	units := []string{"B", "KB", "MB", "GB"}
	unit := 0
	for size >= 1024 && unit < 3 {
		size /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f %s", size, units[unit])
}

// hashPath 生成路径的简短唯一ID（用于HTML折叠）
func hashPath(path string) string {
	h := sha256.Sum256([]byte(path))
	return "toggle_" + hex.EncodeToString(h[:])[:8]
}

// buildTree 构建带大小缓存的树结构
func buildTree(root string) (*FileNode, error) {
	sizeCache := make(map[string]uint64)

	var walk func(string, fs.FileInfo) error
	walk = func(path string, info fs.FileInfo) error {
		if info.IsDir() {
			var total uint64
			err := filepath.WalkDir(path, func(subPath string, d fs.DirEntry, err error) error {
				if err != nil {
					fmt.Fprintf(os.Stderr, "无法访问: %s (%v)\n", subPath, err)
					return nil // 跳过错误
				}
				if d.IsDir() {
					return nil // 不递归进子目录（由外层控制）
				}
				if f, err := d.Info(); err == nil {
					total += uint64(f.Size())
				}
				return nil
			})
			if err != nil {
				return err
			}
			sizeCache[path] = total
		} else {
			sizeCache[path] = uint64(info.Size())
		}
		return nil
	}

	// 第一次遍历：填充 sizeCache
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "跳过路径: %s (%v)\n", path, err)
			return nil
		}
		info, _ := d.Info()
		return walk(path, info)
	})
	if err != nil {
		return nil, err
	}

	totalSize := sizeCache[root]

	// 第二次构建树结构（带排序）
	var build func(string) *FileNode
	build = func(path string) *FileNode {
		info, _ := os.Stat(path)
		name := filepath.Base(path)
		size := sizeCache[path]
		isDir := info.IsDir()
		node := &FileNode{
			Path:    path,
			Name:    name,
			Size:    size,
			IsDir:   isDir,
			Percent: 0,
		}
		if totalSize > 0 {
			node.Percent = float64(size) / float64(totalSize) * 100
		}

		if isDir {
			entries, err := os.ReadDir(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️ 无法读取目录: %s (%v)\n", path, err)
				return node
			}

			children := make([]*FileNode, 0, len(entries))
			for _, entry := range entries {
				childPath := filepath.Join(path, entry.Name())
				child := build(childPath)
				children = append(children, child)
			}

			// 排序：先按大小降序，再按名称升序
			sort.Slice(children, func(i, j int) bool {
				if children[i].Size != children[j].Size {
					return children[i].Size > children[j].Size
				}
				return children[i].Name < children[j].Name
			})

			node.Children = children
		}
		return node
	}

	rootNode := build(root)
	return rootNode, nil
}

// escapeJS 转义字符串用于 JavaScript（简单版）
func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// toHTMLString 手动转义 HTML（因为 template 会自动转义，但我们需要部分不转义）
func toHTMLString(s string) template.HTML {
	return template.HTML(html.EscapeString(s))
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <title>文件结构树 - {{.RootName}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f8f9fa;
            color: #333;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            background: white;
            border-radius: 10px;
            padding: 25px;
            box-shadow: 0 4px 15px rgba(0,0,0,0.1);
            max-width: 1000px;
            margin: auto;
        }
        h1 {
            color: #2c3e50;
            border-bottom: 2px solid #ecf0f1;
            padding-bottom: 10px;
            margin-bottom: 20px;
            font-size: 1.8em;
        }
        ul.tree {
            list-style-type: none;
            padding-left: 20px;
        }
        ul.folder {
            margin-top: 4px;
            margin-bottom: 6px;
        }
        .tree li {
            margin: 2px 0;
        }
        .tree-dir > button {
            background: none;
            border: none;
            font-size: inherit;
            cursor: pointer;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .tree-dir > button:hover {
            color: #2980b9;
        }
        .tree-file {
            color: #555;
            font-size: 0.95em;
        }
        .file-size, .dir-size {
            color: #7f8c8d;
            font-size: 0.85em;
            margin-left: 8px;
        }
        .empty {
            color: #95a5a6;
            font-style: italic;
            font-size: 0.85em;
        }
        .footer {
            margin-top: 30px;
            color: #777;
            font-size: 0.9em;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>📁 {{.RootName}}</h1>
        <ul class="tree">
            {{range .Nodes}}
                {{template "node" .}}
            {{end}}
        </ul>

        <div style="margin-top: 20px; font-weight: bold; color: #27ae60;">
            💾 总大小: {{.TotalSizeStr}}
        </div>

        <div class="footer">
            生成时间: {{.CurrentTime}}
        </div>
    </div>

    <script>
        function toggleFolder(id) {
            let folder = document.getElementById(id);
            folder.style.display = folder.style.display === 'none' ? 'block' : 'none';
        }
    </script>
</body>
</html>

{{define "node"}}
{{if .IsDir}}
<li class="tree-dir">
    {{$id := .Path | hashPath}}
    {{$percent := printf "%.0f" .Percent}}
    {{$style := ""}}
    {{if gt .Percent 10.0}}{{$style = "color:#e74c3c;font-weight:bold;"}}{{end}}
    <button class="toggle-btn" style="{{$style}}" onclick="toggleFolder('{{$id}}')">
        📁 {{.Name | html}}
        <span class="dir-size">({{formatSize .Size}}, {{$percent}}%)</span>
    </button>
    <ul class="folder" id="{{$id}}" style="display:none;">
        {{if eq (len .Children) 0}}
            <li class="empty"><em>空目录</em></li>
        {{else}}
            {{range .Children}}
                {{template "node" .}}
            {{end}}
        {{end}}
    </ul>
</li>
{{else}}
    {{$percent := printf "%.0f" .Percent}}
    {{$style := ""}}
    {{if gt .Percent 10.0}}{{$style = "color:#e74c3c;font-weight:bold;"}}{{end}}
    <li class="tree-file">
        📄 <span style="font-family:monospace;{{$style}}">{{.Name | html}}</span>
        <span class="file-size">({{formatSize .Size}}, {{$percent}}%)</span>
    </li>
{{end}}
{{end}}
`

var tpl *template.Template

func init() {
	tpl = template.Must(template.New("tree").
		Funcs(template.FuncMap{
			"formatSize": formatSize,
			"hashPath":   hashPath,
			"html":       toHTMLString,
		}).
		Parse(htmlTemplate))
}

type PageData struct {
	RootName     string
	Nodes        []*FileNode
	TotalSize    uint64
	TotalSizeStr string
	CurrentTime  string
}

func generateHTML(rootNode *FileNode, outputPath string) error {
	data := PageData{
		RootName:     rootNode.Name,
		Nodes:        []*FileNode{rootNode},
		TotalSize:    rootNode.Size,
		TotalSizeStr: formatSize(rootNode.Size),
		CurrentTime:  time.Now().Format("2006-01-02 15:04:05"),
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("无法创建输出文件: %w", err)
	}
	defer file.Close()

	return tpl.Execute(file, data)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "用法: %s <根目录路径> <输出HTML文件>\n", os.Args[0])
		os.Exit(1)
	}

	rootPath := os.Args[1]
	outputHTML := os.Args[2]

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "路径不存在: %s\n", rootPath)
		os.Exit(1)
	}

	fmt.Printf("正在扫描目录: %s\n", rootPath)
	rootNode, err := buildTree(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
		os.Exit(1)
	}

	err = generateHTML(rootNode, outputHTML)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 HTML 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nHTML 文件已生成: %s\n", outputHTML)
	fmt.Printf("扫描路径: %s\n", rootPath)
	fmt.Printf("总大小: %s\n", formatSize(rootNode.Size))
}
