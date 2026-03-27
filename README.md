# DnD Initiative Tracker

Local FastAPI web server for tracking DnD encounters with markdown-backed NPC/player templates and saved combat state.

## Features

- FastAPI backend with a bundled single-page web UI
- NPC templates in `npc/*.md`
- Player templates in `players/*.md`
- Optional player combat stats
- Automatic NPC initiative based on DnD Dexterity rules
- Manual player initiative entry before combat starts
- Encounter saves in `saves/*.md`
- Autosave and last-encounter restore
- HP tracking, AC display, token labels like `B1/B2/B3`, and turn order management

## Project layout

- `src/dnd_initiative_tracker/app.py` contains the FastAPI app and encounter state logic
- `src/dnd_initiative_tracker/index.html` contains the bundled frontend
- `src/dnd_initiative_tracker/storage.py` reads and writes markdown data
- `npc/`, `players/`, and `saves/` are created automatically in the working directory

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

The application writes encounter state there automatically. Token labels belong to the saved encounter, not to the NPC template.

## Run

Install dependencies:

```powershell
uv sync
```

Start the web server:

```powershell
uv run dnd-initiative-tracker
```

The server listens on `http://127.0.0.1:8000`.

Open the UI in a browser:

```text
http://127.0.0.1:8000
```

Run tests:

```powershell
uv run pytest
```

## Web workflow

1. Create or edit NPC markdown files in `npc/`
2. Create players in `players/` or add them from the setup screen
3. Open `http://127.0.0.1:8000`
4. Start a new encounter or resume a saved one
5. Add NPCs with optional token labels like `B1,B2,B3`
6. Add players
7. Roll NPC initiative
8. Enter player initiative rolls
9. Track turns, HP, and save state from the combat screen

## HTTP endpoints

- `GET /` returns the bundled web UI
- `GET /api/state` returns current application state
- `POST /api/new-encounter` starts setup mode
- `POST /api/resume-encounter` loads a saved encounter
- `POST /api/set-encounter-name` renames the setup encounter
- `POST /api/add-npc` adds NPC combatants
- `POST /api/add-player` adds a player combatant
- `POST /api/roll-npc` rolls initiative for NPCs
- `POST /api/remove-combatant` removes a setup combatant
- `POST /api/select` changes selected combatant
- `POST /api/start-encounter` starts combat or requests player rolls
- `POST /api/submit-rolls` submits player initiative rolls
- `POST /api/next-turn` advances the active combatant
- `POST /api/hp-delta` applies healing or damage
- `POST /api/save` writes the current encounter to `saves/`
