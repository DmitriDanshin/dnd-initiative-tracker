package tracker

import "testing"

func TestCalculateInitiativeModifier(t *testing.T) {
	testCases := []struct {
		score    int
		expected int
	}{
		{score: 8, expected: -1},
		{score: 10, expected: 0},
		{score: 14, expected: 2},
		{score: 18, expected: 4},
	}

	for _, testCase := range testCases {
		if actual := CalculateInitiativeModifier(testCase.score); actual != testCase.expected {
			t.Fatalf("score %d: expected %d, got %d", testCase.score, testCase.expected, actual)
		}
	}
}

func TestResolveInitiativeBonus(t *testing.T) {
	dex := 14
	explicit := 5

	testCases := []struct {
		dexterity *int
		explicit  *int
		expected  int
	}{
		{dexterity: &dex, explicit: nil, expected: 2},
		{dexterity: intPtr(10), explicit: &explicit, expected: 5},
		{dexterity: nil, explicit: nil, expected: 0},
	}

	for _, testCase := range testCases {
		if actual := ResolveInitiativeBonus(testCase.dexterity, testCase.explicit); actual != testCase.expected {
			t.Fatalf("expected %d, got %d", testCase.expected, actual)
		}
	}
}

func TestSortCombatantsForInitiative(t *testing.T) {
	combatants := []Combatant{
		{
			Kind:            "npc",
			SourceName:      "Goblin",
			DisplayName:     "Goblin #1",
			InitiativeTotal: intPtr(14),
			InitiativeBonus: intPtr(2),
			SortIndex:       0,
		},
		{
			Kind:            "player",
			SourceName:      "Aramil",
			DisplayName:     "Aramil",
			InitiativeTotal: intPtr(14),
			InitiativeBonus: intPtr(3),
			SortIndex:       1,
		},
		{
			Kind:            "npc",
			SourceName:      "Orc",
			DisplayName:     "Orc #1",
			InitiativeTotal: intPtr(12),
			InitiativeBonus: intPtr(1),
			SortIndex:       2,
		},
	}

	sortedCombatants := SortCombatantsForInitiative(combatants)

	if sortedCombatants[0].DisplayName != "Aramil" || sortedCombatants[1].DisplayName != "Goblin #1" || sortedCombatants[2].DisplayName != "Orc #1" {
		t.Fatalf("unexpected order: %#v", []string{
			sortedCombatants[0].DisplayName,
			sortedCombatants[1].DisplayName,
			sortedCombatants[2].DisplayName,
		})
	}
}
