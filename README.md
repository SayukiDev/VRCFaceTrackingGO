# VRCFaceTrackingGO

| **English** | [日本語](README.ja.md) |

A Go reimplementation of [VRCFaceTracking](https://github.com/benaclejames/VRCFaceTracking) for Windows, built with [Wails](https://wails.io) and Vue. Linux and macOS are supported with [some limits](#linux).

It receives eye and face tracking from tracking modules and sends it to VRChat over OSC, either as Unified Expressions (v2) avatar parameters, through VRChat's native eye tracking, or both.

This is not a faithful port of the original VRCFaceTracking. It takes the original logic as a reference and implements it in its own way, so behaviour, settings and the module system differ from the original.

## Features

- Unified Expressions (v2) and head parameters
- OSCQuery or plain OSC: OSCQuery finds VRChat and sends only the parameters of the current avatar; plain OSC uses fixed ports and sends every parameter when the avatar is unknown
- VRChat native eye tracking output
- Mutations: parameter adjustment, Unified correctors and the data filter
- Modules run as separate processes and restart automatically when they crash
- Modules can be written in any language ([SDKs for Go and Rust](SDK/README.md))
- Module priority, per module eye / expression selection and manual assignment
- Material Design 3 UI with light / dark themes and a configurable accent colour
- English and Japanese
- Multi-platform: Windows, Linux and macOS ([with limits](#linux) outside Windows)

## Modules

| Module | Hardware | Platforms | Based on |
|---|---|---|---|
| [ALVR](modules/ALVR) | Headsets with face tracking through [ALVR](https://github.com/alvr-org/ALVR) | Windows, Linux | [VRCFT-ALVR](https://github.com/alvr-org/VRCFT-ALVR) |
| [GXRFT](modules/GXRFT) | Samsung Galaxy XR through [gxr-root-ft](https://github.com/compdoge/gxr-root-ft) | Windows, Linux, macOS | [VRCFT-ALVR-UE](https://github.com/compdoge/VRCFT-ALVR-UE) |
| [SteamLink](modules/SteamLink) | Meta Quest through Steam Link | Windows | [LinkFT](https://github.com/ykeara/LinkFT) |
| [VirtualDesktop](modules/VirtualDesktop) | Quest Pro etc. through Virtual Desktop | Windows | [VirtualDesktop.VRCFaceTracking](https://github.com/guygodin/VirtualDesktop.VRCFaceTracking) |
| [VRCFTPicoModule](modules/VRCFTPicoModule) | PICO 4 Pro / Enterprise | Windows | [VRCFTPicoModule](https://github.com/lonelyicer/VRCFTPicoModule) |

Modules are standalone programs (`.vfgmod`). The installer can install the bundled ones; others are installed from the Modules page. To write your own, see the [module SDK](SDK/README.md) for Go and Rust and the protocol.

## Requirements

### Windows

- Windows 10 or later (x64)
- [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (installed by the installer when missing)

### Linux

- x64
- GTK 3 and WebKitGTK 4.1 (`libgtk-3-0t64` and `libwebkit2gtk-4.1-0` on Ubuntu 24.04 or later)
- Limits:
  - Only modules that support Linux can be used.
  - VRChat's OSC setting is not turned on automatically, and the OSC mode cannot read avatar config files, so it sends every parameter. Use the OSCQuery mode to send only the current avatar's parameters.
  - The system accent colour option is not available.

### macOS

- A Mac with Apple silicon (arm64), macOS 11 or later
- The app is not signed; open it with right click → Open the first time.
- Limits:
  - Only modules that support macOS can be used.
  - VRChat's OSC setting is not turned on automatically, and the OSC mode cannot read avatar config files, so it sends every parameter. Use the OSCQuery mode to send only the current avatar's parameters.
  - The system accent colour option is not available.

## Build

Prerequisites: Go 1.25+, Node.js, the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation) and [NSIS](https://nsis.sourceforge.io) for the installer.

```sh
go run ./cmd/build
```

This builds every module whose `metadata.json` lists the current OS into `build/bin/modules`, and the application and the installer (Windows only) into `build/bin`. Flags:

| Flag | Default | |
|---|---|---|
| `-app` | `true` | Build the application after the modules |
| `-nsis` | `true` on Windows | Also build the installer (Windows only) |
| `-install` | `false` | Copy the modules into the module folder (`%APPDATA%\VRCFaceTrackingGO\Modules` on Windows, see [SDK/README.md](SDK/README.md#module-program) for other OSes) |
| `-tags` | | Build tags for `wails build`, e.g. `webkit2_41` for WebKitGTK 4.1 on Linux |

For development, `wails dev` runs the application with hot reload.

Publishing a GitHub release runs [`.github/workflows/release.yml`](.github/workflows/release.yml), which builds on Windows, Linux and macOS and attaches a ZIP for each to the release.

## Use of LLMs

This project is developed by one person for personal use, with limited time and energy, so LLMs (large language models) are used extensively in its development. The code has been checked once, but if you have concerns, reviewing it yourself is recommended.

## License

VRCFaceTrackingGO, including the modules in [`modules`](modules), is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0).

- You may use, modify and redistribute it, also commercially.
- If you distribute it, or a modified version, you must make the complete source code available under the AGPL-3.0, keep the copyright and license notices, and state your changes.
- If you let others use a modified version over a network, you must offer them its source code as well.
- It comes without any warranty.

This is a summary; the [license text](LICENSE) is authoritative.

Modules are separate programs that talk to the application only through the [module protocol](SDK/README.md). A module that you write yourself and that only implements this protocol, without using code from this repository, does not have to use the AGPL-3.0.

Parts of the code are ported from projects under the Apache License 2.0 and the MIT License. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for them and for the licenses of the bundled libraries.
