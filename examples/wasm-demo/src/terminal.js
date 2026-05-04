// xterm.js terminal setup and I/O wiring to the shell emulator.

import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";

/**
 * Create and mount an xterm.js terminal.
 *
 * @param {HTMLElement} container - DOM element to mount the terminal into
 * @param {import("./shell.js").Shell} shell - Shell instance to wire I/O to
 * @returns {{ term: Terminal, fit: FitAddon }}
 */
export function createTerminal(container, shell) {
  const term = new Terminal({
    cursorBlink: true,
    cursorStyle: "bar",
    fontSize: 14,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, monospace",
    theme: {
      background: "#0d1117",
      foreground: "#c9d1d9",
      cursor: "#58a6ff",
      selectionBackground: "#264f78",
      black: "#0d1117",
      red: "#ff7b72",
      green: "#7ee787",
      yellow: "#d29922",
      blue: "#58a6ff",
      magenta: "#bc8cff",
      cyan: "#39c5cf",
      white: "#c9d1d9",
      brightBlack: "#484f58",
      brightRed: "#ffa198",
      brightGreen: "#56d364",
      brightYellow: "#e3b341",
      brightBlue: "#79c0ff",
      brightMagenta: "#d2a8ff",
      brightCyan: "#56d4dd",
      brightWhite: "#f0f6fc",
    },
    allowProposedApi: true,
  });

  const fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon());

  term.open(container);
  fitAddon.fit();

  // Wire terminal input to shell
  term.onData((data) => shell.onData(data));

  // Refit on window resize
  const resizeObserver = new ResizeObserver(() => {
    fitAddon.fit();
  });
  resizeObserver.observe(container);

  return { term, fit: fitAddon };
}

/**
 * Write a welcome banner to the terminal.
 * @param {Terminal} term
 */
export function writeWelcome(term) {
  term.writeln("\x1b[1;36mstructcli\x1b[0m \x1b[2mWASM Playground\x1b[0m");
  term.writeln("\x1b[2mType commands below. Try --help to start.\x1b[0m");
  term.writeln("");
}
