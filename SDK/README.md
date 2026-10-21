# Module SDK

| **English** | [日本語](README.ja.md) |

Tracking modules are separate programs. VRCFaceTrackingGO starts each
enabled module as a child process and talks to it over the child's
standard streams with a small binary request/response protocol.

The SDKs implement the module side of the protocol, so a module only has
to provide its tracking data:

| Language | SDK | Example |
|---|---|---|
| Go | [`pkg/module`](../pkg/module) ([guide](Golang/README.md)) | [`Golang/example`](Golang/example) |
| Rust | [`Rust`](Rust) (crate `vrcftgo-sdk`, tokio) | [`Rust/examples/demo.rs`](Rust/examples/demo.rs) |

The rest of this document describes the protocol itself, for writing a
module without an SDK. The reference implementation is
[`pkg/module/plugin`](../pkg/module/plugin): `Serve` is the module side,
`Client` the host side.

## Module program

A module program is an executable named `<id>.<os>.<arch>.vfgmod` installed
in the `Modules` folder of the application's data directory:

| OS | Module folder |
|---|---|
| Windows | `%APPDATA%\VRCFaceTrackingGO\Modules` |
| Linux | `~/.config/VRCFaceTrackingGO/Modules` |
| macOS | `~/Library/Application Support/VRCFaceTrackingGO/Modules` |

The host recognises it only by the file name suffix, so a module may be
written in any language. `<os>` and `<arch>` name the platform the
program was built for, using Go's `GOOS` and `GOARCH` values; the host
only recognises and installs programs for its own platform:

| Platform | Suffix |
|---|---|
| Windows x64 | `.windows.amd64.vfgmod` |
| Windows ARM64 | `.windows.arm64.vfgmod` |
| Linux x64 | `.linux.amd64.vfgmod` |
| Linux ARM64 | `.linux.arm64.vfgmod` |
| macOS Intel | `.darwin.amd64.vfgmod` |
| macOS Apple silicon | `.darwin.arm64.vfgmod` |

Outside Windows the file must be executable; the Modules page installs it
that way.

It must handle these command lines:

| Command line | Behaviour |
|---|---|
| `<prog> --info` | Print the module description as JSON to stdout and exit with code 0. |
| `<prog> --vfgmod=true` | Serve the host on stdin/stdout until stdin is closed. |
| anything else | Refuse to run (e.g. print a hint to stderr and exit with a non-zero code). |

The last rule keeps a module that is opened by accident from touching any
device.

### Description (`--info`)

```json
{
  "id": "VirtualDesktop",
  "metadata": {
    "name": "Virtual Desktop",
    "description": "Provides face, eye and tongue tracking ...",
    "author": "SayukiDev",
    "version": "1.0.0",
    "pageUrl": "",
    "usageInstructions": "Make sure to install ...",
    "icons": ["<base64 PNG>"],
    "platforms": ["windows"]
  },
  "supported": { "eye": true, "expression": true }
}
```

- `id` is required and must match `^[A-Za-z0-9][A-Za-z0-9._-]*$`. It is
  the settings key and the installed file name, so it must never change.
  Modules with a duplicate ID are skipped.
- `metadata` is shown on the Modules page. `icons` are PNG images encoded
  as base64 strings.
- `platforms` lists the operating systems (Go `GOOS` values: `windows`,
  `linux`, `darwin`) the module is built and distributed for. It is
  optional; an empty list means every platform. `go run ./cmd/build` skips
  modules that do not list the current OS.
- `supported` states what the module can provide before it is initialised.
- The host waits at most 5 seconds for `--info`. It re-reads the
  description only when the file's size or modification time changes.

## Process lifecycle

1. The host starts `<prog> --vfgmod=true` with stdin, stdout and stderr
   connected to pipes (and, on Windows, no console window).
2. It sends **Initialize** once, with a 15 second timeout (configurable in
   the settings).
3. While the module is in use, it sends **Update** every 10 ms
   (configurable). Each Update must answer within 5 seconds.
4. When the module is stopped, the host sends **Teardown** (3 second
   timeout) and closes stdin. The module must exit on EOF; after 3 more
   seconds it is killed.

Modules end together with the host even if the host crashes: on Windows
the host puts every module in a kill-on-close job object, and on Linux the
kernel kills the module when its parent dies (`PR_SET_PDEATHSIG`). On
macOS there is no such facility; `plugin.Serve` exits when its parent
process changes, and modules in other languages should at least exit when
stdin reaches EOF.

If a module exits or an Update times out, the host restarts the program
and initialises it again with the support it had before. It waits 1 s
before the first restart and doubles the delay each time; after more than
3 failures within one minute it gives up. A restarted module that now
provides a different support is treated as a failure.

## Wire format

The host sends one request and waits for its response before sending the
next. All integers and floats are **little endian**.

### Requests

| Op | Byte | Payload |
|---|---|---|
| Initialize | `'I'` (0x49) | support (1 byte) |
| Update | `'U'` (0x55) | none |
| Teardown | `'T'` (0x54) | none |

An unknown op is a protocol error; the module should exit.

### Responses

```
success: 0x00  payload
error:   0x01  length (uint32)  message (length bytes, UTF-8)
```

Error messages are limited to 64 KiB. An Initialize error means the module
failed to start, and the host ends the process. An Update error only skips
that frame: such errors are expected (for example "no device") and are
logged at most once per 10 seconds while they repeat. A Teardown error is
logged.

| Request | Success payload |
|---|---|
| Initialize | granted support (1 byte) |
| Update | tracking data (416 bytes) |
| Teardown | none |

### Support byte

| Bit | Meaning |
|---|---|
| 0 (0x01) | eye tracking |
| 1 (0x02) | expression tracking |

In Initialize the host sends the kinds of tracking it still needs from
this module (another module may already provide the rest). The module
answers with what it will actually provide, which must be a subset.

### Tracking data

104 `float32` values (416 bytes) in this order:

| Offset | Count | Field |
|---|---|---|
| 0 | 4 | left eye: gaze X, gaze Y, pupil diameter (mm), openness |
| 16 | 4 | right eye: gaze X, gaze Y, pupil diameter (mm), openness |
| 32 | 2 | max dilation, min dilation |
| 40 | 88 | Unified Expressions shape weights |
| 392 | 6 | head: yaw, pitch, roll, position X, Y, Z |

- Gaze is a direction in tangent space: X = tan(yaw), right positive;
  Y = tan(pitch), up positive.
- Openness and shape weights are 0 (closed / neutral) to 1.
- Max and min dilation are the pupil diameter range seen so far; the host
  uses them to normalise the pupil size.
- Shapes are in the order of `tracking.UnifiedExpression` in
  [`pkg/tracking/expressions.go`](../pkg/tracking/expressions.go)
  (`EyeSquintRight`, `EyeSquintLeft`, `EyeWideRight`, ...), which follows
  the Unified Expressions specification of VRCFaceTracking.

The data is the module's complete current state, not a delta. Fields the
module does not track keep their initial values: openness 1, min dilation
999, everything else 0.

## Logging

stdout is reserved for the protocol; a module must never print anything
else there. Logs go to **stderr** as JSON lines in zap's format:

```json
{"level":"info","ts":1729500000.0,"logger":"PICO","msg":"Using port","port":29765,"legacy":false}
```

The host re-logs `level`, `logger`, `msg` and every extra field. Levels
above `error` are lowered to `error`, so a module cannot make the host
panic or exit. Lines that are not JSON (for example a crash trace) are
logged as warnings.

## Writing a module in Go or Rust

Use the SDKs: see the [Go guide](Golang/README.md) and the
[Rust crate](Rust/README.md).

## Writing a module in another language

Any language works as long as the result is an executable for the OS the
application runs on. The host starts the `.vfgmod` file directly, so it
must be a native program (PE on Windows, ELF on Linux, Mach-O on macOS),
not a script: bundle scripts into an executable (for example with
PyInstaller) and rename it to `<id>.<os>.<arch>.vfgmod`. On Windows the host already
hides the console window, so a console program is fine. Set `platforms` in
the description to the operating systems you build it for.

Checklist:

- Accept exactly `--info` and `--vfgmod=true`, and refuse anything else.
- Put stdin and stdout in **binary mode** so no newline translation
  happens on Windows (`sys.stdin.buffer` in Python,
  `_setmode(_fileno(stdin), _O_BINARY)` in C, `Console.OpenStandardInput()`
  in .NET).
- **Flush** stdout after every response; the host blocks until it arrives.
- Never write anything but responses to stdout. Send logs and any library
  output to stderr.
- Keep one tracking data buffer for the whole session, start it with the
  initial values above, and update it in place on every Update.
- Return from Update quickly: wait for fresh data for a few milliseconds at
  most, then answer with the current state. 5 seconds without an answer
  counts as a hang.
- Release the device on Teardown, and exit when stdin reaches EOF.

### Example (Python)

A minimal module that provides expression tracking only:

```python
import json
import struct
import sys

EXPRESSIONS = 88
FLOATS = 10 + EXPRESSIONS + 6          # eyes, shapes, head: 416 bytes
SHAPES = 10                            # float index of the first shape


def describe():
    return {
        "id": "MyModule",
        "metadata": {"name": "My Module", "description": "Example module",
                     "author": "me", "version": "1.0.0", "icons": []},
        "supported": {"eye": False, "expression": True},
    }


def log(level, msg, **fields):
    print(json.dumps({"level": level, "logger": "MyModule", "msg": msg, **fields}),
          file=sys.stderr, flush=True)


def ok(out, payload=b""):
    out.write(b"\x00" + payload)
    out.flush()


def fail(out, msg):
    b = msg.encode()
    out.write(b"\x01" + struct.pack("<I", len(b)) + b)
    out.flush()


def main():
    args = sys.argv[1:]
    if args == ["--info"]:
        json.dump(describe(), sys.stdout)
        return
    if args != ["--vfgmod=true"]:
        print("Install this module from the Modules page.", file=sys.stderr)
        sys.exit(2)

    stdin, out = sys.stdin.buffer, sys.stdout.buffer
    sys.stdout = sys.stderr            # stray prints must not corrupt the stream

    data = [0.0] * FLOATS
    data[3] = data[7] = 1.0            # left / right eye openness
    data[9] = 999.0                    # min dilation

    while op := stdin.read(1):         # b"" means the host closed stdin
        if op == b"I":
            avail = stdin.read(1)[0]
            granted = avail & 0b10     # expressions only, if still needed
            log("info", "Initialized", granted=granted)
            ok(out, bytes([granted]))
        elif op == b"U":
            # Read the device here and write the weights, e.g. JawOpen:
            # data[SHAPES + 22] = jaw_open
            ok(out, struct.pack(f"<{FLOATS}f", *data))
        elif op == b"T":
            ok(out)                    # release the device here
        else:
            log("error", "Unknown request", op=op.hex())
            sys.exit(1)


if __name__ == "__main__":
    main()
```

Build it on the platform it is for and name it for that platform:

```sh
pyinstaller --onefile --name MyModule mymodule.py
# Windows x64
copy dist\MyModule.exe MyModule.windows.amd64.vfgmod
# Linux x64
cp dist/MyModule MyModule.linux.amd64.vfgmod
# macOS Apple silicon
cp dist/MyModule MyModule.darwin.arm64.vfgmod
```

Then install it from the Modules page. Shape indices are the positions in
[`pkg/tracking/expressions.go`](../pkg/tracking/expressions.go), counted
from 0.
