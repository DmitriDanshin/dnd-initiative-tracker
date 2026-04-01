package tracker

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

type ValidationError struct {
	Fields map[string]string
}

func (e ValidationError) Error() string {
	return "validation failed"
}

type NpcTemplate struct {
	Name            string   `json:"name" yaml:"name"`
	AC              int      `json:"ac" yaml:"ac"`
	HP              int      `json:"hp" yaml:"hp"`
	Dex             int      `json:"dex" yaml:"dex"`
	InitiativeBonus *int     `json:"initiative_bonus" yaml:"initiative_bonus"`
	Tags            []string `json:"tags" yaml:"tags"`
	Notes           string   `json:"notes" yaml:"notes"`
}

type PlayerTemplate struct {
	Name            string `json:"name" yaml:"name"`
	AC              *int   `json:"ac" yaml:"ac"`
	MaxHP           *int   `json:"max_hp" yaml:"max_hp"`
	CurrentHP       *int   `json:"current_hp" yaml:"current_hp"`
	Dex             *int   `json:"dex" yaml:"dex"`
	InitiativeBonus *int   `json:"initiative_bonus" yaml:"initiative_bonus"`
	Notes           string `json:"notes" yaml:"notes"`
}

type Combatant struct {
	Kind            string   `json:"kind" yaml:"kind"`
	SourceName      string   `json:"source_name" yaml:"source_name"`
	DisplayName     string   `json:"display_name" yaml:"display_name"`
	TokenLabel      *string  `json:"token_label" yaml:"token_label"`
	AC              *int     `json:"ac" yaml:"ac"`
	MaxHP           *int     `json:"max_hp" yaml:"max_hp"`
	CurrentHP       *int     `json:"current_hp" yaml:"current_hp"`
	Dex             *int     `json:"dex" yaml:"dex"`
	InitiativeBonus *int     `json:"initiative_bonus" yaml:"initiative_bonus"`
	InitiativeRoll  *int     `json:"initiative_roll" yaml:"initiative_roll"`
	InitiativeTotal *int     `json:"initiative_total" yaml:"initiative_total"`
	Notes           string   `json:"notes" yaml:"notes"`
	Statuses        []string `json:"statuses" yaml:"statuses"`
	SortIndex       int      `json:"sort_index" yaml:"sort_index"`
}

type EncounterState struct {
	EncounterID   string      `json:"encounter_id" yaml:"encounter_id"`
	EncounterName string      `json:"encounter_name" yaml:"encounter_name"`
	Round         int         `json:"round" yaml:"round"`
	ActiveIndex   int         `json:"active_index" yaml:"active_index"`
	Combatants    []Combatant `json:"combatants" yaml:"combatants"`
}

type EncounterSummary struct {
	EncounterID   string `json:"encounter_id" yaml:"encounter_id"`
	EncounterName string `json:"encounter_name" yaml:"encounter_name"`
	Round         int    `json:"round" yaml:"round"`
	SavedAt       string `json:"saved_at" yaml:"-"`
}

type SaveNPCTemplateInput struct {
	Name            string
	AC              *int
	HP              *int
	Dex             *int
	InitiativeBonus *int
	TagsRaw         string
	Notes           string
	Markdown        string
}

type SavePlayerTemplateInput struct {
	Name            string
	AC              *int
	MaxHP           *int
	CurrentHP       *int
	Dex             *int
	InitiativeBonus *int
	Notes           string
	Markdown        string
}

type PlayerRollRequest struct {
	Name  string `json:"name"`
	Bonus int    `json:"bonus"`
	Index int    `json:"index"`
}

func BuildNpcTemplate(
	name string,
	ac *int,
	hp *int,
	dex *int,
	initiativeBonus *int,
	tags []string,
	notes string,
) (NpcTemplate, error) {
	fields := map[string]string{}
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		fields["name"] = "Value error, NPC name must not be empty."
	}
	if ac == nil {
		fields["ac"] = "Field required."
	}
	if hp == nil {
		fields["hp"] = "Field required."
	}
	if dex == nil {
		fields["dex"] = "Field required."
	}
	if len(fields) > 0 {
		return NpcTemplate{}, ValidationError{Fields: fields}
	}

	cleanTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		cleaned := strings.TrimSpace(tag)
		if cleaned != "" {
			cleanTags = append(cleanTags, cleaned)
		}
	}

	template := NpcTemplate{
		Name:            cleanName,
		AC:              *ac,
		HP:              *hp,
		Dex:             *dex,
		InitiativeBonus: copyIntPtr(initiativeBonus),
		Tags:            cleanTags,
		Notes:           strings.TrimSpace(notes),
	}
	if template.Tags == nil {
		template.Tags = []string{}
	}
	return template, nil
}

func BuildPlayerTemplate(
	name string,
	ac *int,
	maxHP *int,
	currentHP *int,
	dex *int,
	initiativeBonus *int,
	notes string,
) (PlayerTemplate, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return PlayerTemplate{}, ValidationError{
			Fields: map[string]string{"name": "Value error, Player name must not be empty."},
		}
	}

	template := PlayerTemplate{
		Name:            cleanName,
		AC:              copyIntPtr(ac),
		MaxHP:           copyIntPtr(maxHP),
		CurrentHP:       copyIntPtr(currentHP),
		Dex:             copyIntPtr(dex),
		InitiativeBonus: copyIntPtr(initiativeBonus),
		Notes:           strings.TrimSpace(notes),
	}
	template.Normalize()
	return template, nil
}

func (p *PlayerTemplate) Normalize() {
	if p.CurrentHP == nil && p.MaxHP != nil {
		p.CurrentHP = copyIntPtr(p.MaxHP)
	}
}

func (c *Combatant) Normalize() {
	if c.TokenLabel != nil {
		cleaned := strings.ToUpper(strings.TrimSpace(*c.TokenLabel))
		if cleaned == "" {
			c.TokenLabel = nil
		} else {
			c.TokenLabel = &cleaned
		}
	}
	if c.CurrentHP == nil && c.MaxHP != nil {
		c.CurrentHP = copyIntPtr(c.MaxHP)
	}
	if c.Statuses == nil {
		c.Statuses = []string{}
	}
}

func (c Combatant) Clone() Combatant {
	clone := c
	clone.TokenLabel = copyStringPtr(c.TokenLabel)
	clone.AC = copyIntPtr(c.AC)
	clone.MaxHP = copyIntPtr(c.MaxHP)
	clone.CurrentHP = copyIntPtr(c.CurrentHP)
	clone.Dex = copyIntPtr(c.Dex)
	clone.InitiativeBonus = copyIntPtr(c.InitiativeBonus)
	clone.InitiativeRoll = copyIntPtr(c.InitiativeRoll)
	clone.InitiativeTotal = copyIntPtr(c.InitiativeTotal)
	clone.Statuses = append([]string{}, c.Statuses...)
	if clone.Statuses == nil {
		clone.Statuses = []string{}
	}
	return clone
}

func (e *EncounterState) NormalizeAndValidate() error {
	if strings.TrimSpace(e.EncounterName) == "" {
		return errors.New("encounter name must not be empty")
	}
	if e.EncounterID == "" {
		e.EncounterID = newEncounterID()
	}
	if e.Round < 1 {
		e.Round = 1
	}
	if e.ActiveIndex < 0 {
		e.ActiveIndex = 0
	}
	seenLabels := map[string]struct{}{}
	for index := range e.Combatants {
		e.Combatants[index].Normalize()
		if e.Combatants[index].TokenLabel == nil {
			continue
		}
		label := *e.Combatants[index].TokenLabel
		if _, exists := seenLabels[label]; exists {
			return errors.New("Token labels must be unique inside one encounter.")
		}
		seenLabels[label] = struct{}{}
	}
	if len(e.Combatants) == 0 {
		e.ActiveIndex = 0
	} else if e.ActiveIndex >= len(e.Combatants) {
		e.ActiveIndex = 0
	}
	return nil
}

func (e EncounterState) Clone() EncounterState {
	clone := e
	clone.Combatants = cloneCombatants(e.Combatants)
	return clone
}

func cloneCombatants(combatants []Combatant) []Combatant {
	cloned := make([]Combatant, len(combatants))
	for index, combatant := range combatants {
		cloned[index] = combatant.Clone()
	}
	return cloned
}

func copyIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func copyStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func newEncounterID() string {
	buffer := make([]byte, 16)
	if _, err := crand.Read(buffer); err != nil {
		return "encounter"
	}
	return hex.EncodeToString(buffer)
}
