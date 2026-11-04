# VRCFaceTracking Bridge

A driver that runs [VRCFaceTracking](https://github.com/benaclejames/VRCFaceTracking)
tracking modules (C# `ExtTrackingModule` assemblies) in VRCFaceTrackingGO.
Each bundled module appears as its own module on the Modules page; the
bridge itself is listed under Drivers.

This is a prototype. It needs the .NET 9 runtime.

## How it works

The bridge is a .NET program shipped as a module archive
(`VRCFTBridge.<os>.<arch>.vfgmodf`). Its modules live in `modules/<name>/`,
laid out as VRCFaceTracking installs them: the module assembly and its
files plus VRCFaceTracking's `module.json`.

- `--info` lists every module that loads. The ID is `VRCFT-<ModuleId>` from
  `module.json` (or `VRCFT-<folder>` without one); name, author and so on
  come from `module.json`, and the supported tracking kinds from the
  module's `Supported`.
- `--source=<id>` loads that one module into its own
  `AssemblyLoadContext` and serves it. As in VRCFaceTracking, its `Update`
  runs in a loop on its own thread; each host Update returns the latest
  `UnifiedTracking.Data` at once. An exception in `Update` ends the process
  and the host restarts the module.

Modules are built against `VRCFaceTracking.Core` / `VRCFaceTracking.SDK`.
`Upstream/` builds assemblies with those names from VRCFaceTracking's
sources, limited to what modules use: the SDK as is, and from the core the
tracking data types, `UnifiedTracking.Data` / `EyeImageData` /
`LipImageData`, `Utils` and the type forwards older modules rely on.

## Limitations

- Modules that use other parts of VRCFaceTracking (the parameter mutator,
  OSC, settings, its UI) fail to load or to run.
- Module icons are not shown: VRCFaceTracking modules only set them in
  `Initialize`.
- Modules are bundled when the bridge is built (`BundledModule` items in
  `VRCFTBridge.csproj`, see `Samples/DemoModule`). A module folder copied
  into `modules/` of an installed bridge shows up after reloading the
  modules.
- Modules built for a newer .NET than the bridge's runtime cannot load.
- The installer does not include the bridge; install the `.vfgmodf` from
  the Modules page.

## Building

`go run ./cmd/build` publishes it with the .NET SDK (skipped when `dotnet`
is not installed) into `build/bin/modules/VRCFTBridge.<os>.<arch>.vfgmodf`.

## Credits

`Upstream/` contains code from VRCFaceTracking, Copyright 2024 benaclejames,
licensed under the Apache License 2.0; see [Upstream/LICENSE](Upstream/LICENSE).
