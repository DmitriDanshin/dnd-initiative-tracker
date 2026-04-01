# DnD Initiative Tracker

Local Go web server for tracking DnD encounters with markdown-backed NPC/player templates and saved combat state.

## Features

- Go HTTP backend with the same home, setup, NPC, player, and combat pages
- NPC templates in `npc/*.md`
- Player templates in `players/*.md`
- Encounter saves in `saves/*.md`
- Autosave and last-encounter restore
- NPC initiative rolls, manual player rolls, turn order, HP tracking, AC display, and token labels

## Project Layout

- `cmd/dnd-initiative-tracker/main.go` starts the web server
- `internal/tracker/` contains domain models, initiative logic, markdown storage, HTTP handlers, tests, and embedded frontend assets
- `npc/`, `players/`, and `saves/` are created automatically in the working directory

## Markdown Formats

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

Encounter state is written there automatically. Token labels belong to the saved encounter, not to the NPC template.

## Run

Install dependencies:

```powershell
go mod tidy
```

Start the web server:

```powershell
go run ./cmd/dnd-initiative-tracker
```

The server listens on `http://127.0.0.1:8000`.

## Test

```powershell
go test ./...
```

## Web Workflow

1. Create NPC and player templates from the UI or by editing markdown files in `npc/` and `players/`
2. Open `http://127.0.0.1:8000`
3. Start a new encounter or resume a saved one
4. Add NPCs with optional token labels like `B1,B2,B3`
5. Add players
6. Roll NPC initiative
7. Enter player initiative rolls
8. Track turns, HP, and save state from the combat screen

## HTTP Endpoints

- `GET /` returns the bundled web UI
- `GET /api/state` returns current application state
- `POST /api/new-encounter` starts setup mode
- `POST /api/resume-encounter` loads a saved encounter
- `POST /api/set-encounter-name` renames the setup encounter
- `POST /api/add-npc` adds NPC combatants
- `POST /api/add-player` adds a player combatant
- `POST /api/save-npc-template` creates or updates an NPC template
- `POST /api/save-player-template` creates or updates a player template
- `POST /api/add-npc-to-combat` adds NPCs during combat
- `POST /api/roll-npc` rolls initiative for NPCs in setup
- `POST /api/remove-combatant` removes a setup combatant
- `POST /api/select` changes the selected combatant
- `POST /api/start-encounter` starts combat or requests player rolls
- `POST /api/submit-rolls` submits player initiative rolls
- `POST /api/next-turn` advances the active combatant
- `POST /api/hp-delta` applies healing or damage
- `POST /api/save` writes the current encounter to `saves/`
