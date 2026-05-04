// Shell emulator: line editing, command history, argv parsing, WASM execution.

import { createWASI } from "./wasi-shim.js";

/**
 * @typedef {object} ShellOptions
 * @property {function(string): void} write - Write text to the terminal
 * @property {function(): string} getPrompt - Returns the current prompt string
 */

export class Shell {
  /** @type {Map<string, WebAssembly.Module>} */
  #moduleCache = new Map();

  /** @type {string} */
  #line = "";

  /** @type {number} */
  #cursor = 0;

  /** @type {string[]} */
  #history = [];

  /** @type {number} */
  #historyIndex = -1;

  /** @type {string} */
  #savedLine = "";

  /** @type {boolean} */
  #running = false;

  /** @type {string|null} */
  #currentBinary = null;

  /** @type {string} */
  #binaryName = "";

  /** @type {function(string): void} */
  #write;

  /** @type {function(): string} */
  #getPrompt;

  /**
   * @param {ShellOptions} opts
   */
  constructor(opts) {
    this.#write = opts.write;
    this.#getPrompt = opts.getPrompt;
  }

  /**
   * Load (or retrieve from cache) a compiled WASM module.
   * @param {string} url - URL to the .wasm file
   * @returns {Promise<WebAssembly.Module>}
   */
  async #loadModule(url) {
    if (this.#moduleCache.has(url)) {
      return this.#moduleCache.get(url);
    }
    const module = await WebAssembly.compileStreaming(fetch(url));
    this.#moduleCache.set(url, module);
    return module;
  }

  /**
   * Set the active WASM binary. Called when the user switches examples.
   * @param {string} wasmUrl - URL to the .wasm file
   * @param {string} binaryName - Name shown in prompt / prepended to argv
   */
  async setBinary(wasmUrl, binaryName) {
    this.#currentBinary = wasmUrl;
    this.#binaryName = binaryName;
    this.#history = [];
    this.#historyIndex = -1;
    this.#line = "";
    this.#cursor = 0;

    // Pre-load and cache the module
    this.#write("\x1b[2m Loading...\x1b[0m");
    try {
      await this.#loadModule(wasmUrl);
      // Clear the "Loading..." text and show prompt
      this.#write("\r\x1b[2K");
      this.#showPrompt();
    } catch (e) {
      this.#write(`\r\x1b[2K\x1b[31mFailed to load WASM: ${e.message}\x1b[0m\r\n`);
      this.#showPrompt();
    }
  }

  /**
   * Handle raw terminal input data from xterm.js onData.
   * @param {string} data
   */
  async onData(data) {
    if (this.#running) return; // Ignore input while a command is executing

    for (let i = 0; i < data.length; i++) {
      const ch = data[i];

      // ESC sequence (arrow keys, etc.)
      if (ch === "\x1b" && data[i + 1] === "[") {
        const code = data[i + 2];
        i += 2;
        switch (code) {
          case "A": // Up arrow
            this.#historyUp();
            break;
          case "B": // Down arrow
            this.#historyDown();
            break;
          case "C": // Right arrow
            if (this.#cursor < this.#line.length) {
              this.#cursor++;
              this.#write("\x1b[C");
            }
            break;
          case "D": // Left arrow
            if (this.#cursor > 0) {
              this.#cursor--;
              this.#write("\x1b[D");
            }
            break;
          case "H": // Home
            if (this.#cursor > 0) {
              this.#write(`\x1b[${this.#cursor}D`);
              this.#cursor = 0;
            }
            break;
          case "F": // End
            if (this.#cursor < this.#line.length) {
              this.#write(`\x1b[${this.#line.length - this.#cursor}C`);
              this.#cursor = this.#line.length;
            }
            break;
          case "3": // Delete key (ESC [ 3 ~)
            if (data[i + 1] === "~") {
              i++;
              if (this.#cursor < this.#line.length) {
                this.#line =
                  this.#line.slice(0, this.#cursor) +
                  this.#line.slice(this.#cursor + 1);
                this.#redrawLine();
              }
            }
            break;
        }
        continue;
      }

      switch (ch) {
        case "\r": // Enter
          this.#write("\r\n");
          await this.#execute();
          break;

        case "\x7f": // Backspace
          if (this.#cursor > 0) {
            this.#line =
              this.#line.slice(0, this.#cursor - 1) +
              this.#line.slice(this.#cursor);
            this.#cursor--;
            this.#redrawLine();
          }
          break;

        case "\x03": // Ctrl+C
          this.#line = "";
          this.#cursor = 0;
          this.#write("^C\r\n");
          this.#showPrompt();
          break;

        case "\x0c": // Ctrl+L (clear)
          this.#write("\x1b[2J\x1b[H");
          this.#showPrompt();
          this.#write(this.#line);
          // Reposition cursor if not at end
          if (this.#cursor < this.#line.length) {
            this.#write(`\x1b[${this.#line.length - this.#cursor}D`);
          }
          break;

        case "\x01": // Ctrl+A (home)
          if (this.#cursor > 0) {
            this.#write(`\x1b[${this.#cursor}D`);
            this.#cursor = 0;
          }
          break;

        case "\x05": // Ctrl+E (end)
          if (this.#cursor < this.#line.length) {
            this.#write(`\x1b[${this.#line.length - this.#cursor}C`);
            this.#cursor = this.#line.length;
          }
          break;

        case "\x15": // Ctrl+U (clear line)
          this.#line = "";
          this.#cursor = 0;
          this.#redrawLine();
          break;

        case "\x17": // Ctrl+W (delete word backward)
          if (this.#cursor > 0) {
            let pos = this.#cursor - 1;
            while (pos > 0 && this.#line[pos - 1] === " ") pos--;
            while (pos > 0 && this.#line[pos - 1] !== " ") pos--;
            this.#line = this.#line.slice(0, pos) + this.#line.slice(this.#cursor);
            this.#cursor = pos;
            this.#redrawLine();
          }
          break;

        default:
          // Printable character
          if (ch >= " " && ch <= "~") {
            this.#line =
              this.#line.slice(0, this.#cursor) +
              ch +
              this.#line.slice(this.#cursor);
            this.#cursor++;
            this.#redrawLine();
          }
          break;
      }
    }
  }

  /**
   * Execute a command by typing it into the terminal programmatically.
   * Used by sidebar suggestion buttons.
   * @param {string} command
   */
  async executeCommand(command) {
    if (this.#running) return;
    // Set the line and show it
    this.#line = command;
    this.#cursor = command.length;
    this.#redrawLine();
    // Execute
    this.#write("\r\n");
    await this.#execute();
  }

  #showPrompt() {
    this.#write(this.#getPrompt());
  }

  #redrawLine() {
    const prompt = this.#getPrompt();
    // Move to start of line, clear it, write prompt + line, reposition cursor
    this.#write(`\r\x1b[2K${prompt}${this.#line}`);
    // Move cursor back if not at end
    const back = this.#line.length - this.#cursor;
    if (back > 0) {
      this.#write(`\x1b[${back}D`);
    }
  }

  #historyUp() {
    if (this.#history.length === 0) return;
    if (this.#historyIndex === -1) {
      this.#savedLine = this.#line;
      this.#historyIndex = this.#history.length - 1;
    } else if (this.#historyIndex > 0) {
      this.#historyIndex--;
    } else {
      return;
    }
    this.#line = this.#history[this.#historyIndex];
    this.#cursor = this.#line.length;
    this.#redrawLine();
  }

  #historyDown() {
    if (this.#historyIndex === -1) return;
    if (this.#historyIndex < this.#history.length - 1) {
      this.#historyIndex++;
      this.#line = this.#history[this.#historyIndex];
    } else {
      this.#historyIndex = -1;
      this.#line = this.#savedLine;
    }
    this.#cursor = this.#line.length;
    this.#redrawLine();
  }

  async #execute() {
    const input = this.#line.trim();
    this.#line = "";
    this.#cursor = 0;
    this.#historyIndex = -1;

    if (!input) {
      this.#showPrompt();
      return;
    }

    // Add to history (deduplicate consecutive)
    if (this.#history.length === 0 || this.#history[this.#history.length - 1] !== input) {
      this.#history.push(input);
    }

    // Handle built-in commands
    if (input === "clear") {
      this.#write("\x1b[2J\x1b[H");
      this.#showPrompt();
      return;
    }

    if (input === "help") {
      this.#write(
        "Type any command with flags. The binary name is prepended automatically.\r\n" +
        "Try: --help\r\n" +
        "Built-ins: clear, help\r\n"
      );
      this.#showPrompt();
      return;
    }

    if (!this.#currentBinary) {
      this.#write("\x1b[31mNo WASM binary loaded\x1b[0m\r\n");
      this.#showPrompt();
      return;
    }

    // Parse input into argv, prepending the binary name
    const argv = [this.#binaryName, ...this.#parseArgs(input)];

    this.#running = true;
    try {
      const module = await this.#loadModule(this.#currentBinary);

      const wasi = createWASI(
        argv,
        (text) => this.#writeOutput(text),
        (text) => this.#writeOutput(text),
      );

      const inst = await WebAssembly.instantiate(module, wasi.imports);
      const exitCode = await wasi.start(inst);

      if (exitCode !== 0) {
        this.#write(`\x1b[2mexit code: ${exitCode}\x1b[0m\r\n`);
      }
    } catch (e) {
      this.#write(`\x1b[31mError: ${e.message}\x1b[0m\r\n`);
    } finally {
      this.#running = false;
      this.#showPrompt();
    }
  }

  // Write WASM output to terminal, converting \n to \r\n for xterm.js
  #writeOutput(text) {
    this.#write(text.replace(/\n/g, "\r\n"));
  }

  // Simple argument parser that handles quoted strings
  #parseArgs(input) {
    const args = [];
    let current = "";
    let inSingle = false;
    let inDouble = false;

    for (let i = 0; i < input.length; i++) {
      const ch = input[i];

      if (ch === "'" && !inDouble) {
        inSingle = !inSingle;
      } else if (ch === '"' && !inSingle) {
        inDouble = !inDouble;
      } else if (ch === " " && !inSingle && !inDouble) {
        if (current) {
          args.push(current);
          current = "";
        }
      } else {
        current += ch;
      }
    }
    if (current) args.push(current);
    return args;
  }
}
