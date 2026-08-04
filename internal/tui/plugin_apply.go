package tui

import (
	"fmt"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	applypkg "github.com/snakepilot10/ozsh/internal/apply"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
)

type pluginApplyResult struct {
	err error
}

func doApplyWithPlugins(cfg *config.Config, changes plugins.ChangeSet) tea.Cmd {
	configSnapshot := cloneConfig(cfg)
	changeSnapshot := changes.Clone()
	return func() tea.Msg {
		err := applypkg.Apply(applypkg.Request{
			Config:        configSnapshot,
			PluginChanges: changeSnapshot,
		})
		return pluginApplyResult{err: err}
	}
}

func (m Model) reviewedPluginChangesSnapshot() plugins.ChangeSet {
	return m.pluginWizard.ReviewChanges.Clone()
}

func (m Model) pendingPluginReview() string {
	changes := m.pluginWizard.ReviewChanges
	if changes.Empty() {
		changes = m.pluginChanges
	}
	adds, removes := changes.Counts()

	var body strings.Builder
	body.WriteString(renderGroupLabel("Plugin changes"))
	body.WriteString("\n")
	body.WriteString(fmt.Sprintf("%d add · %d remove", adds, removes))

	for _, add := range changes.Adds {
		body.WriteString("\n\n")
		body.WriteString(accentStyle.Render("+ " + changeString(add, "Name")))
		writeChangeDetail(&body, "Repository", changeString(add, "RepositoryURL", "URL"))
		writeChangeDetail(&body, "Load file", changeString(add, "Load"))
		writeChangeDetail(&body, "Final path", changeString(add, "FinalDir", "Destination", "Source"))
	}
	for _, removal := range changes.Removes {
		body.WriteString("\n\n")
		body.WriteString(errorStyle.Render("- " + changeString(removal, "Name")))
		writeChangeDetail(&body, "Remove path", changeString(removal, "Source", "OriginalSource", "FinalDir"))
	}
	if adds == 0 && removes == 0 {
		body.WriteString("\n")
		body.WriteString(renderHint("No plugin filesystem changes pending"))
	}
	return body.String()
}

func (m Model) appendPendingPluginReview(base string, spec layoutSpec) string {
	changes := m.pluginWizard.ReviewChanges
	if changes.Empty() {
		return base
	}
	return fitHeight(base+"\n\n"+m.pendingPluginReview(), spec.contentWidth, spec.workspaceHeight)
}

func writeChangeDetail(body *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	body.WriteString("\n")
	body.WriteString(renderKeyValue(label, value))
}

func changeString(value any, names ...string) string {
	visited := map[uintptr]bool{}
	for _, name := range names {
		if result := findNamedString(reflect.ValueOf(value), name, visited, 0); result != "" {
			return result
		}
	}
	return "unknown"
}

func findNamedString(value reflect.Value, name string, visited map[uintptr]bool, depth int) string {
	if !value.IsValid() || depth > 4 {
		return ""
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		if value.Kind() == reflect.Pointer {
			pointer := value.Pointer()
			if pointer != 0 && visited[pointer] {
				return ""
			}
			if pointer != 0 {
				visited[pointer] = true
			}
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ""
	}
	if field := value.FieldByName(name); field.IsValid() && field.Kind() == reflect.String {
		return strings.TrimSpace(field.String())
	}
	typeInfo := value.Type()
	for index := 0; index < value.NumField(); index++ {
		fieldInfo := typeInfo.Field(index)
		if fieldInfo.PkgPath != "" {
			continue
		}
		field := value.Field(index)
		switch field.Kind() {
		case reflect.Struct, reflect.Pointer, reflect.Interface:
			if result := findNamedString(field, name, visited, depth+1); result != "" {
				return result
			}
		}
	}
	return ""
}
