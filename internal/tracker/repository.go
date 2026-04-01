package tracker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

type MarkdownRepository struct {
	rootPath    string
	npcPath     string
	playersPath string
	savesPath   string
}

type npcTemplateDocument struct {
	Name            string   `yaml:"name"`
	AC              *int     `yaml:"ac"`
	HP              *int     `yaml:"hp"`
	Dex             *int     `yaml:"dex"`
	InitiativeBonus *int     `yaml:"initiative_bonus"`
	Tags            []string `yaml:"tags"`
	Notes           string   `yaml:"notes"`
}

type playerTemplateDocument struct {
	Name            string `yaml:"name"`
	AC              *int   `yaml:"ac"`
	MaxHP           *int   `yaml:"max_hp"`
	CurrentHP       *int   `yaml:"current_hp"`
	Dex             *int   `yaml:"dex"`
	InitiativeBonus *int   `yaml:"initiative_bonus"`
	Notes           string `yaml:"notes"`
}

func NewMarkdownRepository(rootPath string) (*MarkdownRepository, error) {
	repository := &MarkdownRepository{
		rootPath:    rootPath,
		npcPath:     filepath.Join(rootPath, "npc"),
		playersPath: filepath.Join(rootPath, "players"),
		savesPath:   filepath.Join(rootPath, "saves"),
	}
	if err := repository.ensureDirectories(); err != nil {
		return nil, err
	}
	return repository, nil
}

func (r *MarkdownRepository) ensureDirectories() error {
	for _, path := range []string{r.npcPath, r.playersPath, r.savesPath} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (r *MarkdownRepository) ListNPCTemplates() ([]NpcTemplate, error) {
	entries, err := os.ReadDir(r.npcPath)
	if err != nil {
		return nil, err
	}

	templates := make([]NpcTemplate, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		template, err := r.loadNPCTemplate(filepath.Join(r.npcPath, entry.Name()))
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	sort.Slice(templates, func(left, right int) bool {
		return strings.ToLower(templates[left].Name) < strings.ToLower(templates[right].Name)
	})
	return templates, nil
}

func (r *MarkdownRepository) ListPlayerTemplates() ([]PlayerTemplate, error) {
	entries, err := os.ReadDir(r.playersPath)
	if err != nil {
		return nil, err
	}

	templates := make([]PlayerTemplate, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		template, err := r.loadPlayerTemplate(filepath.Join(r.playersPath, entry.Name()))
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	sort.Slice(templates, func(left, right int) bool {
		return strings.ToLower(templates[left].Name) < strings.ToLower(templates[right].Name)
	})
	return templates, nil
}

func (r *MarkdownRepository) ListEncounters() ([]EncounterSummary, error) {
	entries, err := os.ReadDir(r.savesPath)
	if err != nil {
		return nil, err
	}

	encounters := make([]EncounterSummary, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		frontMatter, _, err := r.readMarkdown(filepath.Join(r.savesPath, entry.Name()))
		if err != nil {
			continue
		}

		var summary EncounterSummary
		if err := yaml.Unmarshal([]byte(frontMatter), &summary); err != nil {
			continue
		}

		saveName := strings.TrimSuffix(entry.Name(), ".md")
		if summary.EncounterID == "" {
			summary.EncounterID = saveName
		}
		if summary.EncounterName == "" {
			summary.EncounterName = saveName
		}
		if summary.Round < 1 {
			summary.Round = 1
		}
		if info, err := entry.Info(); err == nil {
			summary.SavedAt = info.ModTime().Format("2006-01-02 15:04")
		}
		encounters = append(encounters, summary)
	}

	sort.Slice(encounters, func(left, right int) bool {
		return strings.ToLower(encounters[left].EncounterName) < strings.ToLower(encounters[right].EncounterName)
	})
	return encounters, nil
}

func (r *MarkdownRepository) SaveNPCTemplate(template NpcTemplate) (string, error) {
	document := npcTemplateDocument{
		Name:            template.Name,
		AC:              &template.AC,
		HP:              &template.HP,
		Dex:             &template.Dex,
		InitiativeBonus: copyIntPtr(template.InitiativeBonus),
		Tags:            append([]string{}, template.Tags...),
		Notes:           template.Notes,
	}
	if document.Tags == nil {
		document.Tags = []string{}
	}

	path := filepath.Join(r.npcPath, slugify(template.Name)+".md")
	return path, r.writeMarkdown(path, document, "")
}

func (r *MarkdownRepository) SavePlayerTemplate(template PlayerTemplate) (string, error) {
	template.Normalize()
	document := playerTemplateDocument{
		Name:            template.Name,
		AC:              copyIntPtr(template.AC),
		MaxHP:           copyIntPtr(template.MaxHP),
		CurrentHP:       copyIntPtr(template.CurrentHP),
		Dex:             copyIntPtr(template.Dex),
		InitiativeBonus: copyIntPtr(template.InitiativeBonus),
		Notes:           template.Notes,
	}

	path := filepath.Join(r.playersPath, slugify(template.Name)+".md")
	return path, r.writeMarkdown(path, document, "")
}

func (r *MarkdownRepository) SaveEncounter(encounter EncounterState) (string, error) {
	encounter = encounter.Clone()
	if err := encounter.NormalizeAndValidate(); err != nil {
		return "", err
	}
	path := filepath.Join(r.savesPath, encounter.EncounterID+".md")
	return path, r.writeMarkdown(path, encounter, "")
}

func (r *MarkdownRepository) LoadEncounter(saveName string) (EncounterState, error) {
	path := filepath.Join(r.savesPath, saveName+".md")
	frontMatter, _, err := r.readMarkdown(path)
	if err != nil {
		return EncounterState{}, err
	}

	var encounter EncounterState
	if err := yaml.Unmarshal([]byte(frontMatter), &encounter); err != nil {
		return EncounterState{}, err
	}
	if encounter.EncounterID == "" {
		encounter.EncounterID = saveName
	}
	if err := encounter.NormalizeAndValidate(); err != nil {
		return EncounterState{}, err
	}
	return encounter, nil
}

func (r *MarkdownRepository) LoadNPCTemplateByName(name string) (*NpcTemplate, error) {
	templates, err := r.ListNPCTemplates()
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		if strings.EqualFold(template.Name, name) {
			copyTemplate := template
			return &copyTemplate, nil
		}
	}
	return nil, nil
}

func (r *MarkdownRepository) LoadPlayerTemplateByName(name string) (*PlayerTemplate, error) {
	templates, err := r.ListPlayerTemplates()
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		if strings.EqualFold(template.Name, name) {
			copyTemplate := template
			return &copyTemplate, nil
		}
	}
	return nil, nil
}

func (r *MarkdownRepository) DeleteNPCTemplate(name string) (bool, error) {
	path := filepath.Join(r.npcPath, slugify(name)+".md")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, os.Remove(path)
}

func (r *MarkdownRepository) DeleteEncounter(encounterID string) (bool, error) {
	path := filepath.Join(r.savesPath, encounterID+".md")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, os.Remove(path)
}

func (r *MarkdownRepository) DeletePlayerTemplate(name string) (bool, error) {
	path := filepath.Join(r.playersPath, slugify(name)+".md")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, os.Remove(path)
}

func (r *MarkdownRepository) ParseNPCTemplateMarkdown(text string) (NpcTemplate, error) {
	frontMatter, body, err := parseMarkdownText(text, "markdown input")
	if err != nil {
		return NpcTemplate{}, err
	}

	var document npcTemplateDocument
	if err := yaml.Unmarshal([]byte(frontMatter), &document); err != nil {
		return NpcTemplate{}, err
	}

	return BuildNpcTemplate(
		document.Name,
		document.AC,
		document.HP,
		document.Dex,
		document.InitiativeBonus,
		document.Tags,
		combineNotes(document.Notes, body),
	)
}

func (r *MarkdownRepository) ParsePlayerTemplateMarkdown(text string) (PlayerTemplate, error) {
	frontMatter, body, err := parseMarkdownText(text, "markdown input")
	if err != nil {
		return PlayerTemplate{}, err
	}

	var document playerTemplateDocument
	if err := yaml.Unmarshal([]byte(frontMatter), &document); err != nil {
		return PlayerTemplate{}, err
	}

	return BuildPlayerTemplate(
		document.Name,
		document.AC,
		document.MaxHP,
		document.CurrentHP,
		document.Dex,
		document.InitiativeBonus,
		combineNotes(document.Notes, body),
	)
}

func (r *MarkdownRepository) loadNPCTemplate(path string) (NpcTemplate, error) {
	frontMatter, body, err := r.readMarkdown(path)
	if err != nil {
		return NpcTemplate{}, err
	}

	var document npcTemplateDocument
	if err := yaml.Unmarshal([]byte(frontMatter), &document); err != nil {
		return NpcTemplate{}, err
	}

	return BuildNpcTemplate(
		document.Name,
		document.AC,
		document.HP,
		document.Dex,
		document.InitiativeBonus,
		document.Tags,
		combineNotes(document.Notes, body),
	)
}

func (r *MarkdownRepository) loadPlayerTemplate(path string) (PlayerTemplate, error) {
	frontMatter, body, err := r.readMarkdown(path)
	if err != nil {
		return PlayerTemplate{}, err
	}

	var document playerTemplateDocument
	if err := yaml.Unmarshal([]byte(frontMatter), &document); err != nil {
		return PlayerTemplate{}, err
	}

	return BuildPlayerTemplate(
		document.Name,
		document.AC,
		document.MaxHP,
		document.CurrentHP,
		document.Dex,
		document.InitiativeBonus,
		combineNotes(document.Notes, body),
	)
}

func (r *MarkdownRepository) readMarkdown(path string) (string, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", path, err)
	}
	return parseMarkdownText(string(content), path)
}

func (r *MarkdownRepository) writeMarkdown(path string, frontMatter any, body string) error {
	content, err := yaml.Marshal(frontMatter)
	if err != nil {
		return err
	}

	text := "---\n" + strings.TrimSpace(string(content)) + "\n---\n"
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody != "" {
		text += "\n" + trimmedBody + "\n"
	}

	return os.WriteFile(path, []byte(text), 0o644)
}

func combineNotes(notes string, body string) string {
	parts := make([]string, 0, 2)
	if trimmed := strings.TrimSpace(notes); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(body); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "\n\n")
}

func parseMarkdownText(text string, source string) (string, string, error) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("%s must start with YAML front matter", source)
	}

	endIndex := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			endIndex = index
			break
		}
	}
	if endIndex == -1 {
		return "", "", fmt.Errorf("%s has invalid YAML front matter", source)
	}

	frontMatter := strings.Join(lines[1:endIndex], "\n")
	body := strings.Join(lines[endIndex+1:], "\n")
	body = strings.TrimLeft(body, "\n\r")
	body = strings.TrimRight(body, "\n\r")
	return frontMatter, body, nil
}

func slugify(value string) string {
	var builder strings.Builder
	lastWasDash := false

	for _, character := range strings.TrimSpace(strings.ToLower(value)) {
		switch {
		case unicode.IsLetter(character), unicode.IsDigit(character):
			builder.WriteRune(character)
			lastWasDash = false
		case character == ' ' || character == '-' || character == '_':
			if builder.Len() > 0 && !lastWasDash {
				builder.WriteByte('-')
				lastWasDash = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "encounter"
	}
	return slug
}
