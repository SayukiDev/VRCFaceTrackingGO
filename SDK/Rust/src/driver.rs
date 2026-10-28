use std::future::Future;
use std::pin::Pin;

use crate::log::Logger;
use crate::metadata::{Description, Metadata, Support};
use crate::serve::{print_description, run_host};
use crate::tracking::TrackingData;
use crate::{Module, Result};

/// Static information of a driver program: one program that bundles
/// several modules. The host lists the driver separately and never loads
/// it; it loads the modules inside it instead.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Driver {
    pub id: String,
    pub metadata: Metadata,
}

/// Runs a driver program for `modules` and exits the process.
///
/// - `--info` prints the driver and all of its modules as JSON.
/// - `--vfgmod=true --source=<id>` (set by the host) serves module `<id>`.
/// - Anything else refuses to run.
///
/// Wrap each module with [`BoxModule::new`]. It creates its own tokio
/// runtime; call it from a plain `fn main`.
pub fn serve_driver(driver: Driver, modules: Vec<BoxModule>) -> ! {
    let mut host = false;
    let mut source = String::new();
    for arg in std::env::args().skip(1) {
        if arg == "--info" {
            print_description(&Description {
                kind: Some("driver".to_owned()),
                id: driver.id.clone(),
                metadata: driver.metadata.clone(),
                supported: Support::default(),
                modules: modules.iter().map(BoxModule::description).collect(),
            });
        } else if arg == "--vfgmod=true" {
            host = true;
        } else if let Some(id) = arg.strip_prefix("--source=") {
            source = id.to_owned();
        } else {
            host = false;
            break;
        }
    }
    if !host {
        eprintln!(
            "{} is a VRCFaceTrackingGO driver. Install it from the Modules page instead of running it.",
            driver.metadata.name
        );
        std::process::exit(2)
    }
    match pick(modules, &source) {
        Ok(module) => run_host(module, &driver.id),
        Err(e) => {
            eprintln!("{e}");
            std::process::exit(2)
        }
    }
}

/// Returns the module of `modules` with the given ID.
fn pick(mut modules: Vec<BoxModule>, source: &str) -> Result<BoxModule> {
    if source.is_empty() {
        return Err("no module selected: use --source".into());
    }
    match modules.iter().position(|m| m.id() == source) {
        Some(i) => Ok(modules.swap_remove(i)),
        None => Err(format!("this driver contains no module {source:?}").into()),
    }
}

type BoxFuture<'a, T> = Pin<Box<dyn Future<Output = T> + Send + 'a>>;

/// Object safe form of [`Module`].
trait DynModule: Send {
    fn dyn_id(&self) -> &str;
    fn dyn_metadata(&self) -> Metadata;
    fn dyn_supported(&self) -> Support;
    fn dyn_initialize<'a>(
        &'a mut self,
        logger: &'a Logger,
        avail: Support,
    ) -> BoxFuture<'a, Result<Support>>;
    fn dyn_update<'a>(&'a mut self, data: &'a mut TrackingData) -> BoxFuture<'a, Result<()>>;
    fn dyn_teardown(&mut self) -> BoxFuture<'_, Result<()>>;
}

impl<M: Module> DynModule for M {
    fn dyn_id(&self) -> &str {
        Module::id(self)
    }
    fn dyn_metadata(&self) -> Metadata {
        Module::metadata(self)
    }
    fn dyn_supported(&self) -> Support {
        Module::supported(self)
    }
    fn dyn_initialize<'a>(
        &'a mut self,
        logger: &'a Logger,
        avail: Support,
    ) -> BoxFuture<'a, Result<Support>> {
        Box::pin(Module::initialize(self, logger, avail))
    }
    fn dyn_update<'a>(&'a mut self, data: &'a mut TrackingData) -> BoxFuture<'a, Result<()>> {
        Box::pin(Module::update(self, data))
    }
    fn dyn_teardown(&mut self) -> BoxFuture<'_, Result<()>> {
        Box::pin(Module::teardown(self))
    }
}

/// A module of any type, so modules of different types can be passed to
/// [`serve_driver`] together.
pub struct BoxModule(Box<dyn DynModule>);

impl BoxModule {
    pub fn new<M: Module + 'static>(module: M) -> Self {
        BoxModule(Box::new(module))
    }

    fn description(&self) -> Description {
        Description {
            kind: None,
            id: self.0.dyn_id().to_owned(),
            metadata: self.0.dyn_metadata(),
            supported: self.0.dyn_supported(),
            modules: Vec::new(),
        }
    }
}

impl Module for BoxModule {
    fn id(&self) -> &str {
        self.0.dyn_id()
    }
    fn metadata(&self) -> Metadata {
        self.0.dyn_metadata()
    }
    fn supported(&self) -> Support {
        self.0.dyn_supported()
    }
    async fn initialize(&mut self, logger: &Logger, avail: Support) -> Result<Support> {
        self.0.dyn_initialize(logger, avail).await
    }
    async fn update(&mut self, data: &mut TrackingData) -> Result<()> {
        self.0.dyn_update(data).await
    }
    async fn teardown(&mut self) -> Result<()> {
        self.0.dyn_teardown().await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Fake {
        id: &'static str,
    }

    impl Module for Fake {
        fn id(&self) -> &str {
            self.id
        }
        fn metadata(&self) -> Metadata {
            Metadata::default()
        }
        fn supported(&self) -> Support {
            Support::BOTH
        }
        async fn initialize(&mut self, _: &Logger, avail: Support) -> Result<Support> {
            Ok(avail)
        }
        async fn update(&mut self, _: &mut TrackingData) -> Result<()> {
            Ok(())
        }
        async fn teardown(&mut self) -> Result<()> {
            Ok(())
        }
    }

    #[test]
    fn pick_selects_one_module() {
        let modules = || {
            vec![
                BoxModule::new(Fake { id: "eye" }),
                BoxModule::new(Fake { id: "face" }),
            ]
        };
        assert!(pick(modules(), "").is_err());
        assert!(pick(modules(), "missing").is_err());
        assert_eq!(pick(modules(), "face").unwrap().id(), "face");
    }
}
