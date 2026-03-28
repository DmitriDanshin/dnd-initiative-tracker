import pytest
from pathlib import Path
import uuid

from dnd_initiative_tracker.models import Combatant, EncounterState, PlayerTemplate
from dnd_initiative_tracker.storage import MarkdownRepository


class TestStorage:
    @pytest.fixture
    def repository(self) -> MarkdownRepository:
        root_path = Path.cwd() / ".tmp_test_runs" / uuid.uuid4().hex / "repository"
        yield MarkdownRepository(root_path)

    def test_save_and_load_player_template(self, repository: MarkdownRepository):
        player_template = PlayerTemplate(name="Lia", initiative_bonus=4)

        repository.save_player_template(player_template)

        loaded_player = repository.load_player_template_by_name("Lia")

        assert loaded_player is not None
        assert loaded_player.name == "Lia"
        assert loaded_player.initiative_bonus == 4

    def test_save_and_load_encounter(self, repository: MarkdownRepository):
        encounter_state = EncounterState(
            encounter_name="Bridge Fight",
            combatants=[
                Combatant(
                    kind="npc",
                    source_name="Goblin",
                    display_name="Goblin #1",
                    token_label="B1",
                    max_hp=7,
                    current_hp=5,
                    initiative_total=16,
                )
            ],
        )

        repository.save_encounter(encounter_state)

        loaded_encounter = repository.load_encounter(encounter_state.encounter_id)

        assert loaded_encounter.encounter_name == "Bridge Fight"
        assert loaded_encounter.combatants[0].display_name == "Goblin #1"
        assert loaded_encounter.combatants[0].token_label == "B1"

    def test_parse_npc_template_markdown(self, repository: MarkdownRepository):
        npc_template = repository.parse_npc_template_markdown(
            """---
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
"""
        )

        assert npc_template.name == "Goblin Boss"
        assert npc_template.ac == 17
        assert npc_template.hp == 45
        assert npc_template.dex == 14
        assert npc_template.initiative_bonus == 2
        assert npc_template.tags == ["goblinoid"]
        assert npc_template.notes == "Leads the ambush.\n\nCarries a horn."

    def test_parse_player_template_markdown(self, repository: MarkdownRepository):
        player_template = repository.parse_player_template_markdown(
            """---
name: Aramil
ac: 15
max_hp: 28
dex: 16
initiative_bonus: 3
notes: Keeps Bless ready.
---
Prefers longbow.
"""
        )

        assert player_template.name == "Aramil"
        assert player_template.ac == 15
        assert player_template.max_hp == 28
        assert player_template.current_hp == 28
        assert player_template.dex == 16
        assert player_template.initiative_bonus == 3
        assert player_template.notes == "Keeps Bless ready.\n\nPrefers longbow."
