from __future__ import annotations

import importlib.resources
from pathlib import Path

from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse, JSONResponse

from dnd_initiative_tracker.initiative import (
    assign_npc_initiative,
    assign_player_initiative,
    sort_combatants_for_initiative,
)
from dnd_initiative_tracker.models import Combatant, EncounterState, NpcTemplate, PlayerTemplate
from dnd_initiative_tracker.storage import MarkdownRepository

HTML_PAGE = importlib.resources.files("dnd_initiative_tracker").joinpath("index.html").read_text(encoding="utf-8")


class DndInitiativeTrackerApp:
    def __init__(self, root_path: Path) -> None:
        self.repository = MarkdownRepository(root_path)
        self.mode = "home"
        self.message = ""
        self.selected_index = 0
        self.setup_encounter_name = "New Encounter"
        self.setup_combatants: list[Combatant] = []
        self.current_encounter: EncounterState | None = None
        self._restore_last_encounter()

    def get_state(self) -> dict:
        base: dict = {
            "mode": self.mode,
            "message": self.message,
            "selected_index": self.selected_index,
        }
        if self.mode == "home":
            base["encounters"] = self.repository.list_encounters()
        elif self.mode == "setup":
            base["setup_encounter_name"] = self.setup_encounter_name
            base["setup_combatants"] = [c.model_dump() for c in self.setup_combatants]
            base["npc_templates"] = [
                {"name": t.name, "hp": t.hp, "ac": t.ac} for t in self.repository.list_npc_templates()
            ]
            base["player_templates"] = [t.name for t in self.repository.list_player_templates()]
        elif self.mode == "combat" and self.current_encounter is not None:
            base["encounter"] = self.current_encounter.model_dump()
        return base

    def go_home(self) -> None:
        if self.current_encounter is not None:
            self._autosave()
        self._clear_last_encounter()
        self.current_encounter = None
        self.mode = "home"
        self.selected_index = 0
        self.message = ""

    def start_new_encounter(self) -> None:
        self.mode = "setup"
        self.setup_encounter_name = "New Encounter"
        self.setup_combatants = []
        self.selected_index = 0
        self.message = ""

    def resume_encounter(self, encounter_id: str) -> None:
        self.current_encounter = self.repository.load_encounter(encounter_id)
        self.mode = "combat"
        self.selected_index = self.current_encounter.active_index
        active = self.current_encounter.combatants[self.current_encounter.active_index]
        self.message = f"Resumed. Round {self.current_encounter.round}, active: {active.display_name}."
        self._save_last_encounter_id(encounter_id)

    def set_encounter_name(self, name: str) -> None:
        self.setup_encounter_name = name
        self.message = f"Encounter name set to {name}."

    def add_npc(
        self,
        name: str,
        count: int,
        labels_raw: str,
        hp_override: int | None = None,
        ac_override: int | None = None,
    ) -> None:
        npc_template = self.repository.load_npc_template_by_name(name)
        if npc_template is None:
            self.message = f"NPC '{name}' not found."
            return
        if count < 1:
            self.message = "NPC count must be at least 1."
            return
        token_labels = [label.strip() for label in labels_raw.split(",") if label.strip()] if labels_raw else []
        if token_labels and len(token_labels) != count:
            self.message = "Token label count must match NPC count."
            return
        existing_count = sum(
            1 for c in self.setup_combatants if c.kind == "npc" and c.source_name == npc_template.name
        )
        effective_hp = hp_override if hp_override is not None else npc_template.hp
        effective_ac = ac_override if ac_override is not None else npc_template.ac
        for index in range(count):
            combatant = self._combatant_from_npc_template(
                template=npc_template,
                display_name=f"{npc_template.name} #{existing_count + index + 1}",
                token_label=token_labels[index] if index < len(token_labels) else None,
                hp_override=effective_hp,
                ac_override=effective_ac,
            )
            self.setup_combatants.append(combatant)
        self.selected_index = len(self.setup_combatants) - 1
        self.message = f"Added {count} {npc_template.name}."

    def add_player(self, name: str, initiative_bonus: int | None) -> None:
        if not name:
            self.message = "Player name is required."
            return
        already_added = any(
            c.kind == "player" and c.source_name.casefold() == name.casefold()
            for c in self.setup_combatants
        )
        if already_added:
            self.message = f"Player '{name}' is already in the roster."
            return
        player_template = self.repository.load_player_template_by_name(name)
        if player_template is None:
            player_template = PlayerTemplate(name=name, initiative_bonus=initiative_bonus)
            self.repository.save_player_template(player_template)
        combatant = Combatant(
            kind="player",
            source_name=player_template.name,
            display_name=player_template.name,
            ac=player_template.ac,
            max_hp=player_template.max_hp,
            current_hp=player_template.current_hp,
            dex=player_template.dex,
            initiative_bonus=player_template.initiative_bonus,
            notes=player_template.notes,
            sort_index=len(self.setup_combatants),
        )
        self.setup_combatants.append(combatant)
        self.selected_index = len(self.setup_combatants) - 1
        self.message = f"Added player {player_template.name}."

    def roll_npc_initiative(self) -> None:
        npc_count = 0
        for combatant in self.setup_combatants:
            if combatant.kind == "npc":
                assign_npc_initiative(combatant)
                npc_count += 1
        self.message = f"Rolled initiative for {npc_count} NPC(s)."

    def remove_combatant(self, index: int) -> None:
        if 0 <= index < len(self.setup_combatants):
            removed = self.setup_combatants.pop(index)
            self.selected_index = max(0, min(self.selected_index, len(self.setup_combatants) - 1))
            self.message = f"Removed {removed.display_name}."

    def get_players_needing_rolls(self) -> list[dict] | None:
        if not self.setup_combatants:
            self.message = "Add at least one combatant first."
            return None
        players = [
            {"name": c.display_name, "bonus": c.initiative_bonus or 0, "index": i}
            for i, c in enumerate(self.setup_combatants)
            if c.kind == "player" and c.initiative_total is None
        ]
        return players if players else None

    def start_encounter(self, rolls: list[int | None] | None = None) -> None:
        if not self.setup_combatants:
            self.message = "Add at least one combatant first."
            return
        for combatant in self.setup_combatants:
            if combatant.kind == "npc" and combatant.initiative_total is None:
                assign_npc_initiative(combatant)
        encounter_state = EncounterState(
            encounter_name=self.setup_encounter_name,
            combatants=self.setup_combatants,
        )
        if rolls is not None:
            player_indices = [
                i for i, c in enumerate(encounter_state.combatants)
                if c.kind == "player" and c.initiative_total is None
            ]
            for pi, roll in zip(player_indices, rolls):
                if roll is not None:
                    assign_player_initiative(encounter_state.combatants[pi], roll)
        encounter_state.combatants = sort_combatants_for_initiative(encounter_state.combatants)
        self.current_encounter = encounter_state
        self.mode = "combat"
        self.selected_index = encounter_state.active_index
        active = encounter_state.combatants[encounter_state.active_index]
        self.message = f"Combat started. Round {encounter_state.round}, active: {active.display_name}."
        self._autosave()

    def advance_turn(self) -> None:
        if self.current_encounter is None or not self.current_encounter.combatants:
            return
        combatants = self.current_encounter.combatants
        total = len(combatants)
        steps = 0
        while steps < total:
            self.current_encounter.active_index += 1
            if self.current_encounter.active_index >= total:
                self.current_encounter.active_index = 0
                self.current_encounter.round += 1
            steps += 1
            active = combatants[self.current_encounter.active_index]
            if not self._is_downed(active):
                break
        self.selected_index = self.current_encounter.active_index
        active = combatants[self.current_encounter.active_index]
        self.message = f"Round {self.current_encounter.round}, active: {active.display_name}."
        self._autosave()

    @staticmethod
    def _is_downed(combatant: Combatant) -> bool:
        return combatant.current_hp is not None and combatant.current_hp <= 0

    def apply_hp_delta(self, index: int, delta: int) -> None:
        if self.current_encounter is None or not self.current_encounter.combatants:
            return
        combatant = self.current_encounter.combatants[index]
        if combatant.current_hp is None:
            self.message = f"{combatant.display_name} does not track HP."
            return
        combatant.current_hp += delta
        combatant.current_hp = max(combatant.current_hp, 0)
        if combatant.max_hp is not None:
            combatant.current_hp = min(combatant.current_hp, combatant.max_hp)
        self.message = f"{combatant.display_name} HP is now {combatant.current_hp}/{combatant.max_hp or '-'}."
        self._autosave()

    def save_encounter(self) -> None:
        if self.current_encounter is not None:
            self.repository.save_encounter(self.current_encounter)
            self.message = "Encounter saved."

    def _autosave(self) -> None:
        if self.current_encounter is not None:
            self.repository.save_encounter(self.current_encounter)
            self._save_last_encounter_id(self.current_encounter.encounter_id)

    def _restore_last_encounter(self) -> None:
        last_file = self.repository.saves_path / "_last.txt"
        if not last_file.exists():
            return
        encounter_id = last_file.read_text(encoding="utf-8").strip()
        save_path = self.repository.saves_path / f"{encounter_id}.md"
        if not encounter_id or not save_path.exists():
            return
        self.current_encounter = self.repository.load_encounter(encounter_id)
        self.mode = "combat"
        self.selected_index = self.current_encounter.active_index
        active = self.current_encounter.combatants[self.current_encounter.active_index]
        self.message = f"Restored. Round {self.current_encounter.round}, active: {active.display_name}."

    def _save_last_encounter_id(self, encounter_id: str) -> None:
        last_file = self.repository.saves_path / "_last.txt"
        last_file.write_text(encounter_id, encoding="utf-8")

    def _clear_last_encounter(self) -> None:
        last_file = self.repository.saves_path / "_last.txt"
        if last_file.exists():
            last_file.unlink()

    def _combatant_from_npc_template(
        self,
        template: NpcTemplate,
        display_name: str,
        token_label: str | None,
        hp_override: int | None = None,
        ac_override: int | None = None,
    ) -> Combatant:
        max_hp = hp_override if hp_override is not None else template.hp
        max_hp = max(0, max_hp)
        current_hp = max_hp
        ac = ac_override if ac_override is not None else template.ac
        return Combatant(
            kind="npc",
            source_name=template.name,
            display_name=display_name,
            token_label=token_label,
            ac=ac,
            max_hp=max_hp,
            current_hp=current_hp,
            dex=template.dex,
            initiative_bonus=template.initiative_bonus,
            notes=template.notes,
            sort_index=len(self.setup_combatants),
        )


def create_fastapi_app(root_path: Path | None = None) -> FastAPI:
    effective_root_path = root_path or Path.cwd()
    tracker = DndInitiativeTrackerApp(effective_root_path)
    app = FastAPI()

    @app.get("/", response_class=HTMLResponse)
    async def index():
        return HTML_PAGE

    @app.get("/api/state")
    async def get_state():
        return JSONResponse(tracker.get_state())

    @app.post("/api/go-home")
    async def go_home():
        tracker.go_home()
        return JSONResponse(tracker.get_state())

    @app.post("/api/new-encounter")
    async def new_encounter():
        tracker.start_new_encounter()
        return JSONResponse(tracker.get_state())

    @app.post("/api/resume-encounter")
    async def resume_encounter(request: Request):
        body = await request.json()
        tracker.resume_encounter(body["encounter_id"])
        return JSONResponse(tracker.get_state())

    @app.post("/api/set-encounter-name")
    async def set_encounter_name(request: Request):
        body = await request.json()
        tracker.set_encounter_name(body["name"])
        return JSONResponse(tracker.get_state())

    @app.post("/api/add-npc")
    async def add_npc(request: Request):
        body = await request.json()
        tracker.add_npc(
            body["name"],
            body.get("count", 1),
            body.get("labels", ""),
            hp_override=body.get("hp"),
            ac_override=body.get("ac"),
        )
        return JSONResponse(tracker.get_state())

    @app.post("/api/add-player")
    async def add_player(request: Request):
        body = await request.json()
        tracker.add_player(body["name"], body.get("initiative_bonus"))
        return JSONResponse(tracker.get_state())

    @app.post("/api/roll-npc")
    async def roll_npc():
        tracker.roll_npc_initiative()
        return JSONResponse(tracker.get_state())

    @app.post("/api/remove-combatant")
    async def remove_combatant(request: Request):
        body = await request.json()
        tracker.remove_combatant(body["index"])
        return JSONResponse(tracker.get_state())

    @app.post("/api/select")
    async def select(request: Request):
        body = await request.json()
        tracker.selected_index = body["index"]
        return JSONResponse(tracker.get_state())

    @app.post("/api/start-encounter")
    async def start_encounter():
        players = tracker.get_players_needing_rolls()
        if players:
            return JSONResponse({"need_rolls": True, "players": players})
        tracker.start_encounter()
        return JSONResponse(tracker.get_state())

    @app.post("/api/submit-rolls")
    async def submit_rolls(request: Request):
        body = await request.json()
        tracker.start_encounter(rolls=body["rolls"])
        return JSONResponse(tracker.get_state())

    @app.post("/api/next-turn")
    async def next_turn():
        tracker.advance_turn()
        return JSONResponse(tracker.get_state())

    @app.post("/api/hp-delta")
    async def hp_delta(request: Request):
        body = await request.json()
        tracker.apply_hp_delta(body["index"], body["delta"])
        return JSONResponse(tracker.get_state())

    @app.post("/api/save")
    async def save():
        tracker.save_encounter()
        return JSONResponse(tracker.get_state())

    return app
