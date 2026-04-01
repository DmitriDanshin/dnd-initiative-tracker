package tracker

import (
	"math/rand/v2"
	"sort"
)

func CalculateInitiativeModifier(dexterityScore int) int {
	return (dexterityScore - 10) / 2
}

func ResolveInitiativeBonus(dexterityScore *int, explicitBonus *int) int {
	if explicitBonus != nil {
		return *explicitBonus
	}
	if dexterityScore == nil {
		return 0
	}
	return CalculateInitiativeModifier(*dexterityScore)
}

func RollD20() int {
	return rand.IntN(20) + 1
}

func AssignNPCInitiative(combatant *Combatant) {
	roll := RollD20()
	bonus := ResolveInitiativeBonus(combatant.Dex, combatant.InitiativeBonus)
	combatant.InitiativeBonus = &bonus
	combatant.InitiativeRoll = &roll
	total := roll + bonus
	combatant.InitiativeTotal = &total
}

func AssignPlayerInitiative(combatant *Combatant, initiativeRoll int) {
	bonus := ResolveInitiativeBonus(combatant.Dex, combatant.InitiativeBonus)
	combatant.InitiativeBonus = &bonus
	combatant.InitiativeRoll = &initiativeRoll
	total := initiativeRoll + bonus
	combatant.InitiativeTotal = &total
}

func SortCombatantsForInitiative(combatants []Combatant) []Combatant {
	sortedCombatants := cloneCombatants(combatants)
	sort.Slice(sortedCombatants, func(left, right int) bool {
		leftCombatant := sortedCombatants[left]
		rightCombatant := sortedCombatants[right]

		leftTotal := -1
		rightTotal := -1
		if leftCombatant.InitiativeTotal != nil {
			leftTotal = *leftCombatant.InitiativeTotal
		}
		if rightCombatant.InitiativeTotal != nil {
			rightTotal = *rightCombatant.InitiativeTotal
		}
		if leftTotal != rightTotal {
			return leftTotal > rightTotal
		}

		leftBonus := -99
		rightBonus := -99
		if leftCombatant.InitiativeBonus != nil {
			leftBonus = *leftCombatant.InitiativeBonus
		}
		if rightCombatant.InitiativeBonus != nil {
			rightBonus = *rightCombatant.InitiativeBonus
		}
		if leftBonus != rightBonus {
			return leftBonus > rightBonus
		}

		return leftCombatant.SortIndex < rightCombatant.SortIndex
	})

	for index := range sortedCombatants {
		sortedCombatants[index].SortIndex = index
	}

	return sortedCombatants
}
