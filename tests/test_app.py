import pytest
from pathlib import Path
import uuid

from dnd_initiative_tracker.app import DndInitiativeTrackerApp


class TestApp:
    @pytest.fixture
    def tracker(self) -> DndInitiativeTrackerApp:
        root_path = Path.cwd() / ".tmp_test_runs" / uuid.uuid4().hex
        yield DndInitiativeTrackerApp(root_path)

    @pytest.mark.parametrize(("method_name", "payload", "expected_name"), [
        (
            "save_npc_template",
            {
                "name": "Ogre",
                "ac": 11,
                "hp": 59,
                "dex": 8,
                "initiative_bonus": -1,
                "tags_raw": "giant, brute",
                "notes": "Big club.",
            },
            "Ogre",
        ),
        (
            "save_player_template",
            {
                "markdown": """---
name: Lia
ac: 14
max_hp: 22
dex: 18
initiative_bonus: 4
notes: Rogue scout.
---
Prefers stealth.
""",
            },
            "Lia",
        ),
    ])
    def test_save_template_methods(self, tracker: DndInitiativeTrackerApp, method_name: str, payload: dict, expected_name: str):
        getattr(tracker, method_name)(**payload)

        assert expected_name in tracker.message
        state = tracker.get_state()
        assert state["mode"] == "home"

    def test_state_includes_templates_on_home_screen(self, tracker: DndInitiativeTrackerApp):
        tracker.save_npc_template(
            name="Goblin",
            ac=15,
            hp=7,
            dex=14,
        )
        tracker.save_player_template(
            name="Aramil",
            initiative_bonus=3,
        )

        state = tracker.get_state()

        assert state["npc_templates"] == [{"name": "Goblin", "hp": 7, "ac": 15}]
        assert state["player_templates"] == ["Aramil"]

    def test_save_npc_template_exposes_field_errors(self, tracker: DndInitiativeTrackerApp):
        tracker.save_npc_template(
            name="",
            ac=15,
            hp=7,
            dex=14,
        )

        state = tracker.get_state()

        assert state["message"] == "NPC template validation failed."
        assert state["field_errors"] == {"name": "Value error, NPC name must not be empty."}

    def test_resume_encounter_opens_specific_fight(self, tracker: DndInitiativeTrackerApp):
        tracker.start_new_encounter()
        tracker.save_player_template(
            name="Aramil",
            initiative_bonus=3,
        )
        tracker.add_player("Aramil", None)
        tracker.start_encounter(rolls=[12])
        encounter_id = tracker.current_encounter.encounter_id

        tracker.go_home()
        tracker.resume_encounter(encounter_id)

        state = tracker.get_state()

        assert state["mode"] == "combat"
        assert state["encounter"]["encounter_id"] == encounter_id

    def test_resume_encounter_missing_fight_returns_home(self, tracker: DndInitiativeTrackerApp):
        tracker.resume_encounter("missing-fight")

        state = tracker.get_state()

        assert state["mode"] == "home"
        assert state["message"] == "Encounter 'missing-fight' not found."
