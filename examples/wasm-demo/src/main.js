// App bootstrap: wires together terminal, shell, sidebar, and example switching.

import "@xterm/xterm/css/xterm.css";
import "./style.css";
import { Shell } from "./shell.js";
import { createTerminal, writeWelcome } from "./terminal.js";
import { examples } from "./examples.js";

const WASM_BASE = import.meta.env.BASE_URL + "wasm/";

// --- Shell setup ---

let currentExample = examples[0];

const shell = new Shell({
  write: (text) => term.write(text),
  getPrompt: () => `\x1b[1;32m${currentExample.prompt}\x1b[0m\x1b[2m$\x1b[0m `,
});

// --- Terminal setup ---

const container = document.getElementById("terminal-container");
const { term, fit } = createTerminal(container, shell);

// --- Sidebar: example selector ---

const select = document.getElementById("example-select");
const descriptionEl = document.getElementById("example-description");
const suggestionsEl = document.getElementById("suggestions");

for (const ex of examples) {
  const opt = document.createElement("option");
  opt.value = ex.id;
  opt.textContent = ex.name;
  select.appendChild(opt);
}

select.addEventListener("change", () => {
  const ex = examples.find((e) => e.id === select.value);
  if (ex) switchExample(ex);
});

// --- Sidebar: suggestions ---

function renderSuggestions(ex) {
  suggestionsEl.innerHTML = "";
  for (const cmd of ex.suggestions) {
    const btn = document.createElement("button");
    btn.className = "suggestion";
    btn.textContent = cmd;
    btn.title = `Run: ${ex.prompt} ${cmd}`;
    btn.addEventListener("click", () => {
      shell.executeCommand(cmd);
      term.focus();
    });
    suggestionsEl.appendChild(btn);
  }
}

// --- Example switching ---

async function switchExample(ex) {
  currentExample = ex;
  descriptionEl.textContent = ex.description;
  renderSuggestions(ex);

  // Clear terminal and show welcome
  term.clear();
  writeWelcome(term);

  // Load the WASM binary
  const wasmUrl = WASM_BASE + ex.binary;
  await shell.setBinary(wasmUrl, ex.prompt);
  term.focus();
}

// --- Mobile sidebar toggle ---

const sidebar = document.getElementById("sidebar");
const toggleBtn = document.createElement("button");
toggleBtn.id = "sidebar-toggle";
toggleBtn.textContent = "☰";
toggleBtn.title = "Toggle sidebar";
toggleBtn.addEventListener("click", () => {
  sidebar.classList.toggle("open");
});
document.getElementById("app").prepend(toggleBtn);

// Close sidebar when clicking terminal on mobile
container.addEventListener("click", () => {
  if (window.innerWidth < 768) {
    sidebar.classList.remove("open");
  }
});

// --- Init ---

switchExample(examples[0]);
