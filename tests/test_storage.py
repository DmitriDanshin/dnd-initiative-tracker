import pytest
from dnd_initiative_tracker.models import Combatant, EncounterState, PlayerTemplate
from dnd_initiative_tracker.storage import MarkdownRepository


class TestStorage:
    @pytest.fixture
    def repository(self, tmp_path) -> MarkdownRepository:
        root_path = tmp_path / "repository"
        return MarkdownRepository(root_path)

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
