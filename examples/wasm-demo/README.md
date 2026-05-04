# WASM Demo

Interactive browser playground for structcli examples. Runs real `wasip1/wasm` binaries in the browser using a minimal WASI shim and xterm.js.

**Live demo**: [leodido.github.io/structcli/](https://leodido.github.io/structcli/)

## Local development

Prerequisites: Go 1.24+, Node.js 22+.

```bash
# From the repo root — build all example WASM binaries
bash examples/wasm-demo/build-wasm.sh

# Start the dev server
cd examples/wasm-demo
npm install
VITE_BASE=/ npm run dev
```

Open `http://localhost:5173` in your browser.

## How it works

Each user command spawns a fresh `WebAssembly.Instance` from a cached compiled module. The WASI shim (~200 LOC, zero dependencies) provides `args_get`, `fd_write`, `proc_exit`, and the other ~15 imports Go's wasip1 runtime requires. Output is piped to xterm.js in real time.

The same `.wasm` binaries work in wasmtime, wazero, and the browser — no Go code changes needed.

## Production build

```bash
cd examples/wasm-demo
npm run build
# Output in dist/ — serve with any static file server
```

CI builds and deploys to GitHub Pages automatically on push to `main`.
