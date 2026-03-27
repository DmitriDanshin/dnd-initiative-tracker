from __future__ import annotations

import random

from dnd_initiative_tracker.models import Combatant


def calculate_initiative_modifier(dexterity_score: int) -> int:
    return (dexterity_score - 10) // 2


def resolve_initiative_bonus(
    dexterity_score: int | None,
    explicit_bonus: int | None,
) -> int:
    if explicit_bonus is not None:
        return explicit_bonus
    if dexterity_score is None:
        return 0
    return calculate_initiative_modifier(dexterity_score)


def roll_d20(random_generator: random.Random | None = None) -> int:
    effective_random_generator = random_generator or random.Random()
    return effective_random_generator.randint(1, 20)


def assign_npc_initiative(
    combatant: Combatant,
    random_generator: random.Random | None = None,
) -> Combatant:
    initiative_roll = roll_d20(random_generator)
    initiative_bonus = resolve_initiative_bonus(
        dexterity_score=combatant.dex,
        explicit_bonus=combatant.initiative_bonus,
    )
    combatant.initiative_bonus = initiative_bonus
    combatant.initiative_roll = initiative_roll
    combatant.initiative_total = initiative_roll + initiative_bonus
    return combatant


def assign_player_initiative(combatant: Combatant, initiative_roll: int) -> Combatant:
    initiative_bonus = resolve_initiative_bonus(
        dexterity_score=combatant.dex,
        explicit_bonus=combatant.initiative_bonus,
    )
    combatant.initiative_bonus = initiative_bonus
    combatant.initiative_roll = initiative_roll
    combatant.initiative_total = initiative_roll + initiative_bonus
    return combatant


def sort_combatants_for_initiative(combatants: list[Combatant]) -> list[Combatant]:
    sorted_combatants = sorted(
        combatants,
        key=lambda combatant: (
            combatant.initiative_total if combatant.initiative_total is not None else -1,
            combatant.initiative_bonus if combatant.initiative_bonus is not None else -99,
            -combatant.sort_index,
        ),
        reverse=True,
    )
    for sort_index, combatant in enumerate(sorted_combatants):
        combatant.sort_index = sort_index
    return sorted_combatants
