# VRCFaceTrackingGO

| **English** | [日本語](README.ja.md) |

A Go reimplementation of [VRCFaceTracking](https://github.com/benaclejames/VRCFaceTracking) for Windows, built with [Wails](https://wails.io) and Vue.

It receives eye and face tracking from tracking modules and sends it to VRChat over OSC, either as Unified Expressions (v2) avatar parameters, through VRChat's native eye tracking, or both.

## Features

- Unified Expressions (v2) and head parameters, with OSCQuery avatar detection
- VRChat native eye tracking output
- Mutations: parameter adjustment, Unified correctors and the data filter
- Modules run as separate processes and restart automatically when they crash
- Module priority, per module eye / expression selection and manual assignment
- Material Design 3 UI with light / dark themes and a configurable accent colour
- English and Japanese

## Modules

| Module | Hardware | Based on |
|---|---|---|
| [GXRFT](modules/GXRFT) | Samsung Galaxy XR through [gxr-root-ft](https://github.com/compdoge/gxr-root-ft) | [VRCFT-ALVR-UE](https://github.com/compdoge/VRCFT-ALVR-UE) |
| [VirtualDesktop](modules/VirtualDesktop) | Quest Pro etc. through Virtual Desktop | [VirtualDesktop.VRCFaceTracking](https://github.com/guygodin/VirtualDesktop.VRCFaceTracking) |
| [VRCFTPicoModule](modules/VRCFTPicoModule) | PICO 4 Pro / Enterprise | [VRCFTPicoModule](https://github.com/lonelyicer/VRCFTPicoModule) |

Modules are standalone programs (`.vfgmod`). The installer can install the bundled ones; others are installed from the Modules page. See [docs/module-protocol.md](docs/module-protocol.md) to write your own.

## Requirements

- Windows 10 or later
- [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (installed by the installer when missing)

## Build

Prerequisites: Go 1.25+, Node.js, the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation) and [NSIS](https://nsis.sourceforge.io) for the installer.

```sh
go run ./cmd/build
```

This builds every module into `build/bin/modules`, the application and the installer into `build/bin`. Flags:

| Flag | Default | |
|---|---|---|
| `-app` | `true` | Build the application after the modules |
| `-nsis` | `true` | Also build the installer |
| `-install` | `false` | Copy the modules into `%APPDATA%\VRCFaceTrackingGO\Modules` |

For development, `wails dev` runs the application with hot reload.

## License

VRCFaceTrackingGO, including the modules in [`modules`](modules), is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0).

- You may use, modify and redistribute it, also commercially.
- If you distribute it, or a modified version, you must make the complete source code available under the AGPL-3.0, keep the copyright and license notices, and state your changes.
- If you let others use a modified version over a network, you must offer them its source code as well.
- It comes without any warranty.

This is a summary; the [license text](LICENSE) is authoritative.

Modules are separate programs that talk to the application only through the [module protocol](docs/module-protocol.md). A module that you write yourself and that only implements this protocol, without using code from this repository, does not have to use the AGPL-3.0.

Parts of the code are ported from projects under the Apache License 2.0 and the MIT License. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for them and for the licenses of the bundled libraries.
