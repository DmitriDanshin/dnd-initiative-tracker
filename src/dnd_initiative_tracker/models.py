from __future__ import annotations

import uuid
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


class NpcTemplate(BaseModel):
    model_config = ConfigDict(extra="forbid")

    name: str
    ac: int
    hp: int
    dex: int
    initiative_bonus: int | None = None
    tags: list[str] = Field(default_factory=list)
    notes: str = ""

    @field_validator("name")
    @classmethod
    def validate_name(cls, value: str) -> str:
        cleaned_value = value.strip()
        if not cleaned_value:
            raise ValueError("NPC name must not be empty.")
        return cleaned_value


class PlayerTemplate(BaseModel):
    model_config = ConfigDict(extra="forbid")

    name: str
    ac: int | None = None
    max_hp: int | None = None
    current_hp: int | None = None
    dex: int | None = None
    initiative_bonus: int | None = None
    notes: str = ""

    @field_validator("name")
    @classmethod
    def validate_name(cls, value: str) -> str:
        cleaned_value = value.strip()
        if not cleaned_value:
            raise ValueError("Player name must not be empty.")
        return cleaned_value

    @model_validator(mode="after")
    def default_current_hp(self) -> "PlayerTemplate":
        if self.current_hp is None and self.max_hp is not None:
            self.current_hp = self.max_hp
        return self


class Combatant(BaseModel):
    model_config = ConfigDict(extra="forbid")

    kind: Literal["npc", "player"]
    source_name: str
    display_name: str
    token_label: str | None = None
    ac: int | None = None
    max_hp: int | None = None
    current_hp: int | None = None
    dex: int | None = None
    initiative_bonus: int | None = None
    initiative_roll: int | None = None
    initiative_total: int | None = None
    notes: str = ""
    statuses: list[str] = Field(default_factory=list)
    sort_index: int = 0

    @field_validator("token_label")
    @classmethod
    def normalize_token_label(cls, value: str | None) -> str | None:
        if value is None:
            return None
        cleaned_value = value.strip().upper()
        return cleaned_value or None

    @model_validator(mode="after")
    def default_current_hp(self) -> "Combatant":
        if self.current_hp is None and self.max_hp is not None:
            self.current_hp = self.max_hp
        return self


class EncounterState(BaseModel):
    model_config = ConfigDict(extra="forbid")

    encounter_id: str = Field(default_factory=lambda: uuid.uuid4().hex)
    encounter_name: str
    round: int = 1
    active_index: int = 0
    combatants: list[Combatant] = Field(default_factory=list)

    @model_validator(mode="after")
    def validate_token_labels(self) -> "EncounterState":
        non_empty_labels = [
            combatant.token_label
            for combatant in self.combatants
            if combatant.token_label is not None
        ]
        if len(non_empty_labels) != len(set(non_empty_labels)):
            raise ValueError("Token labels must be unique inside one encounter.")
        return self
