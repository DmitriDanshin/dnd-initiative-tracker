package tracker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartEncounterRequestsPlayerRollsViaHTTP(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	postJSON(t, handler, "/api/save-player-template", map[string]any{
		"name":             "Aramil",
		"initiative_bonus": 3,
	})
	postJSON(t, handler, "/api/new-encounter", map[string]any{})
	postJSON(t, handler, "/api/add-player", map[string]any{
		"name": "Aramil",
	})

	response := postJSON(t, handler, "/api/start-encounter", map[string]any{})
	if response["need_rolls"] != true {
		t.Fatalf("expected need_rolls=true, got %#v", response)
	}

	players, ok := response["players"].([]any)
	if !ok || len(players) != 1 {
		t.Fatalf("unexpected players payload: %#v", response["players"])
	}
}

func TestNextTurnKeepsActiveEncounterViaHTTP(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	postJSON(t, handler, "/api/save-npc-template", map[string]any{
		"name": "Goblin",
		"ac":   15,
		"hp":   7,
		"dex":  14,
	})
	postJSON(t, handler, "/api/new-encounter", map[string]any{})
	postJSON(t, handler, "/api/add-npc", map[string]any{
		"name":  "Goblin",
		"count": 1,
	})

	startResponse := postJSON(t, handler, "/api/start-encounter", map[string]any{})
	encounter, ok := startResponse["encounter"].(map[string]any)
	if !ok {
		t.Fatalf("expected encounter payload, got %#v", startResponse)
	}

	encounterID, ok := encounter["encounter_id"].(string)
	if !ok || encounterID == "" {
		t.Fatalf("expected encounter_id, got %#v", encounter["encounter_id"])
	}

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/encounter/"+encounterID, nil)
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET /encounter returned %d: %s", getResponse.Code, getResponse.Body.String())
	}

	nextTurnResponse := postJSON(t, handler, "/api/next-turn", map[string]any{})
	if nextTurnResponse["encounter"] == nil {
		t.Fatalf("expected encounter after next turn, got %#v", nextTurnResponse)
	}

	stateResponse := getJSON(t, handler, "/api/state")
	if stateResponse["encounter"] == nil {
		t.Fatalf("expected encounter in state after next turn, got %#v", stateResponse)
	}
}

func postJSON(t *testing.T, handler http.Handler, path string, payload any) map[string]any {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("%s returned %d: %s", path, responseRecorder.Code, responseRecorder.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func getJSON(t *testing.T, handler http.Handler, path string) map[string]any {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, path, nil)
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("%s returned %d: %s", path, responseRecorder.Code, responseRecorder.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}
