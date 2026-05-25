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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FileNode 表示文件或目录节点
type FileNode struct {
	Path     string
	Name     string
	Size     uint64
	IsDir    bool
	Children []*FileNode
	Percent  float64
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

// hashPath 生成路径的简短唯一ID
func hashPath(path string) string {
	h := sha256.Sum256([]byte(path))
	return "toggle_" + hex.EncodeToString(h[:])[:8]
}

// buildTree 构建文件树
func buildTree(root string) (*FileNode, error) {
	sizeCache := make(map[string]uint64)
	totalSize := uint64(0)

	// 第一次遍历：计算大小
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if info.IsDir() {
			dirSize := uint64(0)
			filepath.WalkDir(path, func(subPath string, subD fs.DirEntry, subErr error) error {
				if subErr != nil {
					return nil
				}
				if !subD.IsDir() {
					if f, err := subD.Info(); err == nil {
						dirSize += uint64(f.Size())
					}
				}
				return nil
			})
			sizeCache[path] = dirSize
		} else {
			sizeCache[path] = uint64(info.Size())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	totalSize = sizeCache[root]

	// 第二次遍历：构建树结构
	var build func(string) *FileNode
	build = func(path string) *FileNode {
		info, _ := os.Stat(path)
		name := filepath.Base(path)
		if name == "" {
			name = path
		}
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
				return node
			}

			children := make([]*FileNode, 0, len(entries))
			for _, entry := range entries {
				childPath := filepath.Join(path, entry.Name())
				child := build(childPath)
				children = append(children, child)
			}

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
            max-width: 1200px;
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

func generateHTML(rootNode *FileNode) (string, error) {
	data := PageData{
		RootName:     rootNode.Name,
		Nodes:        []*FileNode{rootNode},
		TotalSize:    rootNode.Size,
		TotalSizeStr: formatSize(rootNode.Size),
		CurrentTime:  time.Now().Format("2006-01-02 15:04:05"),
	}

	var htmlBuilder strings.Builder
	err := tpl.Execute(&htmlBuilder, data)
	if err != nil {
		return "", err
	}
	return htmlBuilder.String(), nil
}

// GUI 应用
type GUI struct {
	window      fyne.Window
	app         fyne.App
	dirEntry    *widget.Entry
	outputEntry *widget.Entry
	progressBar *widget.ProgressBar
	statusLabel *widget.Label
	rootNode    *FileNode
	htmlContent string
	selectBtn   *widget.Button
	generateBtn *widget.Button
	saveBtn     *widget.Button
	previewBtn  *widget.Button
}

func main() {
	gui := &GUI{}
	gui.app = app.NewWithID("com.filetree.viewer")
	gui.app.SetIcon(theme.FolderIcon())

	gui.window = gui.app.NewWindow("文件结构树生成器")
	gui.window.Resize(fyne.NewSize(600, 400))

	gui.setupUI()
	gui.window.ShowAndRun()
}

func (g *GUI) setupUI() {
	// 标题
	titleLabel := widget.NewLabelWithStyle("📁 文件结构树生成器",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true})

	// 目录选择
	g.dirEntry = widget.NewEntry()
	g.dirEntry.SetPlaceHolder("选择要扫描的目录...")

	g.selectBtn = widget.NewButtonWithIcon("选择目录", theme.FolderOpenIcon(), func() {
		g.selectDirectory()
	})

	dirBox := container.NewBorder(nil, nil, nil, g.selectBtn, g.dirEntry)

	// 输出文件路径
	g.outputEntry = widget.NewEntry()
	g.outputEntry.SetPlaceHolder("输出HTML文件路径...")

	g.saveBtn = widget.NewButtonWithIcon("保存位置", theme.DocumentSaveIcon(), func() {
		g.selectOutputFile()
	})

	outputBox := container.NewBorder(nil, nil, nil, g.saveBtn, g.outputEntry)

	// 控制按钮
	g.generateBtn = widget.NewButtonWithIcon("生成文件树", theme.MediaPlayIcon(), func() {
		g.generateTree()
	})

	g.previewBtn = widget.NewButtonWithIcon("预览", theme.SearchIcon(), func() {
		g.showPreview()
	})
	g.previewBtn.Disable()

	// 进度条和状态
	g.progressBar = widget.NewProgressBar()
	g.progressBar.Hide()

	g.statusLabel = widget.NewLabel("就绪")

	// 布局
	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		widget.NewLabel("源目录:"),
		dirBox,
		widget.NewLabel("输出文件:"),
		outputBox,
		container.NewHBox(
			g.generateBtn,
			g.previewBtn,
			layout.NewSpacer(),
		),
		g.progressBar,
		g.statusLabel,
	)

	g.window.SetContent(content)
}

func (g *GUI) selectDirectory() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		if uri == nil {
			return
		}
		g.dirEntry.SetText(uri.Path())
		// 自动设置输出文件路径
		if g.outputEntry.Text == "" {
			defaultOutput := filepath.Join(uri.Path(), "file_tree.html")
			g.outputEntry.SetText(defaultOutput)
		}
	}, g.window)
}

func (g *GUI) selectOutputFile() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		if writer == nil {
			return
		}
		g.outputEntry.SetText(writer.URI().Path())
		writer.Close()
	}, g.window)
}

func (g *GUI) generateTree() {
	rootPath := g.dirEntry.Text
	if rootPath == "" {
		dialog.ShowInformation("提示", "请先选择要扫描的目录", g.window)
		return
	}

	// 禁用按钮
	g.generateBtn.Disable()
	g.selectBtn.Disable()

	// 显示进度条
	g.progressBar.Show()
	g.progressBar.SetValue(0)
	g.statusLabel.SetText("正在扫描目录...")

	// 在后台线程中处理
	go func() {
		rootNode, err := buildTree(rootPath)

		// 使用 fyne.Do 在主线程更新 UI
		fyne.Do(func() {
			defer func() {
				g.generateBtn.Enable()
				g.selectBtn.Enable()
				g.progressBar.Hide()
			}()

			if err != nil {
				dialog.ShowError(fmt.Errorf("扫描失败: %v", err), g.window)
				g.statusLabel.SetText("扫描失败")
				return
			}

			g.rootNode = rootNode
			g.progressBar.SetValue(0.5)
			g.statusLabel.SetText("正在生成HTML...")

			// 生成 HTML
			htmlContent, err := generateHTML(rootNode)
			if err != nil {
				dialog.ShowError(fmt.Errorf("生成HTML失败: %v", err), g.window)
				g.statusLabel.SetText("生成失败")
				return
			}

			g.htmlContent = htmlContent
			g.progressBar.SetValue(1)
			g.statusLabel.SetText(fmt.Sprintf("✅ 完成 - 总大小: %s", formatSize(rootNode.Size)))

			// 启用预览按钮
			g.previewBtn.Enable()

			// 询问是否保存
			if g.outputEntry.Text != "" {
				dialog.ShowConfirm("保存文件",
					fmt.Sprintf("文件树已生成！\n总大小: %s\n\n是否保存HTML文件到:\n%s",
						formatSize(rootNode.Size), g.outputEntry.Text),
					func(save bool) {
						if save {
							g.saveHTML()
						}
					}, g.window)
			}
		})
	}()
}

func (g *GUI) saveHTML() {
	outputPath := g.outputEntry.Text
	if outputPath == "" {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, g.window)
				return
			}
			if writer == nil {
				return
			}
			outputPath = writer.URI().Path()
			writer.Close()
			g.outputEntry.SetText(outputPath)
			g.doSave(outputPath)
		}, g.window)
	} else {
		g.doSave(outputPath)
	}
}

func (g *GUI) doSave(outputPath string) {
	if g.htmlContent == "" {
		dialog.ShowError(fmt.Errorf("没有可保存的内容"), g.window)
		return
	}

	err := os.WriteFile(outputPath, []byte(g.htmlContent), 0644)
	if err != nil {
		dialog.ShowError(fmt.Errorf("保存失败: %v", err), g.window)
		return
	}

	dialog.ShowInformation("成功",
		fmt.Sprintf("HTML文件已保存到:\n%s\n\n总大小: %s",
			outputPath, formatSize(g.rootNode.Size)), g.window)
}

func (g *GUI) showPreview() {
	if g.rootNode == nil {
		dialog.ShowInformation("提示", "请先生成文件树", g.window)
		return
	}

	// 创建预览窗口
	previewWin := g.app.NewWindow("文件树预览 - " + g.rootNode.Name)
	previewWin.Resize(fyne.NewSize(700, 600))

	// 生成树形文本
	treeText := g.buildTreeText(g.rootNode, 0)

	previewContent := widget.NewLabel(treeText)
	previewContent.Wrapping = fyne.TextWrapOff

	scroll := container.NewScroll(previewContent)

	// 添加保存按钮
	saveBtn := widget.NewButtonWithIcon("保存HTML", theme.DocumentSaveIcon(), func() {
		g.saveHTML()
		previewWin.Close()
	})

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle(fmt.Sprintf("📁 %s", g.rootNode.Name),
				fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("💾 总大小: %s", formatSize(g.rootNode.Size))),
			widget.NewLabel(fmt.Sprintf("📊 文件/目录总数: %d", countNodes(g.rootNode))),
			widget.NewSeparator(),
		),
		container.NewHBox(
			layout.NewSpacer(),
			saveBtn,
		),
		nil, nil,
		scroll,
	)

	previewWin.SetContent(content)
	previewWin.Show()
}

func (g *GUI) buildTreeText(node *FileNode, level int) string {
	indent := strings.Repeat("  ", level)
	var result strings.Builder

	if node.IsDir {
		if len(node.Children) == 0 {
			result.WriteString(fmt.Sprintf("%s📁 %s/ (%s, %.1f%%) [空目录]\n",
				indent, node.Name, formatSize(node.Size), node.Percent))
		} else {
			result.WriteString(fmt.Sprintf("%s📁 %s/ (%s, %.1f%%)\n",
				indent, node.Name, formatSize(node.Size), node.Percent))
			for _, child := range node.Children {
				result.WriteString(g.buildTreeText(child, level+1))
			}
		}
	} else {
		// 大文件用红色标记
		marker := ""
		if node.Percent > 10 {
			marker = " 🔴"
		} else if node.Percent > 5 {
			marker = " 🟡"
		}
		result.WriteString(fmt.Sprintf("%s📄 %s (%s, %.1f%%)%s\n",
			indent, node.Name, formatSize(node.Size), node.Percent, marker))
	}

	return result.String()
}

func countNodes(node *FileNode) int {
	count := 1
	if node.IsDir {
		for _, child := range node.Children {
			count += countNodes(child)
		}
	}
	return count
}
