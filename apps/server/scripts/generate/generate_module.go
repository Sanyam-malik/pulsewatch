package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/jinzhu/inflection"
)

type ModuleData struct {
	ModuleName string
}

func ToPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}

	return strings.Join(parts, "")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run ./scripts/generate/generate_module.go <module_name> (from apps/server)")
		os.Exit(1)
	}

	moduleName := strings.ToLower(os.Args[1])
	data := ModuleData{
		ModuleName: moduleName,
	}

	// Create module directory
	moduleDir := filepath.Join("internal", "modules", moduleName)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		fmt.Printf("Error creating module directory: %v\n", err)
		os.Exit(1)
	}

	templateDir := filepath.Join("templates", "module")
	files, err := os.ReadDir(templateDir)
	if err != nil {
		fmt.Printf("Error reading template directory: %v\n", err)
		os.Exit(1)
	}

	var templates []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".tmpl") {
			templates = append(templates, file.Name())
		}
	}

	funcs := template.FuncMap{
		"lower":  strings.ToLower,
		"pascal": ToPascalCase,
		"upper":  strings.ToUpper,
		"plural": inflection.Plural,
	}

	for _, tmpl := range templates {
		templatePath := filepath.Join("templates", "module", tmpl)
		outputPath := filepath.Join(moduleDir, moduleName+"."+strings.Replace(tmpl, ".tmpl", "", 1))

		// Read template
		tmplContent, err := os.ReadFile(templatePath)
		if err != nil {
			fmt.Printf("Error reading template %s: %v\n", tmpl, err)
			os.Exit(1)
		}

		// Parse and execute template
		t, err := template.New(tmpl).Funcs(funcs).Parse(string(tmplContent))
		if err != nil {
			fmt.Printf("Error parsing template %s: %v\n", tmpl, err)
			os.Exit(1)
		}

		// Create output file
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Printf("Error creating output file %s: %v\n", outputPath, err)
			os.Exit(1)
		}

		// Execute template
		if err := t.Execute(f, data); err != nil {
			fmt.Printf("Error executing template %s: %v\n", tmpl, err)
			os.Exit(1)
		}

		f.Close()
		fmt.Printf("Generated %s\n", outputPath)
	}

	fmt.Printf("\nModule '%s' has been generated successfully!\n", moduleName)
	fmt.Println("Next steps:")
	fmt.Println("1. Add your module dependencies in the New<ModuleName> function")
	fmt.Println("2. Implement your module-specific logic in the service and repository")
	fmt.Println("3. Add your module to the dependency injection setup")
}
