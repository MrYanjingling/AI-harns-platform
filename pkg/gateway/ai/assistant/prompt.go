package assistant

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

type systemPromptData struct {
	ModelName string
	Tools     []string
}

func renderSystemPrompt(lang string, data systemPromptData) (string, error) {
	name := "prompts/system.zh-CN.tmpl"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "en") {
		name = "prompts/system.en-US.tmpl"
	}
	raw, err := promptFS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", name, err)
	}
	tpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse prompt %s: %w", name, err)
	}
	var output bytes.Buffer
	if err := tpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("render prompt %s: %w", name, err)
	}
	return output.String(), nil
}
