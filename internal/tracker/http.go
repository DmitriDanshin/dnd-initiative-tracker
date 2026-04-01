package tracker

import (
	"embed"
	"encoding/json"
	"net/http"
	"net/url"
)

//go:embed web/*
var webAssets embed.FS

type Server struct {
	tracker *TrackerApp
}

func NewHandler(rootPath string) (http.Handler, error) {
	app, err := NewTrackerApp(rootPath)
	if err != nil {
		return nil, err
	}
	return NewHandlerWithTracker(app), nil
}

func NewHandlerWithTracker(app *TrackerApp) http.Handler {
	server := &Server{tracker: app}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", server.handleIndex)
	mux.HandleFunc("GET /encounter/new", server.handleEncounterNewPage)
	mux.HandleFunc("GET /encounter/{encounter_id}", server.handleEncounterPage)
	mux.HandleFunc("GET /saves", server.handleSavesPage)
	mux.HandleFunc("GET /npcs", server.handleNPCsPage)
	mux.HandleFunc("GET /players", server.handlePlayersPage)
	mux.HandleFunc("GET /assets/app.js", server.handleAppJS)
	mux.HandleFunc("GET /assets/styles.css", server.handleStylesCSS)

	mux.HandleFunc("GET /api/state", server.handleState)
	mux.HandleFunc("POST /api/go-home", server.handleGoHome)
	mux.HandleFunc("POST /api/new-encounter", server.handleNewEncounter)
	mux.HandleFunc("POST /api/resume-encounter", server.handleResumeEncounter)
	mux.HandleFunc("POST /api/set-encounter-name", server.handleSetEncounterName)
	mux.HandleFunc("POST /api/add-npc", server.handleAddNPC)
	mux.HandleFunc("POST /api/add-player", server.handleAddPlayer)
	mux.HandleFunc("GET /api/encounters", server.handleListEncounters)
	mux.HandleFunc("POST /api/rename-encounter", server.handleRenameEncounter)
	mux.HandleFunc("POST /api/delete-encounter", server.handleDeleteEncounter)
	mux.HandleFunc("GET /api/npc-templates", server.handleListNPCTemplates)
	mux.HandleFunc("GET /api/npc-templates/{name}", server.handleGetNPCTemplate)
	mux.HandleFunc("DELETE /api/npc-templates/{name}", server.handleDeleteNPCTemplate)
	mux.HandleFunc("GET /api/player-templates", server.handleListPlayerTemplates)
	mux.HandleFunc("GET /api/player-templates/{name}", server.handleGetPlayerTemplate)
	mux.HandleFunc("DELETE /api/player-templates/{name}", server.handleDeletePlayerTemplate)
	mux.HandleFunc("POST /api/save-npc-template", server.handleSaveNPCTemplate)
	mux.HandleFunc("POST /api/save-player-template", server.handleSavePlayerTemplate)
	mux.HandleFunc("POST /api/add-npc-to-combat", server.handleAddNPCToCombat)
	mux.HandleFunc("POST /api/roll-npc", server.handleRollNPC)
	mux.HandleFunc("POST /api/remove-combatant", server.handleRemoveCombatant)
	mux.HandleFunc("POST /api/select", server.handleSelect)
	mux.HandleFunc("POST /api/start-encounter", server.handleStartEncounter)
	mux.HandleFunc("POST /api/submit-rolls", server.handleSubmitRolls)
	mux.HandleFunc("POST /api/next-turn", server.handleNextTurn)
	mux.HandleFunc("POST /api/hp-delta", server.handleHPDelta)
	mux.HandleFunc("POST /api/edit-combatant", server.handleEditCombatant)
	mux.HandleFunc("POST /api/save", server.handleSave)

	return mux
}

type encounterIDRequest struct {
	EncounterID string `json:"encounter_id"`
}

type renameEncounterRequest struct {
	EncounterID string `json:"encounter_id"`
	Name        string `json:"name"`
}

type nameRequest struct {
	Name string `json:"name"`
}

type addNPCRequest struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Labels string `json:"labels"`
	HP     *int   `json:"hp"`
	AC     *int   `json:"ac"`
}

type addPlayerRequest struct {
	Name            string `json:"name"`
	InitiativeBonus *int   `json:"initiative_bonus"`
}

type saveNPCTemplateRequest struct {
	Name            string `json:"name"`
	AC              *int   `json:"ac"`
	HP              *int   `json:"hp"`
	Dex             *int   `json:"dex"`
	InitiativeBonus *int   `json:"initiative_bonus"`
	Tags            string `json:"tags"`
	Notes           string `json:"notes"`
	Markdown        string `json:"markdown"`
}

type savePlayerTemplateRequest struct {
	Name            string `json:"name"`
	AC              *int   `json:"ac"`
	MaxHP           *int   `json:"max_hp"`
	CurrentHP       *int   `json:"current_hp"`
	Dex             *int   `json:"dex"`
	InitiativeBonus *int   `json:"initiative_bonus"`
	Notes           string `json:"notes"`
	Markdown        string `json:"markdown"`
}

type indexRequest struct {
	Index int `json:"index"`
}

type submitRollsRequest struct {
	Rolls []*int `json:"rolls"`
}

type hpDeltaRequest struct {
	Index int `json:"index"`
	Delta int `json:"delta"`
}

type editCombatantRequest struct {
	Index          int    `json:"index"`
	DisplayName    *string `json:"display_name"`
	AC             *int    `json:"ac"`
	MaxHP          *int    `json:"max_hp"`
	CurrentHP      *int    `json:"current_hp"`
	Notes          *string `json:"notes"`
	InitiativeTotal *int   `json:"initiative_total"`
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	serveEmbeddedText(w, "home.html", "text/html; charset=utf-8")
}

func (s *Server) handleEncounterNewPage(w http.ResponseWriter, _ *http.Request) {
	s.tracker.StartNewEncounter()
	serveEmbeddedText(w, "setup.html", "text/html; charset=utf-8")
}

func (s *Server) handleEncounterPage(w http.ResponseWriter, r *http.Request) {
	ok, err := s.tracker.ResumeEncounter(pathValue(r, "encounter_id"))
	if err != nil {
		writeServerError(w, err)
		return
	}
	if !ok {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	serveEmbeddedText(w, "combat.html", "text/html; charset=utf-8")
}

func (s *Server) handleSavesPage(w http.ResponseWriter, _ *http.Request) {
	serveEmbeddedText(w, "saves.html", "text/html; charset=utf-8")
}

func (s *Server) handleNPCsPage(w http.ResponseWriter, _ *http.Request) {
	serveEmbeddedText(w, "npc-list.html", "text/html; charset=utf-8")
}

func (s *Server) handlePlayersPage(w http.ResponseWriter, _ *http.Request) {
	serveEmbeddedText(w, "player-list.html", "text/html; charset=utf-8")
}

func (s *Server) handleAppJS(w http.ResponseWriter, _ *http.Request) {
	serveEmbeddedText(w, "app.js", "application/javascript")
}

func (s *Server) handleStylesCSS(w http.ResponseWriter, _ *http.Request) {
	serveEmbeddedText(w, "styles.css", "text/css; charset=utf-8")
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	state, err := s.tracker.GetState()
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleGoHome(w http.ResponseWriter, _ *http.Request) {
	if err := s.tracker.GoHome(); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleNewEncounter(w http.ResponseWriter, _ *http.Request) {
	s.tracker.StartNewEncounter()
	s.handleState(w, nil)
}

func (s *Server) handleResumeEncounter(w http.ResponseWriter, r *http.Request) {
	var request encounterIDRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if _, err := s.tracker.ResumeEncounter(request.EncounterID); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleSetEncounterName(w http.ResponseWriter, r *http.Request) {
	var request nameRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	s.tracker.SetEncounterName(request.Name)
	s.handleState(w, nil)
}

func (s *Server) handleAddNPC(w http.ResponseWriter, r *http.Request) {
	var request addNPCRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Count == 0 {
		request.Count = 1
	}
	if err := s.tracker.AddNPC(request.Name, request.Count, request.Labels, request.HP, request.AC); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleAddPlayer(w http.ResponseWriter, r *http.Request) {
	var request addPlayerRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.tracker.AddPlayer(request.Name, request.InitiativeBonus); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleListEncounters(w http.ResponseWriter, _ *http.Request) {
	encounters, err := s.tracker.ListEncounters()
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"encounters": encounters})
}

func (s *Server) handleRenameEncounter(w http.ResponseWriter, r *http.Request) {
	var request renameEncounterRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	message, err := s.tracker.RenameEncounter(request.EncounterID, request.Name)
	if err != nil {
		writeServerError(w, err)
		return
	}
	encounters, err := s.tracker.ListEncounters()
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"encounters": encounters, "message": message})
}

func (s *Server) handleDeleteEncounter(w http.ResponseWriter, r *http.Request) {
	var request encounterIDRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	message, err := s.tracker.DeleteEncounter(request.EncounterID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	encounters, err := s.tracker.ListEncounters()
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"encounters": encounters, "message": message})
}

func (s *Server) handleListNPCTemplates(w http.ResponseWriter, _ *http.Request) {
	templates, message, err := s.tracker.ListNPCTemplates()
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates, "message": message})
}

func (s *Server) handleGetNPCTemplate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	template, err := s.tracker.GetNPCTemplate(name)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if template == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "NPC '" + name + "' not found."})
		return
	}
	writeJSON(w, http.StatusOK, template)
}

func (s *Server) handleDeleteNPCTemplate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	templates, message, deleted, err := s.tracker.DeleteNPCTemplate(name)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "NPC '" + name + "' not found."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates, "message": message})
}

func (s *Server) handleListPlayerTemplates(w http.ResponseWriter, _ *http.Request) {
	templates, message, err := s.tracker.ListPlayerTemplates()
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates, "message": message})
}

func (s *Server) handleGetPlayerTemplate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	template, err := s.tracker.GetPlayerTemplate(name)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if template == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Player '" + name + "' not found."})
		return
	}
	writeJSON(w, http.StatusOK, template)
}

func (s *Server) handleDeletePlayerTemplate(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	templates, message, deleted, err := s.tracker.DeletePlayerTemplate(name)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Player '" + name + "' not found."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates, "message": message})
}

func (s *Server) handleSaveNPCTemplate(w http.ResponseWriter, r *http.Request) {
	var request saveNPCTemplateRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.tracker.SaveNPCTemplate(SaveNPCTemplateInput{
		Name:            request.Name,
		AC:              request.AC,
		HP:              request.HP,
		Dex:             request.Dex,
		InitiativeBonus: request.InitiativeBonus,
		TagsRaw:         request.Tags,
		Notes:           request.Notes,
		Markdown:        request.Markdown,
	}); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleSavePlayerTemplate(w http.ResponseWriter, r *http.Request) {
	var request savePlayerTemplateRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.tracker.SavePlayerTemplate(SavePlayerTemplateInput{
		Name:            request.Name,
		AC:              request.AC,
		MaxHP:           request.MaxHP,
		CurrentHP:       request.CurrentHP,
		Dex:             request.Dex,
		InitiativeBonus: request.InitiativeBonus,
		Notes:           request.Notes,
		Markdown:        request.Markdown,
	}); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleAddNPCToCombat(w http.ResponseWriter, r *http.Request) {
	var request addNPCRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Count == 0 {
		request.Count = 1
	}
	if err := s.tracker.AddNPCToCombat(request.Name, request.Count, request.Labels, request.HP, request.AC); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleRollNPC(w http.ResponseWriter, _ *http.Request) {
	s.tracker.RollNPCInitiative()
	s.handleState(w, nil)
}

func (s *Server) handleRemoveCombatant(w http.ResponseWriter, r *http.Request) {
	var request indexRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	s.tracker.RemoveCombatant(request.Index)
	s.handleState(w, nil)
}

func (s *Server) handleSelect(w http.ResponseWriter, r *http.Request) {
	var request indexRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	s.tracker.Select(request.Index)
	s.handleState(w, nil)
}

func (s *Server) handleStartEncounter(w http.ResponseWriter, _ *http.Request) {
	players, ready := s.tracker.PlayersNeedingRolls()
	if !ready {
		s.handleState(w, nil)
		return
	}
	if len(players) > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"need_rolls": true, "players": players})
		return
	}
	if err := s.tracker.StartEncounter(nil); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleSubmitRolls(w http.ResponseWriter, r *http.Request) {
	var request submitRollsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.tracker.StartEncounter(request.Rolls); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleNextTurn(w http.ResponseWriter, _ *http.Request) {
	if err := s.tracker.AdvanceTurn(); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleHPDelta(w http.ResponseWriter, r *http.Request) {
	var request hpDeltaRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.tracker.ApplyHPDelta(request.Index, request.Delta); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleEditCombatant(w http.ResponseWriter, r *http.Request) {
	var request editCombatantRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.tracker.EditCombatant(
		request.Index,
		request.DisplayName,
		request.AC,
		request.MaxHP,
		request.CurrentHP,
		request.Notes,
		request.InitiativeTotal,
	); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func (s *Server) handleSave(w http.ResponseWriter, _ *http.Request) {
	if err := s.tracker.SaveEncounter(); err != nil {
		writeServerError(w, err)
		return
	}
	s.handleState(w, nil)
}

func serveEmbeddedText(w http.ResponseWriter, name string, contentType string) {
	content, err := webAssets.ReadFile("web/" + name)
	if err != nil {
		writeServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(destination); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeServerError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
}

func pathValue(r *http.Request, name string) string {
	value := r.PathValue(name)
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}
