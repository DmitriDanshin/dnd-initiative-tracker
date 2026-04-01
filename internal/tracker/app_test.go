package tracker

import "testing"

func TestSaveTemplateMethods(t *testing.T) {
	app, err := NewTrackerApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := app.SaveNPCTemplate(SaveNPCTemplateInput{
		Name:            "Ogre",
		AC:              intPtr(11),
		HP:              intPtr(59),
		Dex:             intPtr(8),
		InitiativeBonus: intPtr(-1),
		TagsRaw:         "giant, brute",
		Notes:           "Big club.",
	}); err != nil {
		t.Fatal(err)
	}

	state, err := app.GetState()
	if err != nil {
		t.Fatal(err)
	}
	if state["mode"] != "home" {
		t.Fatalf("expected home mode, got %#v", state["mode"])
	}
	if state["message"] != "Saved NPC template Ogre." {
		t.Fatalf("unexpected message: %#v", state["message"])
	}
}

func TestStateIncludesTemplatesOnHomeScreen(t *testing.T) {
	app, err := NewTrackerApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := app.SaveNPCTemplate(SaveNPCTemplateInput{
		Name: "Goblin",
		AC:   intPtr(15),
		HP:   intPtr(7),
		Dex:  intPtr(14),
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.SavePlayerTemplate(SavePlayerTemplateInput{
		Name:            "Aramil",
		InitiativeBonus: intPtr(3),
	}); err != nil {
		t.Fatal(err)
	}

	state, err := app.GetState()
	if err != nil {
		t.Fatal(err)
	}

	npcTemplates := state["npc_templates"].([]map[string]any)
	if len(npcTemplates) != 1 || npcTemplates[0]["name"] != "Goblin" || npcTemplates[0]["hp"] != 7 || npcTemplates[0]["ac"] != 15 {
		t.Fatalf("unexpected npc templates: %#v", npcTemplates)
	}

	playerTemplates := state["player_templates"].([]string)
	if len(playerTemplates) != 1 || playerTemplates[0] != "Aramil" {
		t.Fatalf("unexpected player templates: %#v", playerTemplates)
	}
}

func TestSaveNPCTemplateExposesFieldErrors(t *testing.T) {
	app, err := NewTrackerApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := app.SaveNPCTemplate(SaveNPCTemplateInput{
		Name: "",
		AC:   intPtr(15),
		HP:   intPtr(7),
		Dex:  intPtr(14),
	}); err != nil {
		t.Fatal(err)
	}

	state, err := app.GetState()
	if err != nil {
		t.Fatal(err)
	}

	if state["message"] != "NPC template validation failed." {
		t.Fatalf("unexpected message: %#v", state["message"])
	}

	fieldErrors := state["field_errors"].(map[string]string)
	if fieldErrors["name"] != "Value error, NPC name must not be empty." {
		t.Fatalf("unexpected field errors: %#v", fieldErrors)
	}
}

func TestResumeEncounterOpensSpecificFight(t *testing.T) {
	app, err := NewTrackerApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	app.StartNewEncounter()
	if err := app.SavePlayerTemplate(SavePlayerTemplateInput{
		Name:            "Aramil",
		InitiativeBonus: intPtr(3),
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.AddPlayer("Aramil", nil); err != nil {
		t.Fatal(err)
	}
	if err := app.StartEncounter([]*int{intPtr(12)}); err != nil {
		t.Fatal(err)
	}
	encounterID := app.currentEncounter.EncounterID

	if err := app.GoHome(); err != nil {
		t.Fatal(err)
	}
	ok, err := app.ResumeEncounter(encounterID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected encounter to resume")
	}

	state, err := app.GetState()
	if err != nil {
		t.Fatal(err)
	}

	if state["mode"] != "combat" {
		t.Fatalf("expected combat mode, got %#v", state["mode"])
	}
	encounter := state["encounter"].(EncounterState)
	if encounter.EncounterID != encounterID {
		t.Fatalf("expected encounter %s, got %s", encounterID, encounter.EncounterID)
	}
}
