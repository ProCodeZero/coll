# Project Collector
A lightweight command-line tool written in Go that aggregates source code files from a directory into a single text file. This is useful for providing context to Large Language Models (LLMs), creating quick backups of codebases, or inspecting project structure.

## Features
- Recursively walks through directories to collect files.
- Filters out common build artifacts, dependencies, and version control folders (e.g., `node_modules`, `.git`, `venv`).
- Supports a wide range of programming languages and configuration formats.
- Optionally collects all files regardless of extension.
- Outputs a clean, formatted text file with relative paths and content blocks.
- **Tree view**: Print a visual directory tree with `-wr`.
- **Extension filtering**: Include only specific extensions with `-o`, or exclude them with `-e`.
- **Unwrap**: Reconstruct a directory from a previously collected `.txt` file with `-u`.

## Installation

### Using Go Install
You can install the tool directly using the `go install` command:
```bash
go install github.com/ProCodeZero/project-collector/cmd/coll@latest
```

### Building from Source
Ensure you have Go installed. Clone the repository and build the binary:
```bash
git clone https://github.com/ProCodeZero/project-collector.git
cd project-collector
go build -o coll ./cmd/coll/main.go
```

## Usage

### Basic Usage
Run the tool in the root of your project to collect all recognized source files:
```bash
coll
```
This will generate a file named `<directory_name>.txt` in the current directory.
If you built from source, use `./coll` instead of `coll`.

### Specify Output File
Use the `--output` flag to define a custom output filename:
```bash
coll --output context.txt
```

### Collect All Files
Use the `--all` flag to include every file, ignoring extension filters (still respects ignore lists like `.git`):
```bash
coll --all
```

### Specify Target Directory
Provide a path as an argument to scan a specific directory:
```bash
coll ./src --output src_context.txt
```

### View Directory Tree
Use `-w` (`--watch`) and `-r` (`--recursive`) together to print a visual tree of the project structure to the terminal:
```bash
coll -wr
```
Example output:
```
.
├── cmd
│   └── coll
│       └── main.go
├── README.md
├── go.mod
└── go.sum
```

### Filter by Extension (Only)
Use `-o` (`--only`) to collect **only** files with the specified extensions (comma-separated):
```bash
coll -o "md,py,txt"
```
This will include only `.md`, `.py`, and `.txt` files in the result.

### Filter by Extension (Except)
Use `-e` (`--except`) to collect all recognized files **except** those with the specified extensions:
```bash
coll -e "md,py,txt"
```
This will include all recognized files except `.md`, `.py`, and `.txt`.

> **Note:** Using the same extension in both `-o` and `-e` will result in an error.

### Combine Filters
You can combine `-o` and `-e` to narrow down results further (as long as they don't overlap):
```bash
coll -o "js,ts,jsx,tsx" -e "test.js,spec.ts"
```

### Unwrap a Collected File
Use `-u` (`--unwrap`) to reconstruct a directory from a previously collected `.txt` file. A new directory with the same name as the file (without the `.txt` extension) will be created:
```bash
coll -u project.txt
```
This creates a `project/` directory containing all the files stored in `project.txt`.

## Supported Extensions
The tool automatically includes files with the following extensions:
- **Web**: `.html`, `.css`, `.scss`, `.sass`, `.js`, `.ts`, `.jsx`, `.tsx`, `.vue`, `.svelte`
- **Backend**: `.py`, `.java`, `.cpp`, `.c`, `.h`, `.hpp`, `.cs`, `.go`, `.rs`, `.rb`, `.php`, `.sh`, `.bash`, `.sql`, `.swift`, `.kt`, `.kts`, `.dart`, `.lua`, `.scala`, `.groovy`, `.clj`, `.cljs`, `.edn`, `.pl`, `.pm`, `.r`, `.R`
- **Config/Data**: `.json`, `.yaml`, `.yml`, `.xml`, `.md`, `.txt`, `.toml`, `.ini`, `.cfg`, `.conf`, `.gradle`, `.dockerfile`, `.env`, `.lock`, `.puml`, `.pu`, `.dbml`

Special files like `.gitignore`, `package-lock.json`, and `yarn.lock` are always included.

## Ignored Directories
The following directories are skipped by default:
`node_modules`, `__pycache__`, `.git`, `.svn`, `.hg`, `venv`, `env`, `.venv`, `dist`, `build`, `out`, `coverage`, `.next`, `.nuxt`, `.cache`, `.idea`, `.vscode`, `.vs`, `.pytest_cache`, `.mypy_cache`, `.tox`, `.eggs`, `tmp`, `temp`, `vendor`, `.env`

Hidden directories (starting with `.`) are also skipped, except for the current directory itself.

## License
MIT
