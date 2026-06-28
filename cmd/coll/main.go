package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

var codeExtensions = map[string]struct{}{
	".py": {}, ".js": {}, ".ts": {}, ".jsx": {}, ".tsx": {},
	".html": {}, ".css": {}, ".scss": {}, ".sass": {},
	".java": {}, ".cpp": {}, ".c": {}, ".h": {}, ".hpp": {},
	".cs": {}, ".go": {}, ".rs": {}, ".rb": {}, ".php": {},
	".sh": {}, ".bash": {}, ".sql": {}, ".json": {},
	".yaml": {}, ".yml": {}, ".xml": {}, ".md": {}, ".txt": {},
	".vue": {}, ".svelte": {}, ".swift": {}, ".kt": {}, ".kts": {},
	".pl": {}, ".pm": {}, ".r": {}, ".R": {}, ".dart": {}, ".lua": {},
	".scala": {}, ".groovy": {}, ".clj": {}, ".cljs": {}, ".edn": {},
	".toml": {}, ".ini": {}, ".cfg": {}, ".conf": {}, ".gradle": {},
	".dockerfile": {}, ".env": {}, ".lock": {}, ".puml": {}, ".pu": {}, ".dbml": {},
}

var ignoreNamesLower = map[string]struct{}{
	"node_modules": {}, "__pycache__": {}, ".git": {}, ".svn": {}, ".hg": {},
	"venv": {}, "env": {}, ".venv": {}, "dist": {}, "build": {}, "out": {},
	"coverage": {}, ".next": {}, ".nuxt": {}, ".cache": {}, ".idea": {},
	".vscode": {}, ".vs": {}, ".pytest_cache": {}, ".mypy_cache": {}, ".tox": {},
	".eggs": {}, "tmp": {}, "temp": {}, "vendor": {}, ".env": {},
}

var specialFiles = map[string]struct{}{
	".gitignore": {}, "package-lock.json": {}, "yarn.lock": {},
}

type treeNode struct {
	name     string
	isDir    bool
	children []*treeNode
}

func main() {
	outputFile := flag.String("output", "", "Name of the output file")
	allFiles := flag.Bool("all", false, "Collect all files regardless of extension")

	// Changed from bool to int.
	// -1 means disabled (default).
	// 0 means unlimited depth (recursive).
	// >0 limits the tree to that specific depth.
	treeDepth := flag.Int("t", -1, "Recursive tree view with max depth (0 for unlimited)")
	flag.IntVar(treeDepth, "tree", -1, "Recursive tree view with max depth (0 for unlimited)")

	onlyExts := flag.String("o", "", "Only include files with these extensions (comma-separated, e.g., \"md,py,txt\")")
	flag.StringVar(onlyExts, "only", "", "Only include files with these extensions (comma-separated)")

	exceptExts := flag.String("e", "", "Exclude files with these extensions (comma-separated, e.g., \"md,py,txt\")")
	flag.StringVar(exceptExts, "except", "", "Exclude files with these extensions (comma-separated)")

	unwrapFile := flag.String("u", "", "Unwrap a collected .txt file back into a directory")
	flag.StringVar(unwrapFile, "unwrap", "", "Unwrap a collected .txt file back into a directory")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: coll [path] [--output output.txt] [--all] [-t depth] [-o extensions] [-e extensions] [-u file.txt]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Handle unwrap mode first
	if *unwrapFile != "" {
		if err := unwrap(*unwrapFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error during unwrap: %v\n", err)
			os.Exit(1)
		}
		return
	}

	rootPath := "."
	if flag.NArg() > 0 {
		rootPath = flag.Arg(0)
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with an absolute path: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: '%s' is not a directory.\n", rootPath)
		os.Exit(1)
	}

	// If treeDepth is >= 0, tree mode is enabled
	if *treeDepth >= 0 {
		tree := buildTree(rootAbs, 0, *treeDepth)
		printTree(tree)
		return
	}

	onlySet := parseExtensions(*onlyExts)
	exceptSet := parseExtensions(*exceptExts)

	for ext := range onlySet {
		if _, exists := exceptSet[ext]; exists {
			fmt.Fprintf(os.Stderr, "Error: Extension '%s' appears in both -o (only) and -e (except). Please remove the conflict.\n", ext)
			os.Exit(1)
		}
	}

	type fileEntry struct {
		RelPath string
		Content string
	}

	var entries []fileEntry

	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			lower := strings.ToLower(name)
			if _, ignored := ignoreNamesLower[lower]; ignored {
				return filepath.SkipDir
			}
			if strings.HasPrefix(name, ".") && name != "." && name != ".." {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(rootAbs, path)
		relPath = filepath.ToSlash(relPath)

		if pathContainsIgnored(relPath) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		filename := d.Name()
		_, isCode := codeExtensions[ext]
		_, isSpecial := specialFiles[strings.ToLower(filename)]

		include := false

		if len(onlySet) > 0 {
			if _, ok := onlySet[ext]; ok {
				include = true
			}
		} else if len(exceptSet) > 0 {
			if _, ok := exceptSet[ext]; !ok {
				include = *allFiles || isCode || isSpecial
			}
		} else {
			include = *allFiles || isCode || isSpecial
		}

		if include {
			entries = append(entries, fileEntry{RelPath: relPath, Content: readFileContent(path)})
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during the work with directory: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "Didn't found any comparable files.")
		os.Exit(1)
	}

	outName := *outputFile
	if outName == "" {
		outName = filepath.Base(rootAbs) + ".txt"
	}

	f, err := os.Create(outName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create a file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	for _, e := range entries {
		fmt.Fprintf(f, "./%s:\n```\n%s\n```\n", e.RelPath, e.Content)
	}

	outAbs, _ := filepath.Abs(outName)
	fmt.Printf("Done! The result: %s\n", outAbs)
}

// unwrap parses a collected .txt file and reconstructs the directory structure
func unwrap(inputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	// Determine output directory name: same as input file without extension
	baseName := filepath.Base(inputPath)
	ext := filepath.Ext(baseName)
	dirName := strings.TrimSuffix(baseName, ext)
	if dirName == "" {
		return fmt.Errorf("invalid input filename")
	}

	// Create the output directory
	if err := os.MkdirAll(dirName, 0755); err != nil {
		return fmt.Errorf("cannot create directory '%s': %w", dirName, err)
	}

	lines := strings.Split(string(data), "\n")
	fileCount := 0
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Look for file marker: "./path/to/file:"
		if !strings.HasPrefix(line, "./") || !strings.HasSuffix(line, ":") {
			i++
			continue
		}

		relPath := strings.TrimSuffix(strings.TrimPrefix(line, "./"), ":")
		relPath = filepath.FromSlash(relPath)

		// Next line should be opening ```
		i++
		if i >= len(lines) || strings.TrimSpace(lines[i]) != "```" {
			continue
		}
		i++ // skip opening ```

		// Read content until closing ```
		// The closing ``` must be followed by either EOF or another file marker
		var content strings.Builder
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == "```" {
				// Check if this is the real closing ```
				if i+1 >= len(lines) || (strings.HasPrefix(lines[i+1], "./") && strings.HasSuffix(lines[i+1], ":")) {
					break
				}
			}
			content.WriteString(lines[i])
			content.WriteString("\n")
			i++
		}

		// Remove trailing newline if present
		result := content.String()
		if strings.HasSuffix(result, "\n") {
			result = result[:len(result)-1]
		}

		// Skip closing ```
		if i < len(lines) && strings.TrimSpace(lines[i]) == "```" {
			i++
		}

		// Create the file
		fullPath := filepath.Join(dirName, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("cannot create directory for '%s': %w", relPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(result), 0644); err != nil {
			return fmt.Errorf("cannot write file '%s': %w", fullPath, err)
		}

		fileCount++
	}

	if fileCount == 0 {
		return fmt.Errorf("no files found in '%s'", inputPath)
	}

	absDir, _ := filepath.Abs(dirName)
	fmt.Printf("Done! Unwrapped %d file(s) into: %s\n", fileCount, absDir)
	return nil
}

func parseExtensions(extStr string) map[string]struct{} {
	result := make(map[string]struct{})
	if extStr == "" {
		return result
	}

	parts := strings.Split(extStr, ",")
	for _, part := range parts {
		ext := strings.TrimSpace(part)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ext = strings.ToLower(ext)
		result[ext] = struct{}{}
	}

	return result
}

// buildTree recursively builds the tree structure up to maxDepth
func buildTree(dirPath string, currentDepth, maxDepth int) *treeNode {
	name := filepath.Base(dirPath)
	node := &treeNode{name: name, isDir: true}

	// Stop traversing deeper if we've reached the specified max depth
	// (maxDepth == 0 means unlimited depth)
	if maxDepth > 0 && currentDepth >= maxDepth {
		return node
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return node
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, entry := range entries {
		entryName := entry.Name()
		lower := strings.ToLower(entryName)

		if entry.IsDir() {
			if _, ignored := ignoreNamesLower[lower]; ignored {
				continue
			}
			if strings.HasPrefix(entryName, ".") {
				continue
			}
			childPath := filepath.Join(dirPath, entryName)
			// Increment depth for children
			child := buildTree(childPath, currentDepth+1, maxDepth)
			node.children = append(node.children, child)
		} else {
			node.children = append(node.children, &treeNode{
				name:  entryName,
				isDir: false,
			})
		}
	}

	return node
}

func printTree(root *treeNode) {
	fmt.Println(".")
	printTreeNode(root, "", true, true)
}

func printTreeNode(node *treeNode, prefix string, isLast bool, isRoot bool) {
	if !isRoot {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		displayName := node.name
		fmt.Printf("%s%s%s\n", prefix, connector, displayName)
	}

	childPrefix := prefix
	if !isRoot {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.children {
		isLastChild := i == len(node.children)-1
		printTreeNode(child, childPrefix, isLastChild, false)
	}
}

func pathContainsIgnored(path string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		lower := strings.ToLower(part)
		if _, ok := ignoreNamesLower[lower]; ok {
			return true
		}
		if strings.HasPrefix(part, ".") && part != parts[len(parts)-1] {
			return true
		}
	}
	return false
}

func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("[Error reading file: %v]", err)
	}
	if utf8.Valid(data) {
		return string(data)
	}
	var b strings.Builder
	b.Grow(len(data))
	for _, c := range data {
		b.WriteRune(rune(c))
	}
	return b.String()
}
