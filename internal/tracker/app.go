package tracker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type TrackerApp struct {
	mu                 sync.Mutex
	repository         *MarkdownRepository
	mode               string
	message            string
	fieldErrors        map[string]string
	selectedIndex      int
	setupEncounterName string
	setupCombatants    []Combatant
	currentEncounter   *EncounterState
}

func NewTrackerApp(rootPath string) (*TrackerApp, error) {
	repository, err := NewMarkdownRepository(rootPath)
	if err != nil {
		return nil, err
	}

	app := &TrackerApp{
		repository:         repository,
		mode:               "home",
		fieldErrors:        map[string]string{},
		setupEncounterName: "New Encounter",
		setupCombatants:    []Combatant{},
	}
	if err := app.restoreLastEncounter(); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *TrackerApp) GetState() (map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	npcTemplates, err := a.repository.ListNPCTemplates()
	if err != nil {
		return nil, err
	}
	playerTemplates, err := a.repository.ListPlayerTemplates()
	if err != nil {
		return nil, err
	}

	playerNames := make([]string, len(playerTemplates))
	for index, template := range playerTemplates {
		playerNames[index] = template.Name
	}

	state := map[string]any{
		"mode":             a.mode,
		"message":          a.message,
		"field_errors":     cloneStringMap(a.fieldErrors),
		"selected_index":   a.selectedIndex,
		"npc_templates":    summarizeNPCTemplates(npcTemplates),
		"player_templates": playerNames,
	}

	switch a.mode {
	case "home":
		encounters, err := a.repository.ListEncounters()
		if err != nil {
			return nil, err
		}
		state["encounters"] = encounters
	case "setup":
		state["setup_encounter_name"] = a.setupEncounterName
		state["setup_combatants"] = cloneCombatants(a.setupCombatants)
	case "combat":
		if a.currentEncounter != nil {
			state["encounter"] = a.currentEncounter.Clone()
		}
	}

	return state, nil
}

func (a *TrackerApp) GoHome() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentEncounter != nil {
		if _, err := a.repository.SaveEncounter(*a.currentEncounter); err != nil {
			return err
		}
		if err := a.saveLastEncounterID(a.currentEncounter.EncounterID); err != nil {
			return err
		}
	}

	if err := a.clearLastEncounter(); err != nil {
		return err
	}
	a.currentEncounter = nil
	a.mode = "home"
	a.selectedIndex = 0
	a.message = ""
	a.fieldErrors = map[string]string{}
	return nil
}

func (a *TrackerApp) StartNewEncounter() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mode = "setup"
	a.setupEncounterName = generateEncounterName()
	a.setupCombatants = []Combatant{}
	a.selectedIndex = 0
	a.message = ""
	a.fieldErrors = map[string]string{}
}

func (a *TrackerApp) ResumeEncounter(encounterID string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	encounter, err := a.repository.LoadEncounter(encounterID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			a.currentEncounter = nil
			a.mode = "home"
			a.selectedIndex = 0
			a.fieldErrors = map[string]string{}
			a.message = fmt.Sprintf("Encounter '%s' not found.", encounterID)
			return false, nil
		}
		return false, err
	}

	a.currentEncounter = &encounter
	a.mode = "combat"
	a.selectedIndex = encounter.ActiveIndex
	a.fieldErrors = map[string]string{}
	if len(encounter.Combatants) > 0 {
		active := encounter.Combatants[encounter.ActiveIndex]
		a.message = fmt.Sprintf("Resumed. Round %d, active: %s.", encounter.Round, active.DisplayName)
	} else {
		a.message = fmt.Sprintf("Resumed. Round %d.", encounter.Round)
	}
	if err := a.saveLastEncounterID(encounterID); err != nil {
		return false, err
	}
	return true, nil
}

func (a *TrackerApp) SetEncounterName(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.setupEncounterName = name
	a.message = fmt.Sprintf("Encounter name set to %s.", name)
	a.fieldErrors = map[string]string{}
}

func (a *TrackerApp) AddNPC(name string, count int, labelsRaw string, hpOverride *int, acOverride *int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	npcTemplate, err := a.repository.LoadNPCTemplateByName(name)
	if err != nil {
		return err
	}
	if npcTemplate == nil {
		a.message = fmt.Sprintf("NPC '%s' not found.", name)
		return nil
	}
	if count < 1 {
		a.message = "NPC count must be at least 1."
		return nil
	}

	tokenLabels := splitCSV(labelsRaw)
	if len(tokenLabels) > 0 && len(tokenLabels) != count {
		a.message = "Token label count must match NPC count."
		return nil
	}

	existingCount := 0
	for _, combatant := range a.setupCombatants {
		if combatant.Kind == "npc" && combatant.SourceName == npcTemplate.Name {
			existingCount++
		}
	}

	effectiveHP := npcTemplate.HP
	effectiveAC := npcTemplate.AC
	if hpOverride != nil {
		effectiveHP = *hpOverride
	}
	if acOverride != nil {
		effectiveAC = *acOverride
	}

	for index := 0; index < count; index++ {
		var tokenLabel *string
		if index < len(tokenLabels) {
			label := tokenLabels[index]
			tokenLabel = &label
		}
		combatant := a.combatantFromNPCTemplate(
			*npcTemplate,
			fmt.Sprintf("%s #%d", npcTemplate.Name, existingCount+index+1),
			tokenLabel,
			effectiveHP,
			effectiveAC,
		)
		a.setupCombatants = append(a.setupCombatants, combatant)
	}

	a.selectedIndex = len(a.setupCombatants) - 1
	a.message = fmt.Sprintf("Added %d %s.", count, npcTemplate.Name)
	a.fieldErrors = map[string]string{}
	return nil
}

func (a *TrackerApp) AddPlayer(name string, initiativeBonus *int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if strings.TrimSpace(name) == "" {
		a.message = "Player name is required."
		return nil
	}

	for _, combatant := range a.setupCombatants {
		if combatant.Kind == "player" && strings.EqualFold(combatant.SourceName, name) {
			a.message = fmt.Sprintf("Player '%s' is already in the roster.", name)
			return nil
		}
	}

	playerTemplate, err := a.repository.LoadPlayerTemplateByName(name)
	if err != nil {
		return err
	}
	if playerTemplate == nil {
		template, err := BuildPlayerTemplate(name, nil, nil, nil, nil, initiativeBonus, "")
		if err != nil {
			return err
		}
		if _, err := a.repository.SavePlayerTemplate(template); err != nil {
			return err
		}
		playerTemplate = &template
	}

	combatant := Combatant{
		Kind:            "player",
		SourceName:      playerTemplate.Name,
		DisplayName:     playerTemplate.Name,
		AC:              copyIntPtr(playerTemplate.AC),
		MaxHP:           copyIntPtr(playerTemplate.MaxHP),
		CurrentHP:       copyIntPtr(playerTemplate.CurrentHP),
		Dex:             copyIntPtr(playerTemplate.Dex),
		InitiativeBonus: copyIntPtr(playerTemplate.InitiativeBonus),
		Notes:           playerTemplate.Notes,
		Statuses:        []string{},
		SortIndex:       len(a.setupCombatants),
	}
	combatant.Normalize()

	a.setupCombatants = append(a.setupCombatants, combatant)
	a.selectedIndex = len(a.setupCombatants) - 1
	a.message = fmt.Sprintf("Added player %s.", playerTemplate.Name)
	a.fieldErrors = map[string]string{}
	return nil
}

func (a *TrackerApp) SaveNPCTemplate(input SaveNPCTemplateInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var (
		template NpcTemplate
		err      error
	)

	if strings.TrimSpace(input.Markdown) != "" {
		template, err = a.repository.ParseNPCTemplateMarkdown(input.Markdown)
	} else {
		template, err = BuildNpcTemplate(
			input.Name,
			input.AC,
			input.HP,
			input.Dex,
			input.InitiativeBonus,
			splitCSV(input.TagsRaw),
			input.Notes,
		)
	}
	if err != nil {
		a.message = "NPC template validation failed."
		a.fieldErrors = extractFieldErrors(err, "markdown")
		return nil
	}

	if _, err := a.repository.SaveNPCTemplate(template); err != nil {
		return err
	}
	a.message = fmt.Sprintf("Saved NPC template %s.", template.Name)
	a.fieldErrors = map[string]string{}
	return nil
}

func (a *TrackerApp) SavePlayerTemplate(input SavePlayerTemplateInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var (
		template PlayerTemplate
		err      error
	)

	if strings.TrimSpace(input.Markdown) != "" {
		template, err = a.repository.ParsePlayerTemplateMarkdown(input.Markdown)
	} else {
		template, err = BuildPlayerTemplate(
			input.Name,
			input.AC,
			input.MaxHP,
			input.CurrentHP,
			input.Dex,
			input.InitiativeBonus,
			input.Notes,
		)
	}
	if err != nil {
		a.message = "Player template validation failed."
		a.fieldErrors = extractFieldErrors(err, "markdown")
		return nil
	}

	if _, err := a.repository.SavePlayerTemplate(template); err != nil {
		return err
	}
	a.message = fmt.Sprintf("Saved player template %s.", template.Name)
	a.fieldErrors = map[string]string{}
	return nil
}

func (a *TrackerApp) GetNPCTemplate(name string) (*NpcTemplate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.repository.LoadNPCTemplateByName(name)
}

func (a *TrackerApp) GetPlayerTemplate(name string) (*PlayerTemplate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.repository.LoadPlayerTemplateByName(name)
}

func (a *TrackerApp) DeleteNPCTemplate(name string) ([]NpcTemplate, string, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	deleted, err := a.repository.DeleteNPCTemplate(name)
	if err != nil {
		return nil, "", false, err
	}
	if deleted {
		a.message = fmt.Sprintf("Deleted NPC template '%s'.", name)
	} else {
		a.message = fmt.Sprintf("NPC template '%s' not found.", name)
	}
	a.fieldErrors = map[string]string{}

	templates, err := a.repository.ListNPCTemplates()
	if err != nil {
		return nil, "", false, err
	}
	return templates, a.message, deleted, nil
}

func (a *TrackerApp) DeletePlayerTemplate(name string) ([]PlayerTemplate, string, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	deleted, err := a.repository.DeletePlayerTemplate(name)
	if err != nil {
		return nil, "", false, err
	}
	if deleted {
		a.message = fmt.Sprintf("Deleted player template '%s'.", name)
	} else {
		a.message = fmt.Sprintf("Player template '%s' not found.", name)
	}
	a.fieldErrors = map[string]string{}

	templates, err := a.repository.ListPlayerTemplates()
	if err != nil {
		return nil, "", false, err
	}
	return templates, a.message, deleted, nil
}

func (a *TrackerApp) ListEncounters() ([]EncounterSummary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.repository.ListEncounters()
}

func (a *TrackerApp) DeleteEncounter(encounterID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	deleted, err := a.repository.DeleteEncounter(encounterID)
	if err != nil {
		return "", err
	}
	if !deleted {
		return fmt.Sprintf("Encounter '%s' not found.", encounterID), nil
	}

	if a.currentEncounter != nil && a.currentEncounter.EncounterID == encounterID {
		a.currentEncounter = nil
		a.mode = "home"
	}

	return "Encounter deleted.", nil
}

func (a *TrackerApp) RenameEncounter(encounterID string, newName string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cleanName := strings.TrimSpace(newName)
	if cleanName == "" {
		return "Encounter name must not be empty.", nil
	}

	encounter, err := a.repository.LoadEncounter(encounterID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Sprintf("Encounter '%s' not found.", encounterID), nil
		}
		return "", err
	}

	encounter.EncounterName = cleanName
	if _, err := a.repository.SaveEncounter(encounter); err != nil {
		return "", err
	}

	if a.currentEncounter != nil && a.currentEncounter.EncounterID == encounterID {
		a.currentEncounter.EncounterName = cleanName
	}

	return fmt.Sprintf("Renamed to '%s'.", cleanName), nil
}

func (a *TrackerApp) ListNPCTemplates() ([]NpcTemplate, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	templates, err := a.repository.ListNPCTemplates()
	return templates, a.message, err
}

func (a *TrackerApp) ListPlayerTemplates() ([]PlayerTemplate, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	templates, err := a.repository.ListPlayerTemplates()
	return templates, a.message, err
}

func (a *TrackerApp) AddNPCToCombat(name string, count int, labelsRaw string, hpOverride *int, acOverride *int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentEncounter == nil {
		a.message = "No active encounter."
		return nil
	}

	npcTemplate, err := a.repository.LoadNPCTemplateByName(name)
	if err != nil {
		return err
	}
	if npcTemplate == nil {
		a.message = fmt.Sprintf("NPC '%s' not found.", name)
		return nil
	}
	if count < 1 {
		a.message = "NPC count must be at least 1."
		return nil
	}

	tokenLabels := splitCSV(labelsRaw)
	if len(tokenLabels) > 0 && len(tokenLabels) != count {
		a.message = "Token label count must match NPC count."
		return nil
	}

	existingCount := 0
	for _, combatant := range a.currentEncounter.Combatants {
		if combatant.Kind == "npc" && combatant.SourceName == npcTemplate.Name {
			existingCount++
		}
	}

	effectiveHP := npcTemplate.HP
	effectiveAC := npcTemplate.AC
	if hpOverride != nil {
		effectiveHP = *hpOverride
	}
	if acOverride != nil {
		effectiveAC = *acOverride
	}

	combatants := cloneCombatants(a.currentEncounter.Combatants)
	for index := 0; index < count; index++ {
		var tokenLabel *string
		if index < len(tokenLabels) {
			label := tokenLabels[index]
			tokenLabel = &label
		}

		combatant := a.combatantFromNPCTemplate(
			*npcTemplate,
			fmt.Sprintf("%s #%d", npcTemplate.Name, existingCount+index+1),
			tokenLabel,
			effectiveHP,
			effectiveAC,
		)
		combatant.SortIndex = len(combatants)
		AssignNPCInitiative(&combatant)
		combatants = append(combatants, combatant)
	}

	a.currentEncounter.Combatants = SortCombatantsForInitiative(combatants)
	a.selectedIndex = a.currentEncounter.ActiveIndex
	a.message = fmt.Sprintf("Added %d %s to combat.", count, npcTemplate.Name)
	a.fieldErrors = map[string]string{}
	return a.autosaveLocked()
}

func (a *TrackerApp) RollNPCInitiative() {
	a.mu.Lock()
	defer a.mu.Unlock()

	npcCount := 0
	for index := range a.setupCombatants {
		if a.setupCombatants[index].Kind == "npc" {
			AssignNPCInitiative(&a.setupCombatants[index])
			npcCount++
		}
	}
	a.message = fmt.Sprintf("Rolled initiative for %d NPC(s).", npcCount)
}

func (a *TrackerApp) RemoveCombatant(index int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if index < 0 || index >= len(a.setupCombatants) {
		return
	}
	removed := a.setupCombatants[index]
	a.setupCombatants = append(a.setupCombatants[:index], a.setupCombatants[index+1:]...)
	if len(a.setupCombatants) == 0 {
		a.selectedIndex = 0
	} else if a.selectedIndex >= len(a.setupCombatants) {
		a.selectedIndex = len(a.setupCombatants) - 1
	}
	a.message = fmt.Sprintf("Removed %s.", removed.DisplayName)
}

func (a *TrackerApp) PlayersNeedingRolls() ([]PlayerRollRequest, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.setupCombatants) == 0 {
		a.message = "Add at least one combatant first."
		return nil, false
	}

	players := make([]PlayerRollRequest, 0)
	for index, combatant := range a.setupCombatants {
		if combatant.Kind == "player" && combatant.InitiativeTotal == nil {
			bonus := 0
			if combatant.InitiativeBonus != nil {
				bonus = *combatant.InitiativeBonus
			}
			players = append(players, PlayerRollRequest{
				Name:  combatant.DisplayName,
				Bonus: bonus,
				Index: index,
			})
		}
	}
	return players, true
}

func (a *TrackerApp) StartEncounter(rolls []*int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.setupCombatants) == 0 {
		a.message = "Add at least one combatant first."
		return nil
	}

	combatants := cloneCombatants(a.setupCombatants)
	for index := range combatants {
		if combatants[index].Kind == "npc" && combatants[index].InitiativeTotal == nil {
			AssignNPCInitiative(&combatants[index])
		}
	}

	encounter := EncounterState{
		EncounterName: a.setupEncounterName,
		Round:         1,
		ActiveIndex:   0,
		Combatants:    combatants,
	}
	if err := encounter.NormalizeAndValidate(); err != nil {
		return err
	}

	playerIndexes := make([]int, 0)
	for index, combatant := range encounter.Combatants {
		if combatant.Kind == "player" && combatant.InitiativeTotal == nil {
			playerIndexes = append(playerIndexes, index)
		}
	}
	for rollIndex, combatantIndex := range playerIndexes {
		if rollIndex >= len(rolls) || rolls[rollIndex] == nil {
			continue
		}
		AssignPlayerInitiative(&encounter.Combatants[combatantIndex], *rolls[rollIndex])
	}

	encounter.Combatants = SortCombatantsForInitiative(encounter.Combatants)
	a.currentEncounter = &encounter
	a.mode = "combat"
	a.selectedIndex = encounter.ActiveIndex
	if len(encounter.Combatants) > 0 {
		active := encounter.Combatants[encounter.ActiveIndex]
		a.message = fmt.Sprintf("Combat started. Round %d, active: %s.", encounter.Round, active.DisplayName)
	} else {
		a.message = fmt.Sprintf("Combat started. Round %d.", encounter.Round)
	}
	return a.autosaveLocked()
}

func (a *TrackerApp) AdvanceTurn() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentEncounter == nil || len(a.currentEncounter.Combatants) == 0 {
		return nil
	}

	total := len(a.currentEncounter.Combatants)
	steps := 0
	for steps < total {
		a.currentEncounter.ActiveIndex++
		if a.currentEncounter.ActiveIndex >= total {
			a.currentEncounter.ActiveIndex = 0
			a.currentEncounter.Round++
		}
		steps++
		active := a.currentEncounter.Combatants[a.currentEncounter.ActiveIndex]
		if !isDowned(active) {
			break
		}
	}

	a.selectedIndex = a.currentEncounter.ActiveIndex
	active := a.currentEncounter.Combatants[a.currentEncounter.ActiveIndex]
	a.message = fmt.Sprintf("Round %d, active: %s.", a.currentEncounter.Round, active.DisplayName)
	return a.autosaveLocked()
}

func (a *TrackerApp) ApplyHPDelta(index int, delta int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentEncounter == nil || index < 0 || index >= len(a.currentEncounter.Combatants) {
		return nil
	}

	combatant := &a.currentEncounter.Combatants[index]
	if combatant.CurrentHP == nil {
		a.message = fmt.Sprintf("%s does not track HP.", combatant.DisplayName)
		return nil
	}

	currentHP := *combatant.CurrentHP + delta
	if currentHP < 0 {
		currentHP = 0
	}
	if combatant.MaxHP != nil && currentHP > *combatant.MaxHP {
		currentHP = *combatant.MaxHP
	}
	combatant.CurrentHP = &currentHP

	maxHPDisplay := "-"
	if combatant.MaxHP != nil {
		maxHPDisplay = fmt.Sprintf("%d", *combatant.MaxHP)
	}
	a.message = fmt.Sprintf("%s HP is now %d/%s.", combatant.DisplayName, currentHP, maxHPDisplay)
	return a.autosaveLocked()
}

func (a *TrackerApp) EditCombatant(index int, displayName *string, ac *int, maxHP *int, currentHP *int, notes *string, initiativeTotal *int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentEncounter == nil || index < 0 || index >= len(a.currentEncounter.Combatants) {
		return nil
	}

	combatant := &a.currentEncounter.Combatants[index]

	if displayName != nil {
		name := strings.TrimSpace(*displayName)
		if name != "" {
			combatant.DisplayName = name
		}
	}
	if ac != nil {
		combatant.AC = copyIntPtr(ac)
	}
	if maxHP != nil {
		combatant.MaxHP = copyIntPtr(maxHP)
	}
	if currentHP != nil {
		hp := *currentHP
		if hp < 0 {
			hp = 0
		}
		if combatant.MaxHP != nil && hp > *combatant.MaxHP {
			hp = *combatant.MaxHP
		}
		combatant.CurrentHP = &hp
	}
	if notes != nil {
		combatant.Notes = *notes
	}

	needsResort := false
	if initiativeTotal != nil {
		combatant.InitiativeTotal = copyIntPtr(initiativeTotal)
		needsResort = true
	}

	if needsResort {
		activeID := a.currentEncounter.Combatants[a.currentEncounter.ActiveIndex].DisplayName
		a.currentEncounter.Combatants = SortCombatantsForInitiative(a.currentEncounter.Combatants)
		for i, c := range a.currentEncounter.Combatants {
			if c.DisplayName == activeID {
				a.currentEncounter.ActiveIndex = i
				break
			}
		}
	}

	a.selectedIndex = index
	a.message = fmt.Sprintf("Updated %s.", combatant.DisplayName)
	a.fieldErrors = map[string]string{}
	return a.autosaveLocked()
}

func (a *TrackerApp) SaveEncounter() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.currentEncounter == nil {
		return nil
	}
	if _, err := a.repository.SaveEncounter(*a.currentEncounter); err != nil {
		return err
	}
	a.message = "Encounter saved."
	a.fieldErrors = map[string]string{}
	return nil
}

func (a *TrackerApp) Select(index int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.selectedIndex = index
}

func (a *TrackerApp) autosaveLocked() error {
	if a.currentEncounter == nil {
		return nil
	}
	if _, err := a.repository.SaveEncounter(*a.currentEncounter); err != nil {
		return err
	}
	return a.saveLastEncounterID(a.currentEncounter.EncounterID)
}

func (a *TrackerApp) restoreLastEncounter() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	lastFile := filepath.Join(a.repository.savesPath, "_last.txt")
	content, err := os.ReadFile(lastFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	encounterID := strings.TrimSpace(string(content))
	if encounterID == "" {
		return nil
	}
	savePath := filepath.Join(a.repository.savesPath, encounterID+".md")
	if _, err := os.Stat(savePath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	encounter, err := a.repository.LoadEncounter(encounterID)
	if err != nil {
		return err
	}
	a.currentEncounter = &encounter
	a.mode = "combat"
	a.selectedIndex = encounter.ActiveIndex
	if len(encounter.Combatants) > 0 {
		active := encounter.Combatants[encounter.ActiveIndex]
		a.message = fmt.Sprintf("Restored. Round %d, active: %s.", encounter.Round, active.DisplayName)
	}
	return nil
}

func (a *TrackerApp) saveLastEncounterID(encounterID string) error {
	lastFile := filepath.Join(a.repository.savesPath, "_last.txt")
	return os.WriteFile(lastFile, []byte(encounterID), 0o644)
}

func (a *TrackerApp) clearLastEncounter() error {
	lastFile := filepath.Join(a.repository.savesPath, "_last.txt")
	if err := os.Remove(lastFile); errors.Is(err, os.ErrNotExist) {
		return nil
	} else {
		return err
	}
}

func (a *TrackerApp) combatantFromNPCTemplate(template NpcTemplate, displayName string, tokenLabel *string, hp int, ac int) Combatant {
	if hp < 0 {
		hp = 0
	}
	combatant := Combatant{
		Kind:            "npc",
		SourceName:      template.Name,
		DisplayName:     displayName,
		TokenLabel:      copyStringPtr(tokenLabel),
		AC:              &ac,
		MaxHP:           &hp,
		CurrentHP:       &hp,
		Dex:             intPtr(template.Dex),
		InitiativeBonus: copyIntPtr(template.InitiativeBonus),
		Notes:           template.Notes,
		Statuses:        []string{},
		SortIndex:       len(a.setupCombatants),
	}
	combatant.Normalize()
	return combatant
}

func summarizeNPCTemplates(templates []NpcTemplate) []map[string]any {
	summary := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		summary = append(summary, map[string]any{
			"name": template.Name,
			"hp":   template.HP,
			"ac":   template.AC,
		})
	}
	return summary
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			trimmed = append(trimmed, cleaned)
		}
	}
	return trimmed
}

func cloneStringMap(input map[string]string) map[string]string {
	cloned := map[string]string{}
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func extractFieldErrors(err error, markdownField string) map[string]string {
	var validationError ValidationError
	if errors.As(err, &validationError) {
		return cloneStringMap(validationError.Fields)
	}
	return map[string]string{markdownField: err.Error()}
}

func isDowned(combatant Combatant) bool {
	return combatant.CurrentHP != nil && *combatant.CurrentHP <= 0
}

func intPtr(value int) *int {
	return &value
}
