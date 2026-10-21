# Rust SDK

| **English** | [日本語](README.ja.md) |

`vrcftgo-sdk` is the module side of the [module protocol](../README.md) for
Rust, built on tokio. [`serve`](src/serve.rs) handles the command line
(`--info`, `--vfgmod=true`), the wire format, the async runtime and teardown
when the host goes away.

## Install

```toml
[dependencies]
vrcftgo-sdk = { git = "https://github.com/SayukiDev/VRCFaceTrackingGO" }
tokio = { version = "1", features = ["time"] }   # only if you use tokio yourself
serde_json = "1"                                 # for log fields
```

## Write a module

```rust
use vrcftgo_sdk::{Logger, Metadata, Module, Result, Support, TrackingData, UnifiedExpression};

struct MyModule;

impl Module for MyModule {
    // Stable: the settings key and the installed file name.
    fn id(&self) -> &str { "MyModule" }

    fn metadata(&self) -> Metadata {
        Metadata {
            name: "My Module".into(),
            description: "Provides ...".into(),
            icons: vec![include_bytes!("icon.png").to_vec()],
            platforms: vec!["windows".into(), "linux".into()],
            ..Default::default()
        }
    }

    fn supported(&self) -> Support { Support::BOTH }

    // Open the device. `avail` is what the host still needs.
    async fn initialize(&mut self, logger: &Logger, avail: Support) -> Result<Support> {
        logger.info("Started", serde_json::Value::Null);
        Ok(avail)
    }

    // Write the current state; `data` keeps its values between calls.
    async fn update(&mut self, data: &mut TrackingData) -> Result<()> {
        data.set_shape(UnifiedExpression::JawOpen, 0.5);
        Ok(())
    }

    // Release the device.
    async fn teardown(&mut self) -> Result<()> { Ok(()) }
}

fn main() {
    vrcftgo_sdk::serve(MyModule);
}
```

- `initialize` returns the subset of `avail` the module will provide;
  `serve` drops anything outside it.
- `update` should wait a few milliseconds at most for fresh data. An error
  only skips the frame.
- Log with the `Logger` passed to `initialize`; it writes JSON lines to
  stderr, which the host forwards to its log. **Never print to stdout**: it
  carries the protocol.
- A panic ends the process; the host restarts the module.

Build it and install `<id>.vfgmod` from the Modules page:

```sh
cargo build --release
# copy target/release/<binary>(.exe) to MyModule.vfgmod
```

## Example

[`examples/demo.rs`](examples/demo.rs) is a demo module that needs no
hardware: it moves the jaw, smile, eyes and blinks on its own.

```sh
cargo build --release --example demo
# copy target/release/examples/demo(.exe) to DemoRust.vfgmod
```

## License

AGPL-3.0, like the rest of VRCFaceTrackingGO. A module that only implements
the protocol without using this crate does not have to use it.
