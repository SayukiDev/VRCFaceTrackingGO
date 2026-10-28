use std::io;

use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};

use crate::log::Logger;
use crate::metadata::{Description, Support};
use crate::protocol::*;
use crate::tracking::TrackingData;
use crate::{Error, Module};

/// Runs `module` as a module program and exits the process.
///
/// - `--info` prints the module description as JSON.
/// - `--vfgmod=true` (set by the host) serves the host on stdin/stdout
///   until stdin is closed.
/// - Anything else refuses to run, so a module opened by accident does not
///   touch any device.
///
/// It creates its own tokio runtime; call it from a plain `fn main`.
pub fn serve<M: Module>(module: M) -> ! {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match args.as_slice() {
        [a] if a == "--info" => print_description(&Description {
            kind: None,
            id: module.id().to_owned(),
            metadata: module.metadata(),
            supported: module.supported(),
            modules: Vec::new(),
        }),
        [a] if a == "--vfgmod=true" => {}
        _ => {
            eprintln!(
                "{} is a VRCFaceTrackingGO module. Install it from the Modules page instead of running it.",
                module.metadata().name
            );
            std::process::exit(2)
        }
    }

    let name = module.id().to_owned();
    run_host(module, &name)
}

/// Prints `d` for `--info` and exits.
pub(crate) fn print_description(d: &Description) -> ! {
    match serde_json::to_string(d) {
        Ok(json) => {
            println!("{json}");
            std::process::exit(0)
        }
        Err(_) => std::process::exit(1),
    }
}

/// Serves the host on stdin/stdout with `module`, logging as `name`, and
/// exits.
pub(crate) fn run_host<M: Module>(module: M, name: &str) -> ! {
    let logger = Logger::new(name);
    let runtime = match tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
    {
        Ok(rt) => rt,
        Err(e) => {
            logger.error(
                "Cannot start the async runtime",
                serde_json::json!({"error": e.to_string()}),
            );
            std::process::exit(1)
        }
    };
    let code = runtime.block_on(async {
        #[cfg(unix)]
        tokio::spawn(watch_parent(logger.clone()));
        match serve_io(module, &logger, tokio::io::stdin(), tokio::io::stdout()).await {
            Ok(()) => 0,
            Err(e) => {
                logger.error(
                    "Protocol error",
                    serde_json::json!({"error": e.to_string()}),
                );
                1
            }
        }
    });
    std::process::exit(code)
}

/// Answers the host's requests read from `r` until it ends, then tears the
/// module down. [`serve`] calls it with stdin and stdout; it is public for
/// tests and custom transports.
pub async fn serve_io<M, R, W>(mut module: M, logger: &Logger, mut r: R, mut w: W) -> io::Result<()>
where
    M: Module,
    R: AsyncRead + Unpin,
    W: AsyncWrite + Unpin,
{
    let mut data = TrackingData::default();
    let mut initialized = false;
    let mut buf = Vec::with_capacity(1 + DATA_SIZE);

    let result = loop {
        let op = match r.read_u8().await {
            Ok(op) => op,
            Err(e) if e.kind() == io::ErrorKind::UnexpectedEof => break Ok(()),
            Err(e) => break Err(e),
        };
        buf.clear();
        match op {
            OP_INITIALIZE => {
                let avail = Support::from_byte(r.read_u8().await?);
                match module.initialize(logger, avail).await {
                    Ok(granted) => {
                        initialized = true;
                        append_ok(&mut buf, &[granted.intersect(avail).to_byte()]);
                    }
                    Err(e) => append_err(&mut buf, &e.to_string()),
                }
            }
            OP_UPDATE => match module.update(&mut data).await {
                Ok(()) => {
                    buf.push(STATUS_OK);
                    append_data(&mut buf, &data);
                }
                Err(e) => append_err(&mut buf, &e.to_string()),
            },
            OP_TEARDOWN => match teardown(&mut module, &mut initialized).await {
                Ok(()) => append_ok(&mut buf, &[]),
                Err(e) => append_err(&mut buf, &e.to_string()),
            },
            other => {
                break Err(io::Error::new(
                    io::ErrorKind::InvalidData,
                    format!("unknown request {other:#04x}"),
                ));
            }
        }
        w.write_all(&buf).await?;
        w.flush().await?;
    };

    // The host is gone (or asked us to stop); release the device.
    if let Err(e) = teardown(&mut module, &mut initialized).await {
        logger.warn(
            "Module teardown failed",
            serde_json::json!({"error": e.to_string()}),
        );
    }
    result
}

async fn teardown<M: Module>(module: &mut M, initialized: &mut bool) -> Result<(), Error> {
    if !std::mem::take(initialized) {
        return Ok(());
    }
    module.teardown().await
}

/// Exits once the host is gone. A dead parent's children are re-parented,
/// so the parent PID changes. This also ends a module that is stuck and
/// never sees stdin close (macOS has no parent death signal).
#[cfg(unix)]
async fn watch_parent(logger: Logger) {
    let ppid = std::os::unix::process::parent_id();
    let mut tick = tokio::time::interval(std::time::Duration::from_millis(500));
    loop {
        tick.tick().await;
        if std::os::unix::process::parent_id() != ppid {
            logger.warn("Host is gone, exiting", serde_json::Value::Null);
            std::process::exit(1);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Metadata, Result};
    use tokio::io::{AsyncReadExt, AsyncWriteExt, duplex};

    struct Fake {
        torn_down: std::sync::Arc<std::sync::atomic::AtomicBool>,
    }

    impl Module for Fake {
        fn id(&self) -> &str {
            "Fake"
        }
        fn metadata(&self) -> Metadata {
            Metadata::default()
        }
        fn supported(&self) -> Support {
            Support::BOTH
        }
        async fn initialize(&mut self, _: &Logger, _avail: Support) -> Result<Support> {
            // Asks for more than offered; serve must clamp it.
            Ok(Support::BOTH)
        }
        async fn update(&mut self, data: &mut TrackingData) -> Result<()> {
            data.shapes[0] += 0.5;
            Ok(())
        }
        async fn teardown(&mut self) -> Result<()> {
            self.torn_down
                .store(true, std::sync::atomic::Ordering::SeqCst);
            Err("device busy".into())
        }
    }

    #[tokio::test]
    async fn round_trip() {
        let torn_down = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
        let (mut host, module_side) = duplex(4096);
        let (r, w) = tokio::io::split(module_side);
        let logger = Logger::new("test");
        let module = Fake {
            torn_down: torn_down.clone(),
        };
        let task = tokio::spawn(async move { serve_io(module, &logger, r, w).await });

        host.write_all(&[OP_INITIALIZE, 0b10]).await.unwrap();
        let mut init = [0u8; 2];
        host.read_exact(&mut init).await.unwrap();
        assert_eq!(
            init,
            [STATUS_OK, 0b10],
            "granted support is clamped to the offer"
        );

        for want in [0.5f32, 1.0] {
            host.write_all(&[OP_UPDATE]).await.unwrap();
            let mut resp = vec![0u8; 1 + DATA_SIZE];
            host.read_exact(&mut resp).await.unwrap();
            assert_eq!(resp[0], STATUS_OK);
            let shape0 = f32::from_le_bytes(resp[1 + 40..1 + 44].try_into().unwrap());
            assert_eq!(shape0, want, "data keeps its values between updates");
        }

        host.write_all(&[OP_TEARDOWN]).await.unwrap();
        let mut status = [0u8; 5];
        host.read_exact(&mut status).await.unwrap();
        assert_eq!(status[0], STATUS_ERR);
        let len = u32::from_le_bytes(status[1..5].try_into().unwrap()) as usize;
        let mut msg = vec![0u8; len];
        host.read_exact(&mut msg).await.unwrap();
        assert_eq!(msg, b"device busy");
        assert!(torn_down.load(std::sync::atomic::Ordering::SeqCst));

        drop(host);
        task.await.unwrap().unwrap();
    }

    #[tokio::test]
    async fn unknown_request_is_an_error() {
        let (mut host, module_side) = duplex(64);
        let (r, w) = tokio::io::split(module_side);
        let logger = Logger::new("test");
        let module = Fake {
            torn_down: Default::default(),
        };
        let task = tokio::spawn(async move { serve_io(module, &logger, r, w).await });
        host.write_all(b"X").await.unwrap();
        assert!(task.await.unwrap().is_err());
    }
}
