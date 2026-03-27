import pytest

from dnd_initiative_tracker.initiative import calculate_initiative_modifier, resolve_initiative_bonus, sort_combatants_for_initiative
from dnd_initiative_tracker.models import Combatant


class TestInitiative:
    @pytest.mark.parametrize(("dexterity_score", "expected_modifier"), [
        (8, -1),
        (10, 0),
        (14, 2),
        (18, 4),
    ])
    def test_calculate_initiative_modifier(self, dexterity_score, expected_modifier):
        assert calculate_initiative_modifier(dexterity_score) == expected_modifier

    @pytest.mark.parametrize(("dexterity_score", "explicit_bonus", "expected_bonus"), [
        (14, None, 2),
        (10, 5, 5),
        (None, None, 0),
    ])
    def test_resolve_initiative_bonus(self, dexterity_score, explicit_bonus, expected_bonus):
        assert resolve_initiative_bonus(dexterity_score, explicit_bonus) == expected_bonus

    def test_sort_combatants_for_initiative(self):
        combatants = [
            Combatant(
                kind="npc",
                source_name="Goblin",
                display_name="Goblin #1",
                initiative_total=14,
                initiative_bonus=2,
                sort_index=0,
            ),
            Combatant(
                kind="player",
                source_name="Aramil",
                display_name="Aramil",
                initiative_total=14,
                initiative_bonus=3,
                sort_index=1,
            ),
            Combatant(
                kind="npc",
                source_name="Orc",
                display_name="Orc #1",
                initiative_total=12,
                initiative_bonus=1,
                sort_index=2,
            ),
        ]

        sorted_combatants = sort_combatants_for_initiative(combatants)

        assert [combatant.display_name for combatant in sorted_combatants] == [
            "Aramil",
            "Goblin #1",
            "Orc #1",
        ]
