# 文件结构树生成器

一个使用 Go + Fyne 开发的桌面应用程序，用于扫描目录并生成可视化的文件结构树，支持导出为 HTML 文件。

## 功能特性

- 📁 **目录扫描** - 选择任意目录进行扫描，计算每个文件/子目录的大小
- 🌲 **文件树展示** - 以树形结构展示文件和目录，按大小排序
- 📊 **空间占用分析** - 显示每个文件/目录占总大小的百分比
- 🎨 **HTML 导出** - 将文件树导出为交互式 HTML 页面，支持折叠/展开目录
- 👁️ **内置预览** - 在应用内预览文件树结构
- 🔴 **大文件标记** - 超过 10% 占比的文件标记为红色，5-10% 标记为黄色

## 界面预览

应用程序主要界面包含：
- 源目录选择
- 输出文件路径设置
- 生成、预览、保存按钮
- 进度条和状态提示

生成的 HTML 页面特点：
- 可折叠/展开的目录树
- 按大小排序的子项
- 大文件高亮显示
- 总大小统计

## 技术栈

| 组件 | 技术 |
|------|------|
| GUI 框架 | [Fyne v2](https://fyne.io/) |
| 模板引擎 | Go 标准库 `html/template` |
| 文件处理 | Go 标准库 `os`, `path/filepath` |

## 系统要求

- Go 1.16 或更高版本
- 支持 Fyne 的操作系统：Windows、macOS、Linux

## 安装与运行

### 1. 克隆代码

```bash
git clone <your-repo-url>
cd file-tree-generator
```

### 2. 安装依赖

```bash
go mod init file-tree-generator
go get fyne.io/fyne/v2
```

### 3. 运行程序

```bash
go run main.go
```

### 4. 编译可执行文件

```bash
# Windows
go build -o file-tree-generator.exe main.go

# macOS/Linux
go build -o file-tree-generator main.go
```

## 使用说明

### 基本流程

1. **选择目录** - 点击「选择目录」按钮，选择要扫描的文件夹
2. **设置输出路径**（可选）- 点击「保存位置」指定 HTML 输出文件路径
   - 如果未指定，默认保存在源目录下的 `file_tree.html`
3. **生成文件树** - 点击「生成文件树」按钮开始扫描
4. **预览** - 生成完成后点击「预览」查看树形结构
5. **保存** - 确认后保存 HTML 文件

### 交互式 HTML 使用

生成的 HTML 文件特点：
- 点击目录旁的按钮可折叠/展开子目录
- 占比超过 10% 的目录/文件会以红色加粗显示
- 支持所有现代浏览器打开

## 代码结构

```
├── main.go          # 主程序，包含所有逻辑
│   ├── FileNode     # 文件树节点结构
│   ├── buildTree()  # 构建文件树
│   ├── generateHTML() # 生成HTML
│   ├── GUI          # GUI应用结构
│   └── main()       # 程序入口
└── README.md        # 本文件
```

### 核心数据结构

```go
type FileNode struct {
    Path     string      // 完整路径
    Name     string      // 文件/目录名
    Size     uint64      // 大小（字节）
    IsDir    bool        // 是否为目录
    Children []*FileNode // 子节点列表
    Percent  float64     // 占总大小的百分比
}
```

## 注意事项

- 扫描大型目录可能耗时较长，请耐心等待
- 程序会递归扫描所有子目录，确保有足够的读取权限
- 生成的 HTML 文件为独立文件，无需外部依赖即可在浏览器中查看

## 开源协议

MIT License

## 待改进功能

- [ ] 添加取消扫描功能
- [ ] 支持更多导出格式（JSON、TXT）
- [ ] 添加文件类型过滤
- [ ] 支持拖拽选择目录
- [ ] 增加扫描进度实时显示