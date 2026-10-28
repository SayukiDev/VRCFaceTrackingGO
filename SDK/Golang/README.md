# Go SDK

| **English** | [日本語](README.ja.md) |

The Go SDK is the module side of the [module protocol](../README.md), in
two packages of this repository:

| Package | Contents |
|---|---|
| [`pkg/module/adapter`](../../pkg/module/adapter) | The `Module` interface, `Metadata`, `Support` and `LoadMetadata` |
| [`pkg/module/plugin`](../../pkg/module/plugin) | `Serve`, which runs a `Module` as a module program |
| [`pkg/tracking`](../../pkg/tracking) | `TrackingData` and the Unified Expressions shape indices |

`plugin.Serve` handles the command line (`--info`, `--vfgmod=true`), the
wire format, logging to the host and recovering from panics.

## Install

```sh
go get github.com/SayukiDev/VRCFaceTrackingGO@latest
```

## Write a module

1. Implement `adapter.Module`:

   ```go
   type Module interface {
       ID() string                 // stable, e.g. "MyModule"
       Metadata() Metadata
       Supported() Support         // what the module can provide
       Initialize(ctx context.Context, logger *zap.Logger, avail Support) (Support, error)
       Update(ctx context.Context, data *tracking.TrackingData) error
       Teardown() error
   }
   ```

   - `Initialize` opens the device. `avail` is what the host still needs;
     return the subset the module will provide.
   - `Update` writes the current state into `data`, which keeps its values
     between calls, and returns at once. Receive from the device in a
     goroutine started by `Initialize`, keep only the newest values, and
     copy them out in `Update`; reading the device in `Update` lets data
     pile up when the device sends faster than the host asks.
   - `Teardown` releases the device.
   - Log with the `logger` passed to `Initialize`; it is forwarded to the
     host's log. Never write to stdout.

2. Keep the metadata in `metadata.json` next to the code and embed it with
   its icons:

   ```json
   {
     "name": "My Module",
     "description": "Provides ...",
     "author": "me",
     "version": "1.0.0",
     "usageInstructions": "Start ... before activating this module",
     "icons": ["icon.png"],
     "platforms": ["windows", "linux"]
   }
   ```

   ```go
   //go:embed metadata.json icon.png
   var files embed.FS

   var metadata = adapter.MustLoadMetadata(files, "metadata.json")
   ```

3. Serve it from `main`:

   ```go
   func main() { plugin.Serve(New()) }
   ```

4. Build it as `<id>.<GOOS>.<GOARCH>.vfgmod` (see the
   [suffix table](../README.md#module-program)) and install it from the
   Modules page:

   ```sh
   # Windows x64
   go build -ldflags="-s -w" -o MyModule.windows.amd64.vfgmod .
   ```

Modules inside this repository live in `modules/<id>` and are built with
`go run ./cmd/build`, which also reads `platforms` and adds them to the
installer.

## Write a driver

A driver bundles several modules in one program (see
[Driver programs](../README.md#driver-programs)). Implement each module as
above and serve them together:

```go
func main() {
	plugin.ServeDriver(plugin.Driver{
		ID:       "MyDriver",
		Metadata: adapter.MustLoadMetadata(files, "metadata.json"),
	}, face.New(), eye.New())
}
```

The host starts the program with `--source=<id>` for each module it
loads; `ServeDriver` serves that one module.

## Example

[`example`](example) is a demo module that needs no hardware: it moves the
jaw, smile, eyes and blinks on its own.

```sh
go build -o Demo.windows.amd64.vfgmod ./SDK/Golang/example
```

[`example-driver`](example-driver) is a demo driver with three such modules
(eye only, face only, both).

```sh
go build -o DemoDriver.windows.amd64.vfgmod ./SDK/Golang/example-driver
```
