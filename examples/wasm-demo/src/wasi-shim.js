// Minimal WASI preview1 shim for running Go wasip1/wasm binaries in the browser.
// Implements only the ~18 imports that Go's wasip1 runtime actually calls.
// No filesystem support — just args, stdio, clock, random, and process lifecycle.

// WASI error codes
const ESUCCESS = 0;
const EBADF = 8;
const ENOSYS = 52;
const ENOENT = 44;

// WASI file types
const FILETYPE_CHARACTER_DEVICE = 2;

// WASI fd flags
const FDFLAGS_APPEND = 1;

// WASI rights (unused but required by fdstat)
const RIGHTS_ALL = BigInt("0x1FFFFFFF");

/**
 * Create a WASI shim instance for a single command execution.
 *
 * @param {string[]} argv - Command arguments (e.g., ["myapp", "--help"])
 * @param {function(string): void} onStdout - Called with decoded text from fd 1
 * @param {function(string): void} onStderr - Called with decoded text from fd 2
 * @returns {{ imports: object, start: (instance: WebAssembly.Instance) => Promise<number> }}
 */
export function createWASI(argv, onStdout, onStderr) {
  const decoder = new TextDecoder();
  let instance = null;
  let exitCode = 0;
  let exited = false;
  let resolveExit = null;

  // Encode argv into null-terminated byte arrays
  const encodedArgs = argv.map((a) => new TextEncoder().encode(a + "\0"));

  function getMemory() {
    return new DataView(instance.exports.memory.buffer);
  }

  function getMemoryU8() {
    return new Uint8Array(instance.exports.memory.buffer);
  }

  // Read iovec array and extract bytes written to a fd
  function readIovecs(iovsPtr, iovsLen) {
    const mem = getMemory();
    const u8 = getMemoryU8();
    const chunks = [];
    for (let i = 0; i < iovsLen; i++) {
      const ptr = mem.getUint32(iovsPtr + i * 8, true);
      const len = mem.getUint32(iovsPtr + i * 8 + 4, true);
      chunks.push(u8.slice(ptr, ptr + len));
    }
    return chunks;
  }

  const imports = {
    wasi_snapshot_preview1: {
      // --- Args ---

      args_sizes_get(argcPtr, argvBufSizePtr) {
        const mem = getMemory();
        mem.setUint32(argcPtr, encodedArgs.length, true);
        const totalSize = encodedArgs.reduce((sum, a) => sum + a.length, 0);
        mem.setUint32(argvBufSizePtr, totalSize, true);
        return ESUCCESS;
      },

      args_get(argvPtr, argvBufPtr) {
        const mem = getMemory();
        const u8 = getMemoryU8();
        let bufOffset = argvBufPtr;
        for (let i = 0; i < encodedArgs.length; i++) {
          mem.setUint32(argvPtr + i * 4, bufOffset, true);
          u8.set(encodedArgs[i], bufOffset);
          bufOffset += encodedArgs[i].length;
        }
        return ESUCCESS;
      },

      // --- Environment ---

      environ_sizes_get(countPtr, sizePtr) {
        const mem = getMemory();
        mem.setUint32(countPtr, 0, true);
        mem.setUint32(sizePtr, 0, true);
        return ESUCCESS;
      },

      environ_get() {
        return ESUCCESS;
      },

      // --- File descriptors (stdio only) ---

      fd_write(fd, iovsPtr, iovsLen, nwrittenPtr) {
        if (fd !== 1 && fd !== 2) return EBADF;
        const chunks = readIovecs(iovsPtr, iovsLen);
        let totalLen = 0;
        for (const chunk of chunks) {
          const text = decoder.decode(chunk, { stream: true });
          if (fd === 1) onStdout(text);
          else onStderr(text);
          totalLen += chunk.length;
        }
        getMemory().setUint32(nwrittenPtr, totalLen, true);
        return ESUCCESS;
      },

      fd_read(fd, iovsPtr, iovsLen, nreadPtr) {
        // Return EOF for stdin, error for anything else
        if (fd !== 0) return EBADF;
        getMemory().setUint32(nreadPtr, 0, true);
        return ESUCCESS;
      },

      fd_close() {
        return ESUCCESS;
      },

      fd_fdstat_get(fd, statPtr) {
        if (fd > 2) return EBADF;
        const mem = getMemory();
        // filetype: character device
        mem.setUint8(statPtr, FILETYPE_CHARACTER_DEVICE);
        // fdflags
        mem.setUint16(statPtr + 2, fd === 1 ? FDFLAGS_APPEND : 0, true);
        // rights_base
        mem.setBigUint64(statPtr + 8, RIGHTS_ALL, true);
        // rights_inheriting
        mem.setBigUint64(statPtr + 16, RIGHTS_ALL, true);
        return ESUCCESS;
      },

      fd_fdstat_set_flags() {
        return ENOSYS;
      },

      fd_filestat_get(fd, bufPtr) {
        if (fd > 2) return EBADF;
        // Zero out the filestat struct (64 bytes)
        const u8 = getMemoryU8();
        u8.fill(0, bufPtr, bufPtr + 64);
        // Set filetype to character device at offset 16
        u8[bufPtr + 16] = FILETYPE_CHARACTER_DEVICE;
        return ESUCCESS;
      },

      fd_seek(fd, offsetLo, offsetHi, whence, newOffsetPtr) {
        // No seekable fds in this shim — stdio is not seekable
        return ENOSYS;
      },

      fd_sync() {
        return ESUCCESS;
      },

      fd_readdir() {
        return ENOSYS;
      },

      // --- Filesystem preopens (none) ---

      fd_prestat_get() {
        return EBADF;
      },

      fd_prestat_dir_name() {
        return EBADF;
      },

      // --- Filesystem paths (not supported) ---

      path_open() {
        return ENOENT;
      },

      path_filestat_get() {
        return ENOENT;
      },

      // --- Clock ---

      clock_time_get(clockId, precision, timePtr) {
        const now = BigInt(Math.round(performance.now() * 1e6));
        getMemory().setBigUint64(timePtr, now, true);
        return ESUCCESS;
      },

      // --- Random ---

      random_get(bufPtr, bufLen) {
        const u8 = getMemoryU8();
        crypto.getRandomValues(u8.subarray(bufPtr, bufPtr + bufLen));
        return ESUCCESS;
      },

      // --- Scheduling ---

      poll_oneoff(inPtr, outPtr, nsubscriptions, neventsPtr) {
        // For CLI apps this is only called for sleep/timeout.
        // Return immediately with all subscriptions ready.
        const mem = getMemory();
        const u8 = getMemoryU8();
        for (let i = 0; i < nsubscriptions; i++) {
          const eventOut = outPtr + i * 32;
          u8.fill(0, eventOut, eventOut + 32);
          // userdata: copy from subscription input (first 8 bytes of each 48-byte subscription)
          const subIn = inPtr + i * 48;
          u8.copyWithin(eventOut, subIn, subIn + 8);
          // error: 0 (success) at offset 8 — already zeroed
          // type: 0 (clock) at offset 10 — already zeroed
        }
        mem.setUint32(neventsPtr, nsubscriptions, true);
        return ESUCCESS;
      },

      sched_yield() {
        return ESUCCESS;
      },

      // --- Process ---

      proc_exit(code) {
        exitCode = code;
        exited = true;
        if (resolveExit) resolveExit(code);
        // Throw to unwind the WASM stack. The caller catches this.
        throw new WASIExitError(code);
      },
    },
  };

  return {
    imports,

    /**
     * Run the WASM instance's _start export.
     * Returns a promise that resolves with the exit code.
     */
    start(inst) {
      instance = inst;
      return new Promise((resolve) => {
        resolveExit = resolve;
        try {
          instance.exports._start();
          // If _start returns normally (no proc_exit), treat as exit 0
          if (!exited) resolve(0);
        } catch (e) {
          if (e instanceof WASIExitError) {
            resolve(e.code);
          } else {
            // Re-throw unexpected errors
            throw e;
          }
        }
      });
    },
  };
}

class WASIExitError extends Error {
  constructor(code) {
    super(`WASI exit: ${code}`);
    this.code = code;
  }
}
