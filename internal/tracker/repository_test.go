package tracker

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadPlayerTemplate(t *testing.T) {
	repository, err := NewMarkdownRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	template, err := BuildPlayerTemplate("Lia", nil, nil, nil, nil, intPtr(4), "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repository.SavePlayerTemplate(template); err != nil {
		t.Fatal(err)
	}

	loadedTemplate, err := repository.LoadPlayerTemplateByName("Lia")
	if err != nil {
		t.Fatal(err)
	}
	if loadedTemplate == nil {
		t.Fatal("expected player template")
	}
	if loadedTemplate.Name != "Lia" {
		t.Fatalf("expected Lia, got %s", loadedTemplate.Name)
	}
	if loadedTemplate.InitiativeBonus == nil || *loadedTemplate.InitiativeBonus != 4 {
		t.Fatalf("expected initiative bonus 4, got %#v", loadedTemplate.InitiativeBonus)
	}
}

func TestSaveAndLoadEncounter(t *testing.T) {
	repository, err := NewMarkdownRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	encounter := EncounterState{
		EncounterName: "Bridge Fight",
		Combatants: []Combatant{
			{
				Kind:            "npc",
				SourceName:      "Goblin",
				DisplayName:     "Goblin #1",
				TokenLabel:      stringPtr("B1"),
				MaxHP:           intPtr(7),
				CurrentHP:       intPtr(5),
				InitiativeTotal: intPtr(16),
			},
		},
	}

	path, err := repository.SaveEncounter(encounter)
	if err != nil {
		t.Fatal(err)
	}

	loadedEncounter, err := repository.LoadEncounter(trimMDExt(path))
	if err != nil {
		t.Fatal(err)
	}
	if loadedEncounter.EncounterName != "Bridge Fight" {
		t.Fatalf("expected Bridge Fight, got %s", loadedEncounter.EncounterName)
	}
	if loadedEncounter.Combatants[0].DisplayName != "Goblin #1" {
		t.Fatalf("expected Goblin #1, got %s", loadedEncounter.Combatants[0].DisplayName)
	}
	if loadedEncounter.Combatants[0].TokenLabel == nil || *loadedEncounter.Combatants[0].TokenLabel != "B1" {
		t.Fatalf("expected token B1, got %#v", loadedEncounter.Combatants[0].TokenLabel)
	}
}

func TestParseNPCTemplateMarkdown(t *testing.T) {
	repository, err := NewMarkdownRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	template, err := repository.ParseNPCTemplateMarkdown(`---
name: Goblin Boss
ac: 17
hp: 45
dex: 14
initiative_bonus: 2
tags:
  - goblinoid
notes: Leads the ambush.
---
Carries a horn.
`)
	if err != nil {
		t.Fatal(err)
	}

	if template.Name != "Goblin Boss" || template.AC != 17 || template.HP != 45 || template.Dex != 14 {
		t.Fatalf("unexpected template: %#v", template)
	}
	if template.InitiativeBonus == nil || *template.InitiativeBonus != 2 {
		t.Fatalf("expected initiative bonus 2, got %#v", template.InitiativeBonus)
	}
	if len(template.Tags) != 1 || template.Tags[0] != "goblinoid" {
		t.Fatalf("unexpected tags: %#v", template.Tags)
	}
	if template.Notes != "Leads the ambush.\n\nCarries a horn." {
		t.Fatalf("unexpected notes: %q", template.Notes)
	}
}

func TestParsePlayerTemplateMarkdown(t *testing.T) {
	repository, err := NewMarkdownRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	template, err := repository.ParsePlayerTemplateMarkdown(`---
name: Aramil
ac: 15
max_hp: 28
dex: 16
initiative_bonus: 3
notes: Keeps Bless ready.
---
Prefers longbow.
`)
	if err != nil {
		t.Fatal(err)
	}

	if template.Name != "Aramil" {
		t.Fatalf("expected Aramil, got %s", template.Name)
	}
	if template.AC == nil || *template.AC != 15 {
		t.Fatalf("expected AC 15, got %#v", template.AC)
	}
	if template.MaxHP == nil || *template.MaxHP != 28 {
		t.Fatalf("expected max HP 28, got %#v", template.MaxHP)
	}
	if template.CurrentHP == nil || *template.CurrentHP != 28 {
		t.Fatalf("expected current HP 28, got %#v", template.CurrentHP)
	}
	if template.Dex == nil || *template.Dex != 16 {
		t.Fatalf("expected dex 16, got %#v", template.Dex)
	}
	if template.InitiativeBonus == nil || *template.InitiativeBonus != 3 {
		t.Fatalf("expected initiative bonus 3, got %#v", template.InitiativeBonus)
	}
	if template.Notes != "Keeps Bless ready.\n\nPrefers longbow." {
		t.Fatalf("unexpected notes: %q", template.Notes)
	}
}

func stringPtr(value string) *string {
	return &value
}

func trimMDExt(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".md")
}
