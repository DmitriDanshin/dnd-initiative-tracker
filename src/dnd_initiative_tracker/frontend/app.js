let state = { mode: "home", message: "" };

window.onerror = function(message) {
  document.getElementById("message").textContent = "JS Error: " + message;
};

window.onunhandledrejection = function(event) {
  document.getElementById("message").textContent = "Error: " + (event.reason?.message || event.reason || "unknown");
};

async function api(method, path, body) {
  const options = { method, headers: { "Content-Type": "application/json" } };
  if (body !== undefined) options.body = JSON.stringify(body);
  const response = await fetch(path, options);
  if (!response.ok) throw new Error("HTTP " + response.status + ": " + response.statusText);
  return response.json();
}

async function load() {
  try {
    const page = currentPage();
    if (page === "npcs") {
      const data = await api("GET", "/api/npc-templates");
      state = { mode: "npcs", npc_templates_full: data.templates, message: data.message || "" };
    } else if (page === "players") {
      const data = await api("GET", "/api/player-templates");
      state = { mode: "players", player_templates_full: data.templates, message: data.message || "" };
    } else {
      state = await api("GET", "/api/state");
    }
    render();
  } catch (error) {
    console.error("Failed to load state:", error);
    document.getElementById("message").textContent = "Error: " + error.message;
  }
}

function currentPage() {
  return document.body.dataset.page || "home";
}

function getEncounterUrl(encounterId) {
  return "/encounter/" + encodeURIComponent(encounterId);
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function parseOptionalInteger(elementId) {
  const raw = document.getElementById(elementId).value.trim();
  return raw === "" ? null : parseInt(raw, 10);
}

function setMessage(message) {
  document.getElementById("message").textContent = message || "";
}

function setTitle(title) {
  document.getElementById("screenTitle").textContent = title;
}

function clearModalValidation() {
  const overlay = document.getElementById("modalOverlay");
  if (!overlay || overlay.classList.contains("hidden")) return;
  overlay.querySelectorAll(".invalid").forEach((element) => element.classList.remove("invalid"));
  overlay.querySelectorAll(".field-error").forEach((element) => {
    element.textContent = "";
  });
}

function applyTemplateFieldErrors(fieldMap) {
  clearModalValidation();
  Object.entries(fieldMap || {}).forEach(([fieldName, message]) => {
    const input = document.querySelector(`[data-field-name="${fieldName}"]`);
    if (input) input.classList.add("invalid");
    const errorNode = document.querySelector(`[data-field-error="${fieldName}"]`);
    if (errorNode) errorNode.textContent = message;
  });
}

function handleTemplateSaveResponse(response) {
  const page = currentPage();
  if (page === "npcs" || page === "players") {
    if (response.field_errors && Object.keys(response.field_errors).length) {
      state.message = response.message || "";
      state.field_errors = response.field_errors;
      setMessage(state.message);
      applyTemplateFieldErrors(response.field_errors);
      return;
    }
    closeModal();
    load();
    return;
  }
  state = response;
  render();
  if (response.field_errors && Object.keys(response.field_errors).length) {
    applyTemplateFieldErrors(response.field_errors);
    return;
  }
  closeModal();
}

function render() {
  setMessage(state.message || "");
  const app = document.getElementById("app");
  const page = currentPage();
  if (page === "home") renderHome(app);
  else if (page === "setup") renderSetup(app);
  else if (page === "combat") renderCombat(app);
  else if (page === "npcs") renderNpcList(app);
  else if (page === "players") renderPlayerList(app);
}

function renderHome(app) {
  setTitle("Home");
  const encounters = state.encounters || [];
  const npcCount = (state.npc_templates || []).length;
  const playerCount = (state.player_templates || []).length;
  let list = "";
  if (encounters.length) {
    list = "<h2>Saved Encounters</h2><ul class=\"saves-list\">";
    encounters.forEach((encounter) => {
      list += "<li onclick=\"resumeEncounter('" + encounter.encounter_id + "')\">"
        + encounter.encounter_name + " <span style=\"color:var(--muted)\">(round " + encounter.round + ")</span></li>";
    });
    list += "</ul>";
  }
  app.innerHTML = `
    <div class="home-menu">
      <button class="primary" onclick="newEncounter()">New Encounter</button>
      <button onclick="window.location.href='/npcs'">NPCs <span style="color:var(--muted)">(${npcCount})</span></button>
      <button onclick="window.location.href='/players'">Players <span style="color:var(--muted)">(${playerCount})</span></button>
    </div>
    ${list}`;
}

function renderSetup(app) {
  setTitle("Setup | " + (state.setup_encounter_name || "New Encounter"));
  const npcs = (state.npc_templates || []).map((template) => template.name).join(", ") || "-";
  const players = (state.player_templates || []).join(", ") || "-";
  let rows = "";
  (state.setup_combatants || []).forEach((combatant, index) => {
    const hp = combatant.max_hp != null ? combatant.current_hp + "/" + combatant.max_hp : "-";
    const ac = combatant.ac != null ? combatant.ac : "-";
    const initiative = combatant.initiative_total != null ? combatant.initiative_total : "-";
    const token = combatant.token_label || "-";
    rows += "<tr onclick=\"selectSetup(" + index + ")\" class=\"" + (index === state.selected_index ? "selected" : "") + "\">"
      + "<td>" + initiative + "</td><td>" + token + "</td><td>" + combatant.display_name + "</td>"
      + "<td>" + combatant.kind + "</td><td>" + hp + "</td><td>" + ac + "</td>"
      + "<td><button onclick=\"event.stopPropagation();removeSetup(" + index + ")\">X</button></td></tr>";
  });
  if (!rows) rows = "<tr><td colspan=\"7\" style=\"color:var(--muted)\">No combatants yet</td></tr>";

  app.innerHTML = `
    <div style="margin-bottom:12px;color:var(--muted)">NPC templates: ${npcs}<br>Player templates: ${players}</div>
    <div class="btn-group">
      <button onclick="showAddNpc()">Add NPC</button>
      <button onclick="showAddPlayer()">Add Player</button>
      <button onclick="rollNpc()">Roll NPC Initiative</button>
      <button onclick="showEditName()">Encounter Name</button>
      <button class="green" onclick="startEncounter()">Start Encounter</button>
      <button onclick="goHome()">Back</button>
    </div>
    <table>
      <thead><tr><th>Init</th><th>Token</th><th>Name</th><th>Type</th><th>HP</th><th>AC</th><th></th></tr></thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function renderCombat(app) {
  const encounter = state.encounter;
  if (!encounter) {
    app.innerHTML = "<p>No active encounter.</p>";
    return;
  }
  setTitle("Combat | " + encounter.encounter_name + " | Round " + encounter.round);
  let rows = "";
  (encounter.combatants || []).forEach((combatant, index) => {
    const isActive = index === encounter.active_index;
    const isSelected = index === state.selected_index;
    const isDowned = combatant.current_hp != null && combatant.current_hp <= 0;
    const classes = (isActive ? "active-turn " : "") + (isSelected ? "selected " : "") + (isDowned ? "downed" : "");
    const hp = combatant.max_hp != null ? combatant.current_hp + "/" + combatant.max_hp : "-";
    const hpPercent = combatant.max_hp ? Math.max(0, Math.min(100, (combatant.current_hp / combatant.max_hp) * 100)) : 0;
    const hpColor = hpPercent > 50 ? "var(--green)" : hpPercent > 25 ? "var(--yellow)" : "var(--red)";
    const hpBar = combatant.max_hp != null
      ? "<div class=\"hp-bar\"><div class=\"hp-fill\" style=\"width:" + hpPercent + "%;background:" + hpColor + "\"></div></div>"
      : "";
    const ac = combatant.ac != null ? combatant.ac : "-";
    const initiative = combatant.initiative_total != null ? combatant.initiative_total : "-";
    const token = combatant.token_label || "-";
    const activeIcon = isActive ? "\u25B6 " : "";
    rows += "<tr class=\"" + classes + "\" onclick=\"selectCombat(" + index + ")\">"
      + "<td>" + activeIcon + initiative + "</td><td>" + token + "</td><td>" + combatant.display_name + "</td>"
      + "<td>" + hp + " " + hpBar + "</td><td>" + ac + "</td></tr>";
  });

  let detail = "";
  if (encounter.combatants && encounter.combatants.length > 0) {
    const selectedCombatant = encounter.combatants[state.selected_index] || encounter.combatants[0];
    detail = "<div class=\"detail-card\"><dl>"
      + "<dt>Name</dt><dd>" + selectedCombatant.display_name + "</dd>"
      + "<dt>Token</dt><dd>" + (selectedCombatant.token_label || "-") + "</dd>"
      + "<dt>Initiative</dt><dd>" + (selectedCombatant.initiative_total != null ? selectedCombatant.initiative_total : "-") + "</dd>"
      + "<dt>HP</dt><dd>" + (selectedCombatant.current_hp != null ? selectedCombatant.current_hp : "-") + "/" + (selectedCombatant.max_hp != null ? selectedCombatant.max_hp : "-") + "</dd>"
      + "<dt>AC</dt><dd>" + (selectedCombatant.ac != null ? selectedCombatant.ac : "-") + "</dd>"
      + "<dt>Notes</dt><dd>" + (selectedCombatant.notes || "-") + "</dd>"
      + "</dl></div>";
  }

  app.innerHTML = `
    <div class="btn-group">
      <button class="primary" onclick="nextTurn()">Next Turn</button>
      <button onclick="showHpDelta()">HP +/-</button>
      <button onclick="showAddNpcCombat()">Add NPC</button>
      <button onclick="saveEncounter()">Save</button>
      <button onclick="goHome()">Back</button>
    </div>
    <table>
      <thead><tr><th>Init</th><th>Token</th><th>Name</th><th>HP</th><th>AC</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    ${detail}`;
}

// --- Add NPC during Combat ---

function showAddNpcCombat() {
  const templates = state.npc_templates || [];
  if (!templates.length) {
    showAddNpcCombatNewOnly();
    return;
  }
  const options = templates.map((template) => {
    return "<option value=\"" + escapeHtml(template.name) + "\" data-hp=\"" + template.hp + "\" data-ac=\"" + template.ac + "\">" + escapeHtml(template.name) + "</option>";
  }).join("");
  const first = templates[0] || { hp: "", ac: "" };
  showModal(`
    <h3>Add NPC to Combat</h3>
    <div class="field"><label>NPC Template</label>
      <select id="mdl_combat_npc_name" onchange="onCombatNpcSelect()">
        ${options}
        <option value="">-- create new --</option>
      </select></div>
    <div id="combatNpcExistingFields">
      <div class="field"><label>HP</label><input id="mdl_combat_npc_hp" type="number" value="${first.hp}" min="1"></div>
      <div class="field"><label>AC</label><input id="mdl_combat_npc_ac" type="number" value="${first.ac}" min="0"></div>
      <div class="field"><label>Count</label><input id="mdl_combat_npc_count" type="number" value="1" min="1"></div>
      <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_combat_npc_labels" placeholder="B1,B2,B3"></div>
    </div>
    <div id="combatNpcNewFields" class="hidden">
      <div class="field"><label>Name</label><input id="mdl_combat_new_npc_name" data-field-name="name"><div class="field-error" data-field-error="name"></div></div>
      <div class="field"><label>AC</label><input id="mdl_combat_new_npc_ac" data-field-name="ac" type="number" min="0"><div class="field-error" data-field-error="ac"></div></div>
      <div class="field"><label>HP</label><input id="mdl_combat_new_npc_hp" data-field-name="hp" type="number" min="0"><div class="field-error" data-field-error="hp"></div></div>
      <div class="field"><label>DEX</label><input id="mdl_combat_new_npc_dex" data-field-name="dex" type="number"><div class="field-error" data-field-error="dex"></div></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_combat_new_npc_bonus" data-field-name="initiative_bonus" type="number"><div class="field-error" data-field-error="initiative_bonus"></div></div>
      <div class="field"><label>Count</label><input id="mdl_combat_new_npc_count" type="number" value="1" min="1"></div>
      <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_combat_new_npc_labels" placeholder="B1,B2,B3"></div>
    </div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitAddNpcCombat()">Add</button>
    </div>
  `);
}

function showAddNpcCombatNewOnly() {
  showModal(`
    <h3>Add NPC to Combat</h3>
    <p style="color:var(--muted);margin-bottom:12px">No NPC templates yet. Create one:</p>
    <div class="field"><label>Name</label><input id="mdl_combat_new_npc_name" data-field-name="name"><div class="field-error" data-field-error="name"></div></div>
    <div class="field"><label>AC</label><input id="mdl_combat_new_npc_ac" data-field-name="ac" type="number" min="0"><div class="field-error" data-field-error="ac"></div></div>
    <div class="field"><label>HP</label><input id="mdl_combat_new_npc_hp" data-field-name="hp" type="number" min="0"><div class="field-error" data-field-error="hp"></div></div>
    <div class="field"><label>DEX</label><input id="mdl_combat_new_npc_dex" data-field-name="dex" type="number"><div class="field-error" data-field-error="dex"></div></div>
    <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_combat_new_npc_bonus" data-field-name="initiative_bonus" type="number"><div class="field-error" data-field-error="initiative_bonus"></div></div>
    <div class="field"><label>Count</label><input id="mdl_combat_new_npc_count" type="number" value="1" min="1"></div>
    <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_combat_new_npc_labels" placeholder="B1,B2,B3"></div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitAddNpcCombatNew()">Create & Add</button>
    </div>
  `);
}

function onCombatNpcSelect() {
  const select = document.getElementById("mdl_combat_npc_name");
  const isNew = select.value === "";
  document.getElementById("combatNpcExistingFields").classList.toggle("hidden", isNew);
  document.getElementById("combatNpcNewFields").classList.toggle("hidden", !isNew);
  if (!isNew) {
    const option = select.options[select.selectedIndex];
    document.getElementById("mdl_combat_npc_hp").value = option.dataset.hp || "";
    document.getElementById("mdl_combat_npc_ac").value = option.dataset.ac || "";
  }
}

async function submitAddNpcCombat() {
  const select = document.getElementById("mdl_combat_npc_name");
  if (select && select.value === "") {
    await submitAddNpcCombatNew();
    return;
  }
  const name = select ? select.value : "";
  const count = parseInt(document.getElementById("mdl_combat_npc_count").value, 10) || 1;
  const labels = document.getElementById("mdl_combat_npc_labels").value;
  const hp = parseOptionalInteger("mdl_combat_npc_hp");
  const ac = parseOptionalInteger("mdl_combat_npc_ac");
  closeModal();
  await api("POST", "/api/add-npc-to-combat", { name, count, labels, hp, ac });
  await load();
}

async function submitAddNpcCombatNew() {
  clearModalValidation();
  const name = document.getElementById("mdl_combat_new_npc_name").value.trim();
  const ac = parseOptionalInteger("mdl_combat_new_npc_ac");
  const hp = parseOptionalInteger("mdl_combat_new_npc_hp");
  const dex = parseOptionalInteger("mdl_combat_new_npc_dex");
  const initiative_bonus = parseOptionalInteger("mdl_combat_new_npc_bonus");
  const count = parseInt(document.getElementById("mdl_combat_new_npc_count").value, 10) || 1;
  const labels = document.getElementById("mdl_combat_new_npc_labels").value;

  const saveResponse = await api("POST", "/api/save-npc-template", { name, ac, hp, dex, initiative_bonus, tags: "", notes: "" });
  if (saveResponse.field_errors && Object.keys(saveResponse.field_errors).length) {
    applyTemplateFieldErrors(saveResponse.field_errors);
    setMessage(saveResponse.message || "Validation failed.");
    return;
  }
  closeModal();
  await api("POST", "/api/add-npc-to-combat", { name, count, labels, hp, ac });
  await load();
}

// --- NPC List Page ---

let selectedNpcName = null;

function renderNpcList(app) {
  setTitle("NPCs");
  const templates = state.npc_templates_full || [];
  let rows = "";
  templates.forEach((t) => {
    const isSelected = t.name === selectedNpcName;
    rows += "<tr class=\"" + (isSelected ? "selected" : "") + "\" onclick=\"selectNpc('" + escapeHtml(t.name) + "')\">"
      + "<td>" + escapeHtml(t.name) + "</td>"
      + "<td>" + t.ac + "</td>"
      + "<td>" + t.hp + "</td>"
      + "<td>" + t.dex + "</td>"
      + "<td>" + (t.initiative_bonus != null ? t.initiative_bonus : "-") + "</td>"
      + "<td>" + escapeHtml((t.tags || []).join(", ") || "-") + "</td>"
      + "</tr>";
  });
  if (!rows) rows = "<tr><td colspan=\"6\" style=\"color:var(--muted)\">No NPC templates yet</td></tr>";

  let detail = "";
  if (selectedNpcName) {
    const t = templates.find((t) => t.name === selectedNpcName);
    if (t) {
      detail = "<div class=\"detail-card\"><dl>"
        + "<dt>Name</dt><dd>" + escapeHtml(t.name) + "</dd>"
        + "<dt>AC</dt><dd>" + t.ac + "</dd>"
        + "<dt>HP</dt><dd>" + t.hp + "</dd>"
        + "<dt>DEX</dt><dd>" + t.dex + "</dd>"
        + "<dt>Initiative Bonus</dt><dd>" + (t.initiative_bonus != null ? t.initiative_bonus : "-") + "</dd>"
        + "<dt>Tags</dt><dd>" + escapeHtml((t.tags || []).join(", ") || "-") + "</dd>"
        + "<dt>Notes</dt><dd>" + escapeHtml(t.notes || "-") + "</dd>"
        + "</dl>"
        + "<div class=\"btn-group\" style=\"margin-top:12px\">"
        + "<button onclick=\"showEditNpcTemplate('" + escapeHtml(t.name) + "')\">Edit</button>"
        + "<button class=\"danger\" onclick=\"confirmDeleteNpc('" + escapeHtml(t.name) + "')\">Delete</button>"
        + "</div></div>";
    }
  }

  app.innerHTML = `
    <div class="btn-group">
      <button onclick="showCreateNpcTemplate()">Add NPC Template</button>
      <button onclick="goHome()">Back</button>
    </div>
    <table>
      <thead><tr><th>Name</th><th>AC</th><th>HP</th><th>DEX</th><th>Init Bonus</th><th>Tags</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    ${detail}`;
}

function selectNpc(name) {
  selectedNpcName = selectedNpcName === name ? null : name;
  render();
}

function showEditNpcTemplate(name) {
  const templates = state.npc_templates_full || [];
  const t = templates.find((t) => t.name === name);
  if (!t) return;
  showTemplateModal({
    title: "Edit NPC Template",
    modeKey: "npcTemplate",
    formContent: `
      <div class="field"><label>Name</label><input id="mdl_create_npc_name" data-field-name="name" value="${escapeHtml(t.name)}"><div class="field-error" data-field-error="name"></div></div>
      <div class="field"><label>AC</label><input id="mdl_create_npc_ac" data-field-name="ac" type="number" min="0" value="${t.ac}"><div class="field-error" data-field-error="ac"></div></div>
      <div class="field"><label>HP</label><input id="mdl_create_npc_hp" data-field-name="hp" type="number" min="0" value="${t.hp}"><div class="field-error" data-field-error="hp"></div></div>
      <div class="field"><label>DEX</label><input id="mdl_create_npc_dex" data-field-name="dex" type="number" value="${t.dex}"><div class="field-error" data-field-error="dex"></div></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_create_npc_bonus" data-field-name="initiative_bonus" type="number" value="${t.initiative_bonus != null ? t.initiative_bonus : ""}"><div class="field-error" data-field-error="initiative_bonus"></div></div>
      <div class="field"><label>Tags (comma-separated)</label><input id="mdl_create_npc_tags" data-field-name="tags" value="${escapeHtml((t.tags || []).join(", "))}"><div class="field-error" data-field-error="tags"></div></div>
      <div class="field"><label>Notes</label><textarea id="mdl_create_npc_notes" data-field-name="notes">${escapeHtml(t.notes || "")}</textarea><div class="field-error" data-field-error="notes"></div></div>
    `,
    markdownPlaceholder: `---\nname: Goblin Boss\nac: 17\nhp: 45\ndex: 14\ninitiative_bonus: 2\ntags:\n  - goblinoid\nnotes: Boss of the ambush.\n---`,
    submitAction: "submitCreateNpcTemplate()",
  });
}

function confirmDeleteNpc(name) {
  showModal(`
    <h3>Delete NPC Template</h3>
    <p>Are you sure you want to delete <strong>${escapeHtml(name)}</strong>?</p>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="danger" onclick="deleteNpcTemplate('${escapeHtml(name)}')">Delete</button>
    </div>
  `);
}

async function deleteNpcTemplate(name) {
  closeModal();
  const data = await api("DELETE", "/api/npc-templates/" + encodeURIComponent(name));
  selectedNpcName = null;
  state.npc_templates_full = data.templates;
  state.message = data.message || "";
  render();
}

// --- Player List Page ---

let selectedPlayerName = null;

function renderPlayerList(app) {
  setTitle("Players");
  const templates = state.player_templates_full || [];
  let rows = "";
  templates.forEach((t) => {
    const isSelected = t.name === selectedPlayerName;
    rows += "<tr class=\"" + (isSelected ? "selected" : "") + "\" onclick=\"selectPlayer('" + escapeHtml(t.name) + "')\">"
      + "<td>" + escapeHtml(t.name) + "</td>"
      + "<td>" + (t.ac != null ? t.ac : "-") + "</td>"
      + "<td>" + (t.max_hp != null ? (t.current_hp != null ? t.current_hp : t.max_hp) + "/" + t.max_hp : "-") + "</td>"
      + "<td>" + (t.dex != null ? t.dex : "-") + "</td>"
      + "<td>" + (t.initiative_bonus != null ? t.initiative_bonus : "-") + "</td>"
      + "</tr>";
  });
  if (!rows) rows = "<tr><td colspan=\"5\" style=\"color:var(--muted)\">No player templates yet</td></tr>";

  let detail = "";
  if (selectedPlayerName) {
    const t = templates.find((t) => t.name === selectedPlayerName);
    if (t) {
      detail = "<div class=\"detail-card\"><dl>"
        + "<dt>Name</dt><dd>" + escapeHtml(t.name) + "</dd>"
        + "<dt>AC</dt><dd>" + (t.ac != null ? t.ac : "-") + "</dd>"
        + "<dt>Max HP</dt><dd>" + (t.max_hp != null ? t.max_hp : "-") + "</dd>"
        + "<dt>Current HP</dt><dd>" + (t.current_hp != null ? t.current_hp : "-") + "</dd>"
        + "<dt>DEX</dt><dd>" + (t.dex != null ? t.dex : "-") + "</dd>"
        + "<dt>Initiative Bonus</dt><dd>" + (t.initiative_bonus != null ? t.initiative_bonus : "-") + "</dd>"
        + "<dt>Notes</dt><dd>" + escapeHtml(t.notes || "-") + "</dd>"
        + "</dl>"
        + "<div class=\"btn-group\" style=\"margin-top:12px\">"
        + "<button onclick=\"showEditPlayerTemplate('" + escapeHtml(t.name) + "')\">Edit</button>"
        + "<button class=\"danger\" onclick=\"confirmDeletePlayer('" + escapeHtml(t.name) + "')\">Delete</button>"
        + "</div></div>";
    }
  }

  app.innerHTML = `
    <div class="btn-group">
      <button onclick="showCreatePlayerTemplate()">Add Player Template</button>
      <button onclick="goHome()">Back</button>
    </div>
    <table>
      <thead><tr><th>Name</th><th>AC</th><th>HP</th><th>DEX</th><th>Init Bonus</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    ${detail}`;
}

function selectPlayer(name) {
  selectedPlayerName = selectedPlayerName === name ? null : name;
  render();
}

function showEditPlayerTemplate(name) {
  const templates = state.player_templates_full || [];
  const t = templates.find((t) => t.name === name);
  if (!t) return;
  showTemplateModal({
    title: "Edit Player Template",
    modeKey: "playerTemplate",
    formContent: `
      <div class="field"><label>Name</label><input id="mdl_create_player_name" data-field-name="name" value="${escapeHtml(t.name)}"><div class="field-error" data-field-error="name"></div></div>
      <div class="field"><label>AC (optional)</label><input id="mdl_create_player_ac" data-field-name="ac" type="number" min="0" value="${t.ac != null ? t.ac : ""}"><div class="field-error" data-field-error="ac"></div></div>
      <div class="field"><label>Max HP (optional)</label><input id="mdl_create_player_max_hp" data-field-name="max_hp" type="number" min="0" value="${t.max_hp != null ? t.max_hp : ""}"><div class="field-error" data-field-error="max_hp"></div></div>
      <div class="field"><label>Current HP (optional)</label><input id="mdl_create_player_current_hp" data-field-name="current_hp" type="number" min="0" value="${t.current_hp != null ? t.current_hp : ""}"><div class="field-error" data-field-error="current_hp"></div></div>
      <div class="field"><label>DEX (optional)</label><input id="mdl_create_player_dex" data-field-name="dex" type="number" value="${t.dex != null ? t.dex : ""}"><div class="field-error" data-field-error="dex"></div></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_create_player_bonus" data-field-name="initiative_bonus" type="number" value="${t.initiative_bonus != null ? t.initiative_bonus : ""}"><div class="field-error" data-field-error="initiative_bonus"></div></div>
      <div class="field"><label>Notes</label><textarea id="mdl_create_player_notes" data-field-name="notes">${escapeHtml(t.notes || "")}</textarea><div class="field-error" data-field-error="notes"></div></div>
    `,
    markdownPlaceholder: `---\nname: Aramil\nac: 15\nmax_hp: 28\ncurrent_hp: 28\ndex: 16\ninitiative_bonus: 3\nnotes: Keeps Bless ready.\n---`,
    submitAction: "submitCreatePlayerTemplate()",
  });
}

function confirmDeletePlayer(name) {
  showModal(`
    <h3>Delete Player Template</h3>
    <p>Are you sure you want to delete <strong>${escapeHtml(name)}</strong>?</p>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="danger" onclick="deletePlayerTemplate('${escapeHtml(name)}')">Delete</button>
    </div>
  `);
}

async function deletePlayerTemplate(name) {
  closeModal();
  const data = await api("DELETE", "/api/player-templates/" + encodeURIComponent(name));
  selectedPlayerName = null;
  state.player_templates_full = data.templates;
  state.message = data.message || "";
  render();
}

function newEncounter() {
  window.location.href = "/encounter/new";
}

function resumeEncounter(encounterId) {
  window.location.href = getEncounterUrl(encounterId);
}

function goHome() {
  window.location.href = "/";
}

async function selectSetup(index) {
  await api("POST", "/api/select", { index });
  await load();
}

async function removeSetup(index) {
  await api("POST", "/api/remove-combatant", { index });
  await load();
}

async function rollNpc() {
  await api("POST", "/api/roll-npc");
  await load();
}

function showAddNpc() {
  const templates = state.npc_templates || [];
  if (!templates.length) {
    showAddNpcNewOnly();
    return;
  }
  const options = templates.map((template) => {
    return "<option value=\"" + escapeHtml(template.name) + "\" data-hp=\"" + template.hp + "\" data-ac=\"" + template.ac + "\">" + escapeHtml(template.name) + "</option>";
  }).join("");
  const first = templates[0] || { hp: "", ac: "" };
  showModal(`
    <h3>Add NPC</h3>
    <div class="field"><label>NPC Template</label>
      <select id="mdl_npc_name" onchange="onNpcSelect()">
        ${options}
        <option value="">-- create new --</option>
      </select></div>
    <div id="setupNpcExistingFields">
      <div class="field"><label>HP</label><input id="mdl_npc_hp" type="number" value="${first.hp}" min="1"></div>
      <div class="field"><label>AC</label><input id="mdl_npc_ac" type="number" value="${first.ac}" min="0"></div>
      <div class="field"><label>Count</label><input id="mdl_npc_count" type="number" value="1" min="1"></div>
      <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_npc_labels" placeholder="B1,B2,B3"></div>
    </div>
    <div id="setupNpcNewFields" class="hidden">
      <div class="field"><label>Name</label><input id="mdl_setup_new_npc_name" data-field-name="name"><div class="field-error" data-field-error="name"></div></div>
      <div class="field"><label>AC</label><input id="mdl_setup_new_npc_ac" data-field-name="ac" type="number" min="0"><div class="field-error" data-field-error="ac"></div></div>
      <div class="field"><label>HP</label><input id="mdl_setup_new_npc_hp" data-field-name="hp" type="number" min="0"><div class="field-error" data-field-error="hp"></div></div>
      <div class="field"><label>DEX</label><input id="mdl_setup_new_npc_dex" data-field-name="dex" type="number"><div class="field-error" data-field-error="dex"></div></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_setup_new_npc_bonus" data-field-name="initiative_bonus" type="number"><div class="field-error" data-field-error="initiative_bonus"></div></div>
      <div class="field"><label>Count</label><input id="mdl_setup_new_npc_count" type="number" value="1" min="1"></div>
      <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_setup_new_npc_labels" placeholder="B1,B2,B3"></div>
    </div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitAddNpc()">Add</button>
    </div>
  `);
}

function showAddNpcNewOnly() {
  showModal(`
    <h3>Add NPC</h3>
    <p style="color:var(--muted);margin-bottom:12px">No NPC templates yet. Create one:</p>
    <div class="field"><label>Name</label><input id="mdl_setup_new_npc_name" data-field-name="name"><div class="field-error" data-field-error="name"></div></div>
    <div class="field"><label>AC</label><input id="mdl_setup_new_npc_ac" data-field-name="ac" type="number" min="0"><div class="field-error" data-field-error="ac"></div></div>
    <div class="field"><label>HP</label><input id="mdl_setup_new_npc_hp" data-field-name="hp" type="number" min="0"><div class="field-error" data-field-error="hp"></div></div>
    <div class="field"><label>DEX</label><input id="mdl_setup_new_npc_dex" data-field-name="dex" type="number"><div class="field-error" data-field-error="dex"></div></div>
    <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_setup_new_npc_bonus" data-field-name="initiative_bonus" type="number"><div class="field-error" data-field-error="initiative_bonus"></div></div>
    <div class="field"><label>Count</label><input id="mdl_setup_new_npc_count" type="number" value="1" min="1"></div>
    <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_setup_new_npc_labels" placeholder="B1,B2,B3"></div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitAddNpcNew()">Create & Add</button>
    </div>
  `);
}

function onNpcSelect() {
  const select = document.getElementById("mdl_npc_name");
  const isNew = select.value === "";
  document.getElementById("setupNpcExistingFields").classList.toggle("hidden", isNew);
  document.getElementById("setupNpcNewFields").classList.toggle("hidden", !isNew);
  if (!isNew) {
    const option = select.options[select.selectedIndex];
    document.getElementById("mdl_npc_hp").value = option.dataset.hp || "";
    document.getElementById("mdl_npc_ac").value = option.dataset.ac || "";
  }
}

async function submitAddNpc() {
  const select = document.getElementById("mdl_npc_name");
  if (select && select.value === "") {
    await submitAddNpcNew();
    return;
  }
  const name = select ? select.value : "";
  const count = parseInt(document.getElementById("mdl_npc_count").value, 10) || 1;
  const labels = document.getElementById("mdl_npc_labels").value;
  const hp = parseOptionalInteger("mdl_npc_hp");
  const ac = parseOptionalInteger("mdl_npc_ac");
  closeModal();
  await api("POST", "/api/add-npc", { name, count, labels, hp, ac });
  await load();
}

async function submitAddNpcNew() {
  clearModalValidation();
  const name = document.getElementById("mdl_setup_new_npc_name").value.trim();
  const ac = parseOptionalInteger("mdl_setup_new_npc_ac");
  const hp = parseOptionalInteger("mdl_setup_new_npc_hp");
  const dex = parseOptionalInteger("mdl_setup_new_npc_dex");
  const initiative_bonus = parseOptionalInteger("mdl_setup_new_npc_bonus");
  const count = parseInt(document.getElementById("mdl_setup_new_npc_count").value, 10) || 1;
  const labels = document.getElementById("mdl_setup_new_npc_labels").value;

  const saveResponse = await api("POST", "/api/save-npc-template", { name, ac, hp, dex, initiative_bonus, tags: "", notes: "" });
  if (saveResponse.field_errors && Object.keys(saveResponse.field_errors).length) {
    applyTemplateFieldErrors(saveResponse.field_errors);
    setMessage(saveResponse.message || "Validation failed.");
    return;
  }
  closeModal();
  await api("POST", "/api/add-npc", { name, count, labels, hp, ac });
  await load();
}

function showAddPlayer() {
  const templates = (state.player_templates || []).map((name) => "<option value=\"" + name + "\">" + name + "</option>").join("");
  const newOption = "<option value=\"\">-- new player --</option>";
  showModal(`
    <h3>Add Player</h3>
    <div class="field"><label>Player</label>
      <select id="mdl_player_name" onchange="toggleNewPlayer()">${templates}${newOption}</select></div>
    <div id="newPlayerFields" class="hidden">
      <div class="field"><label>New Player Name</label><input id="mdl_new_player_name"></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_player_bonus" type="number"></div>
    </div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitAddPlayer()">Add</button>
    </div>
  `);
}

function toggleNewPlayer() {
  const selectedValue = document.getElementById("mdl_player_name").value;
  document.getElementById("newPlayerFields").classList.toggle("hidden", selectedValue !== "");
}

async function submitAddPlayer() {
  let name = document.getElementById("mdl_player_name").value;
  let bonus = null;
  if (!name) {
    name = document.getElementById("mdl_new_player_name").value.trim();
    bonus = parseOptionalInteger("mdl_player_bonus");
  }
  closeModal();
  await api("POST", "/api/add-player", { name, initiative_bonus: bonus });
  await load();
}

function showEditName() {
  showModal(`
    <h3>Encounter Name</h3>
    <div class="field"><label>Name</label><input id="mdl_enc_name" value="${state.setup_encounter_name || ""}"></div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitEditName()">Set</button>
    </div>
  `);
}

async function submitEditName() {
  const name = document.getElementById("mdl_enc_name").value.trim();
  closeModal();
  if (name) {
    await api("POST", "/api/set-encounter-name", { name });
    await load();
  }
}

async function startEncounter() {
  const response = await api("POST", "/api/start-encounter");
  if (response.need_rolls) {
    await promptPlayerRolls(response.players);
    return;
  }
  if (response.encounter) {
    window.location.href = getEncounterUrl(response.encounter.encounter_id);
  }
}

async function promptPlayerRolls(players) {
  let html = "<h3>Player Initiative Rolls</h3>";
  players.forEach((player, index) => {
    html += "<div class=\"field\"><label>" + player.name + " (bonus " + (player.bonus || 0) + ")</label>"
      + "<input id=\"mdl_roll_" + index + "\" type=\"number\" placeholder=\"d20 roll\"></div>";
  });
  html += "<div class=\"btn-group\"><button onclick=\"closeModal()\">Cancel</button>"
    + "<button class=\"primary\" onclick=\"submitRolls(" + players.length + ")\">Start</button></div>";
  showModal(html);
}

async function submitRolls(count) {
  const rolls = [];
  for (let index = 0; index < count; index += 1) {
    const value = document.getElementById("mdl_roll_" + index).value.trim();
    rolls.push(value ? parseInt(value, 10) : null);
  }
  closeModal();
  const response = await api("POST", "/api/submit-rolls", { rolls });
  if (response.encounter) {
    window.location.href = getEncounterUrl(response.encounter.encounter_id);
  }
}

async function selectCombat(index) {
  await api("POST", "/api/select", { index });
  await load();
}

async function nextTurn() {
  await api("POST", "/api/next-turn");
  await load();
}

async function saveEncounter() {
  await api("POST", "/api/save");
  await load();
}

function showHpDelta() {
  const encounter = state.encounter;
  if (!encounter || !encounter.combatants.length) return;
  const combatant = encounter.combatants[state.selected_index];
  if (combatant.current_hp == null) {
    setMessage(combatant.display_name + " does not track HP.");
    return;
  }
  showModal(`
    <h3>HP Delta: ${combatant.display_name}</h3>
    <p>Current HP: ${combatant.current_hp}/${combatant.max_hp || "-"}</p>
    <div class="field"><label>Delta (negative = damage)</label><input id="mdl_hp_delta" type="number" value="-1"></div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitHpDelta()">Apply</button>
    </div>
  `);
}

async function submitHpDelta() {
  const delta = parseInt(document.getElementById("mdl_hp_delta").value, 10) || 0;
  closeModal();
  await api("POST", "/api/hp-delta", { index: state.selected_index, delta });
  await load();
}

function showCreateNpcTemplate() {
  showTemplateModal({
    title: "Add NPC Template",
    modeKey: "npcTemplate",
    formContent: `
      <div class="field"><label>Name</label><input id="mdl_create_npc_name" data-field-name="name"><div class="field-error" data-field-error="name"></div></div>
      <div class="field"><label>AC</label><input id="mdl_create_npc_ac" data-field-name="ac" type="number" min="0"><div class="field-error" data-field-error="ac"></div></div>
      <div class="field"><label>HP</label><input id="mdl_create_npc_hp" data-field-name="hp" type="number" min="0"><div class="field-error" data-field-error="hp"></div></div>
      <div class="field"><label>DEX</label><input id="mdl_create_npc_dex" data-field-name="dex" type="number"><div class="field-error" data-field-error="dex"></div></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_create_npc_bonus" data-field-name="initiative_bonus" type="number"><div class="field-error" data-field-error="initiative_bonus"></div></div>
      <div class="field"><label>Tags (comma-separated)</label><input id="mdl_create_npc_tags" data-field-name="tags" placeholder="goblinoid, scout"><div class="field-error" data-field-error="tags"></div></div>
      <div class="field"><label>Notes</label><textarea id="mdl_create_npc_notes" data-field-name="notes" placeholder="Traits or notes"></textarea><div class="field-error" data-field-error="notes"></div></div>
    `,
    markdownPlaceholder: `---\nname: Goblin Boss\nac: 17\nhp: 45\ndex: 14\ninitiative_bonus: 2\ntags:\n  - goblinoid\nnotes: Boss of the ambush.\n---\nBonus body text is merged into notes.`,
    submitAction: "submitCreateNpcTemplate()",
  });
}

function showCreatePlayerTemplate() {
  showTemplateModal({
    title: "Add Player Template",
    modeKey: "playerTemplate",
    formContent: `
      <div class="field"><label>Name</label><input id="mdl_create_player_name" data-field-name="name"><div class="field-error" data-field-error="name"></div></div>
      <div class="field"><label>AC (optional)</label><input id="mdl_create_player_ac" data-field-name="ac" type="number" min="0"><div class="field-error" data-field-error="ac"></div></div>
      <div class="field"><label>Max HP (optional)</label><input id="mdl_create_player_max_hp" data-field-name="max_hp" type="number" min="0"><div class="field-error" data-field-error="max_hp"></div></div>
      <div class="field"><label>Current HP (optional)</label><input id="mdl_create_player_current_hp" data-field-name="current_hp" type="number" min="0"><div class="field-error" data-field-error="current_hp"></div></div>
      <div class="field"><label>DEX (optional)</label><input id="mdl_create_player_dex" data-field-name="dex" type="number"><div class="field-error" data-field-error="dex"></div></div>
      <div class="field"><label>Initiative Bonus (optional)</label><input id="mdl_create_player_bonus" data-field-name="initiative_bonus" type="number"><div class="field-error" data-field-error="initiative_bonus"></div></div>
      <div class="field"><label>Notes</label><textarea id="mdl_create_player_notes" data-field-name="notes" placeholder="Class, traits, reminders"></textarea><div class="field-error" data-field-error="notes"></div></div>
    `,
    markdownPlaceholder: `---\nname: Aramil\nac: 15\nmax_hp: 28\ncurrent_hp: 28\ndex: 16\ninitiative_bonus: 3\nnotes: Keeps Bless ready.\n---`,
    submitAction: "submitCreatePlayerTemplate()",
  });
}

function showTemplateModal({ title, modeKey, formContent, markdownPlaceholder, submitAction }) {
  showModal(`
    <h3>${title}</h3>
    <div class="template-tabs">
      <button id="tab_${modeKey}_form" class="primary" onclick="setTemplateModalMode('${modeKey}', 'form')">Form</button>
      <button id="tab_${modeKey}_markdown" onclick="setTemplateModalMode('${modeKey}', 'markdown')">Markdown</button>
    </div>
    <div id="panel_${modeKey}_form">${formContent}</div>
    <div id="panel_${modeKey}_markdown" class="hidden">
      <div class="field">
        <label>Markdown</label>
        <textarea id="mdl_${modeKey}_markdown" data-field-name="markdown" placeholder="${escapeHtml(markdownPlaceholder)}"></textarea>
        <div class="field-error" data-field-error="markdown"></div>
      </div>
    </div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="${submitAction}">Save</button>
    </div>
  `);
  setTemplateModalMode(modeKey, "form");
}

function setTemplateModalMode(modeKey, mode) {
  const isForm = mode === "form";
  document.getElementById(`panel_${modeKey}_form`).classList.toggle("hidden", !isForm);
  document.getElementById(`panel_${modeKey}_markdown`).classList.toggle("hidden", isForm);
  document.getElementById(`tab_${modeKey}_form`).classList.toggle("primary", isForm);
  document.getElementById(`tab_${modeKey}_markdown`).classList.toggle("primary", !isForm);
}

function getTemplateModalMode(modeKey) {
  return document.getElementById(`panel_${modeKey}_form`).classList.contains("hidden") ? "markdown" : "form";
}

async function submitCreateNpcTemplate() {
  clearModalValidation();
  const mode = getTemplateModalMode("npcTemplate");
  const body = mode === "markdown"
    ? { markdown: document.getElementById("mdl_npcTemplate_markdown").value.trim() }
    : {
        name: document.getElementById("mdl_create_npc_name").value.trim(),
        ac: parseOptionalInteger("mdl_create_npc_ac"),
        hp: parseOptionalInteger("mdl_create_npc_hp"),
        dex: parseOptionalInteger("mdl_create_npc_dex"),
        initiative_bonus: parseOptionalInteger("mdl_create_npc_bonus"),
        tags: document.getElementById("mdl_create_npc_tags").value.trim(),
        notes: document.getElementById("mdl_create_npc_notes").value.trim(),
      };
  const response = await api("POST", "/api/save-npc-template", body);
  handleTemplateSaveResponse(response);
}

async function submitCreatePlayerTemplate() {
  clearModalValidation();
  const mode = getTemplateModalMode("playerTemplate");
  const body = mode === "markdown"
    ? { markdown: document.getElementById("mdl_playerTemplate_markdown").value.trim() }
    : {
        name: document.getElementById("mdl_create_player_name").value.trim(),
        ac: parseOptionalInteger("mdl_create_player_ac"),
        max_hp: parseOptionalInteger("mdl_create_player_max_hp"),
        current_hp: parseOptionalInteger("mdl_create_player_current_hp"),
        dex: parseOptionalInteger("mdl_create_player_dex"),
        initiative_bonus: parseOptionalInteger("mdl_create_player_bonus"),
        notes: document.getElementById("mdl_create_player_notes").value.trim(),
      };
  const response = await api("POST", "/api/save-player-template", body);
  handleTemplateSaveResponse(response);
}

function showModal(html) {
  let overlay = document.getElementById("modalOverlay");
  if (!overlay) {
    overlay = document.createElement("div");
    overlay.id = "modalOverlay";
    overlay.className = "modal-overlay";
    overlay.addEventListener("click", (event) => {
      if (event.target === overlay) {
        closeModal();
      }
    });
    document.body.appendChild(overlay);
  }
  overlay.innerHTML = "<div class=\"modal\">" + html + "</div>";
  overlay.classList.remove("hidden");
}

function closeModal() {
  const overlay = document.getElementById("modalOverlay");
  if (overlay) overlay.classList.add("hidden");
}

load();
