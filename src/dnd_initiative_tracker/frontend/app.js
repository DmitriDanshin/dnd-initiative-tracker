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
    state = await api("GET", "/api/state");
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
}

function renderHome(app) {
  setTitle("Home");
  const encounters = state.encounters || [];
  const npcTemplates = (state.npc_templates || []).map((template) => escapeHtml(template.name)).join(", ") || "-";
  const playerTemplates = (state.player_templates || []).map((name) => escapeHtml(name)).join(", ") || "-";
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
      <button onclick="showCreateNpcTemplate()">Add NPC Template</button>
      <button onclick="showCreatePlayerTemplate()">Add Player Template</button>
    </div>
    <div class="template-section">
      <h2>Templates</h2>
      <div class="template-list">NPC: ${npcTemplates}</div>
      <div class="template-list">Players: ${playerTemplates}</div>
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
      <button onclick="saveEncounter()">Save</button>
      <button onclick="goHome()">Back</button>
    </div>
    <table>
      <thead><tr><th>Init</th><th>Token</th><th>Name</th><th>HP</th><th>AC</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>
    ${detail}`;
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
  const options = templates.map((template) => {
    return "<option value=\"" + template.name + "\" data-hp=\"" + template.hp + "\" data-ac=\"" + template.ac + "\">" + template.name + "</option>";
  }).join("");
  const first = templates[0] || { hp: "", ac: "" };
  showModal(`
    <h3>Add NPC</h3>
    <div class="field"><label>NPC Name</label>
      <select id="mdl_npc_name" onchange="onNpcSelect()">${options}</select></div>
    <div class="field"><label>HP</label><input id="mdl_npc_hp" type="number" value="${first.hp}" min="1"></div>
    <div class="field"><label>AC</label><input id="mdl_npc_ac" type="number" value="${first.ac}" min="0"></div>
    <div class="field"><label>Count</label><input id="mdl_npc_count" type="number" value="1" min="1"></div>
    <div class="field"><label>Token Labels (comma-separated, optional)</label><input id="mdl_npc_labels" placeholder="B1,B2,B3"></div>
    <div class="btn-group">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="submitAddNpc()">Add</button>
    </div>
  `);
}

function onNpcSelect() {
  const select = document.getElementById("mdl_npc_name");
  const option = select.options[select.selectedIndex];
  document.getElementById("mdl_npc_hp").value = option.dataset.hp || "";
  document.getElementById("mdl_npc_ac").value = option.dataset.ac || "";
}

async function submitAddNpc() {
  const name = document.getElementById("mdl_npc_name").value;
  const count = parseInt(document.getElementById("mdl_npc_count").value, 10) || 1;
  const labels = document.getElementById("mdl_npc_labels").value;
  const hp = parseOptionalInteger("mdl_npc_hp");
  const ac = parseOptionalInteger("mdl_npc_ac");
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
