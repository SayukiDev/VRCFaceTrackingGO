//! SDK for VRCFaceTrackingGO tracking modules.
//!
//! A module is a separate program that the host starts and talks to over
//! stdin/stdout (see `SDK/README.md`). Implement [`Module`] and hand it to
//! [`serve`] from `main`:
//!
//! ```no_run
//! # use vrcftgo_sdk::*;
//! # struct MyModule;
//! # impl Module for MyModule {
//! #     fn id(&self) -> &str { "MyModule" }
//! #     fn metadata(&self) -> Metadata { Metadata::default() }
//! #     fn supported(&self) -> Support { Support::BOTH }
//! #     async fn initialize(&mut self, _: &Logger, avail: Support) -> Result<Support> { Ok(avail) }
//! #     async fn update(&mut self, _: &mut TrackingData) -> Result<()> { Ok(()) }
//! #     async fn teardown(&mut self) -> Result<()> { Ok(()) }
//! # }
//! fn main() {
//!     vrcftgo_sdk::serve(MyModule);
//! }
//! ```
//!
//! [`serve`] handles the command line (`--info`, `--vfgmod=true`), the wire
//! format and the async runtime. Never print to stdout: it carries the
//! protocol. Log with the [`Logger`] passed to [`Module::initialize`].

mod expressions;
mod log;
mod metadata;
mod protocol;
mod serve;
mod tracking;

pub use expressions::{EXPRESSION_COUNT, UnifiedExpression};
pub use log::Logger;
pub use metadata::{Description, Metadata, Support};
pub use serve::{serve, serve_io};
pub use tracking::{EyeData, HeadData, SingleEyeData, TrackingData, Vector2};

/// Error returned by module methods. Its message is shown to the user.
pub type Error = Box<dyn std::error::Error + Send + Sync>;

/// Result of module methods.
pub type Result<T> = std::result::Result<T, Error>;

/// A tracking module. The host calls `initialize` once, `update` repeatedly
/// while the module is in use, and `teardown` when it stops the module.
pub trait Module: Send {
    /// Stable, machine friendly identifier: the settings key and the
    /// installed file name (`<id>.<os>.<arch>.vfgmod`). It must never change
    /// and must match `^[A-Za-z0-9][A-Za-z0-9._-]*$`.
    fn id(&self) -> &str;

    /// User facing information shown on the Modules page.
    fn metadata(&self) -> Metadata;

    /// What the module can provide before it is initialised.
    fn supported(&self) -> Support;

    /// Opens the device. `avail` is what the host still needs; return the
    /// subset the module will provide.
    fn initialize(
        &mut self,
        logger: &Logger,
        avail: Support,
    ) -> impl Future<Output = Result<Support>> + Send;

    /// Writes the current state into `data`, which keeps its values between
    /// calls. Wait a few milliseconds at most for fresh data; an error only
    /// skips this frame.
    fn update(&mut self, data: &mut TrackingData) -> impl Future<Output = Result<()>> + Send;

    /// Releases the device.
    fn teardown(&mut self) -> impl Future<Output = Result<()>> + Send;
}
