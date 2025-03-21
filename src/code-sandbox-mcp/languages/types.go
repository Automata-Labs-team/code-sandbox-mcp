package languages

// Language represents a supported programming language
type Language string
type LanguageList []Language

// Supported languages
const (
	Python Language = "python"
	Go     Language = "go"
	NodeJS Language = "nodejs"
)

// Language configurations
type LanguageConfig struct {
	Image string // Docker image to use
	// Dependency management
	DependencyFiles []string // Files that indicate dependencies (e.g., go.mod, requirements.txt)
	InstallCommand  []string // Command to install dependencies (e.g., pip install -r requirements.txt)
	RunCommand      []string // Run command
	FileExtension   string   // File extension for the language
}

// AllLanguages contains all supported languages in a specific order
var AllLanguages = LanguageList{Python, Go, NodeJS}

// SupportedLanguages maps Language to their configurations
var SupportedLanguages = map[Language]LanguageConfig{
	Python: {
		Image:           "ghcr.io/astral-sh/uv:debian-slim",
		DependencyFiles: []string{"requirements.txt", "pyproject.toml", "setup.py"},
		InstallCommand:  []string{"pip", "install", "-r", "requirements.txt"},
		RunCommand:      []string{"uv", "run", "main.py"},
		FileExtension:   "py",
	},
	Go: {
		Image:           "docker.io/library/golang:1.23.6-bookworm",
		DependencyFiles: []string{"go.mod"},
		InstallCommand:  []string{"go", "mod", "tidy"},
		RunCommand:      []string{"go", "run", "main.go"},
		FileExtension:   "go",
	},
	NodeJS: {
		Image:           "oven/bun:debian",
		DependencyFiles: []string{"package.json"},
		InstallCommand:  []string{"npm", "install"},
		RunCommand:      []string{"bun", "run", "main.ts"},
		FileExtension:   "ts",
	},
}

// String returns the string representation of the language
func (l Language) String() string {
	return string(l)
}

// IsValid checks if the language is supported
func (l Language) IsValid() bool {
	for _, valid := range AllLanguages {
		if l == valid {
			return true
		}
	}
	return false
}

// ToArray converts the AllLanguages slice to an array of strings
func (l LanguageList) ToArray() []string {
	result := make([]string, len(l))
	for i, lang := range l {
		result[i] = string(lang)
	}
	return result
}
