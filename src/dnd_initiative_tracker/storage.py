from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml

from dnd_initiative_tracker.models import Combatant, EncounterState, NpcTemplate, PlayerTemplate


class MarkdownRepository:
    def __init__(self, root_path: Path) -> None:
        self.root_path = root_path
        self.npc_path = root_path / "npc"
        self.players_path = root_path / "players"
        self.saves_path = root_path / "saves"
        self.ensure_directories()

    def ensure_directories(self) -> None:
        for path in (self.npc_path, self.players_path, self.saves_path):
            path.mkdir(parents=True, exist_ok=True)

    def list_npc_templates(self) -> list[NpcTemplate]:
        return sorted(
            [self.load_npc_template(path) for path in self.npc_path.glob("*.md")],
            key=lambda template: template.name.lower(),
        )

    def list_player_templates(self) -> list[PlayerTemplate]:
        return sorted(
            [self.load_player_template(path) for path in self.players_path.glob("*.md")],
            key=lambda template: template.name.lower(),
        )

    def list_save_names(self) -> list[str]:
        return sorted(path.stem for path in self.saves_path.glob("*.md"))

    def list_encounters(self) -> list[dict[str, str]]:
        encounters = []
        for path in self.saves_path.glob("*.md"):
            try:
                front_matter, _ = self._read_markdown(path)
                encounters.append({
                    "encounter_id": front_matter.get("encounter_id", path.stem),
                    "encounter_name": front_matter.get("encounter_name", path.stem),
                    "round": front_matter.get("round", 1),
                })
            except (ValueError, KeyError):
                continue
        encounters.sort(key=lambda e: e["encounter_name"].lower())
        return encounters

    def load_npc_template(self, path: Path) -> NpcTemplate:
        front_matter, body = self._read_markdown(path)
        front_matter["notes"] = self._combine_notes(front_matter.get("notes", ""), body)
        return NpcTemplate.model_validate(front_matter)

    def load_player_template(self, path: Path) -> PlayerTemplate:
        front_matter, body = self._read_markdown(path)
        front_matter["notes"] = self._combine_notes(front_matter.get("notes", ""), body)
        return PlayerTemplate.model_validate(front_matter)

    def save_player_template(self, template: PlayerTemplate) -> Path:
        path = self.players_path / f"{self.slugify(template.name)}.md"
        self._write_markdown(path, template.model_dump(), "")
        return path

    def save_npc_template(self, template: NpcTemplate) -> Path:
        path = self.npc_path / f"{self.slugify(template.name)}.md"
        self._write_markdown(path, template.model_dump(), "")
        return path

    def save_encounter(self, encounter_state: EncounterState) -> Path:
        path = self.saves_path / f"{encounter_state.encounter_id}.md"
        self._write_markdown(path, encounter_state.model_dump(), "")
        return path

    def load_encounter(self, save_name: str) -> EncounterState:
        path = self.saves_path / f"{save_name}.md"
        front_matter, _ = self._read_markdown(path)
        combatants = [
            Combatant.model_validate(combatant_payload)
            for combatant_payload in front_matter.get("combatants", [])
        ]
        front_matter["combatants"] = combatants
        if "encounter_id" not in front_matter:
            front_matter["encounter_id"] = save_name
        return EncounterState.model_validate(front_matter)

    def load_npc_template_by_name(self, name: str) -> NpcTemplate | None:
        for template in self.list_npc_templates():
            if template.name.casefold() == name.casefold():
                return template
        return None

    def load_player_template_by_name(self, name: str) -> PlayerTemplate | None:
        for template in self.list_player_templates():
            if template.name.casefold() == name.casefold():
                return template
        return None

    def delete_npc_template(self, name: str) -> bool:
        path = self.npc_path / f"{self.slugify(name)}.md"
        if path.exists():
            path.unlink()
            return True
        return False

    def delete_player_template(self, name: str) -> bool:
        path = self.players_path / f"{self.slugify(name)}.md"
        if path.exists():
            path.unlink()
            return True
        return False

    def parse_npc_template_markdown(self, text: str) -> NpcTemplate:
        front_matter, body = self._parse_markdown_text(text)
        front_matter["notes"] = self._combine_notes(front_matter.get("notes", ""), body)
        return NpcTemplate.model_validate(front_matter)

    def parse_player_template_markdown(self, text: str) -> PlayerTemplate:
        front_matter, body = self._parse_markdown_text(text)
        front_matter["notes"] = self._combine_notes(front_matter.get("notes", ""), body)
        return PlayerTemplate.model_validate(front_matter)

    def _read_markdown(self, path: Path) -> tuple[dict[str, Any], str]:
        text = path.read_text(encoding="utf-8")
        return self._parse_markdown_text(text, source=str(path))

    def _write_markdown(self, path: Path, front_matter: dict[str, Any], body: str) -> None:
        serialized_front_matter = yaml.safe_dump(
            front_matter,
            allow_unicode=False,
            sort_keys=False,
        ).strip()
        text = f"---\n{serialized_front_matter}\n---\n"
        if body.strip():
            text += f"\n{body.strip()}\n"
        path.write_text(text, encoding="utf-8")

    @staticmethod
    def _combine_notes(notes: str, body: str) -> str:
        notes_parts = [part.strip() for part in (notes, body) if part.strip()]
        return "\n\n".join(notes_parts)

    @staticmethod
    def _parse_markdown_text(text: str, source: str = "markdown input") -> tuple[dict[str, Any], str]:
        if not text.startswith("---"):
            raise ValueError(f"{source} must start with YAML front matter.")
        parts = text.split("---", 2)
        if len(parts) != 3:
            raise ValueError(f"{source} has invalid YAML front matter.")
        _, raw_front_matter, remainder = parts
        front_matter = yaml.safe_load(raw_front_matter) or {}
        body = remainder.lstrip("\r\n")
        return front_matter, body.rstrip()

    @staticmethod
    def slugify(value: str) -> str:
        allowed_characters = []
        for character in value.lower().strip():
            if character.isalnum():
                allowed_characters.append(character)
            elif character in {" ", "-", "_"}:
                allowed_characters.append("-")
        slug = "".join(allowed_characters).strip("-")
        while "--" in slug:
            slug = slug.replace("--", "-")
        return slug or "encounter"
