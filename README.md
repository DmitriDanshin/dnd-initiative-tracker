# DnD Initiative Tracker

Local keyboard-only TUI application for a DM who wants to track initiative, NPC stats, player initiative, token labels like `B1/B2/B3`, and the current state of a fight.

## Features

- NPC templates in `npc/*.md`
- Player templates in `players/*.md`
- Optional player combat stats
- Automatic NPC initiative based on DnD Dexterity rules
- Step-by-step player initiative entry before combat starts
- Saved encounters in `saves/*.md`
- Primitive fullscreen TUI built with `prompt_toolkit`

## Markdown formats

### `npc/*.md`

```md
---
name: Goblin
ac: 15
hp: 7
dex: 14
initiative_bonus: null
tags:
  - goblinoid
notes: Nimble Escape
---
```

### `players/*.md`

```md
---
name: Aramil
initiative_bonus: 3
---
```

Detailed player records are also supported:

```md
---
name: Brakka
ac: 17
max_hp: 24
current_hp: 24
dex: 10
initiative_bonus: null
notes: Shield user
---
```

### `saves/*.md`

The application writes encounter state there automatically. Token labels like `B1` belong to the saved encounter, not to the NPC template.

## Run

Install dependencies:

```powershell
uv sync
```

Run the TUI:

```powershell
uv run dnd-initiative-tracker
```

Run tests:

```powershell
uv run pytest
```

## Current workflow

1. Create or edit NPC markdown files in `npc/`
2. Create players in `players/` or add them from the setup screen
3. Start a new encounter
4. Add NPCs with optional token labels like `B1,B2,B3`
5. Add players
6. Roll NPC initiative
7. Enter player initiative one by one
8. Track turns and HP in the combat screen

## Controls

- `n` new encounter from home, `n` next turn in combat
- `r` resume from home, `r` roll NPC initiative in setup
- `j` / `k` move selection
- `a` add NPC in setup
- `p` add player in setup
- `e` rename encounter in setup
- `Enter` start encounter or load save
- `Backspace` remove selected combatant in setup
- `d` apply HP delta in combat
- `s` save encounter in combat
- `q` go back home, or quit from home
