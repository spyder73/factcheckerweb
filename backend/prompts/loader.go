package prompts

import (
	"embed"
	"strings"
)

//go:embed assets/*.txt
var promptFiles embed.FS

// Prompts holds all loaded prompt templates
var Prompts = loadPrompts()

type PromptTemplates struct {
	ImageAnalysis         string
	ImageAnalysisFallback string
	CondenseSystem        string
	CondenseUser          string
	EvaluateSystem        string
	EvaluateUser          string
}

func loadPrompts() PromptTemplates {
	return PromptTemplates{
		ImageAnalysis:         mustRead("assets/image_analysis.txt"),
		ImageAnalysisFallback: mustRead("assets/image_analysis_fallback.txt"),
		CondenseSystem:        mustRead("assets/condense_system.txt"),
		CondenseUser:          mustRead("assets/condense_user.txt"),
		EvaluateSystem:        mustRead("assets/evaluate_system.txt"),
		EvaluateUser:          mustRead("assets/evaluate_user.txt"),
	}
}

func mustRead(filename string) string {
	data, err := promptFiles.ReadFile(filename)
	if err != nil {
		panic("failed to load prompt: " + filename + ": " + err.Error())
	}
	return string(data)
}

// Replace replaces placeholders in a template
func Replace(template string, replacements map[string]string) string {
	result := template
	for key, value := range replacements {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
