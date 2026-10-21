//! A demo tracking module built with the Rust SDK. It needs no hardware: it
//! opens and closes the jaw, blinks and moves the eyes in a loop so the whole
//! pipeline can be tried out.
//!
//! Build it into a module program and install it from the Modules page:
//!
//! ```sh
//! cargo build --release --example demo
//! # then copy target/release/examples/demo(.exe) to DemoRust.vfgmod
//! ```

use std::time::{Duration, Instant};

use serde_json::json;
use vrcftgo_sdk::{
    Logger, Metadata, Module, Result, Support, TrackingData, UnifiedExpression, Vector2,
};

const BLINK_EVERY: f64 = 4.0;
const BLINK_DURATION: f64 = 0.15;

struct Demo {
    logger: Option<Logger>,
    granted: Support,
    start: Instant,
}

impl Module for Demo {
    // Must never change: it is the settings key and the installed file name.
    fn id(&self) -> &str {
        "DemoRust"
    }

    fn metadata(&self) -> Metadata {
        Metadata {
            name: "Demo (Rust)".into(),
            description: "Animates the face without any hardware, as an example of the Rust SDK".into(),
            author: "SayukiDev".into(),
            version: "1.0.0".into(),
            usage_instructions: "Activate the module and watch the avatar's jaw, smile, eyes and blinks move on their own".into(),
            platforms: vec!["windows".into(), "linux".into(), "darwin".into()],
            ..Default::default()
        }
    }

    fn supported(&self) -> Support {
        Support::BOTH
    }

    // A real module would open the device here.
    async fn initialize(&mut self, logger: &Logger, avail: Support) -> Result<Support> {
        logger.info(
            "Demo module started",
            json!({"eye": avail.eye, "expression": avail.expression}),
        );
        self.logger = Some(logger.clone());
        self.granted = avail;
        self.start = Instant::now();
        Ok(avail)
    }

    // A real module would wait a few milliseconds for fresh device data.
    async fn update(&mut self, data: &mut TrackingData) -> Result<()> {
        tokio::time::sleep(Duration::from_millis(5)).await;
        let t = self.start.elapsed().as_secs_f64();

        if self.granted.expression {
            data.set_shape(
                UnifiedExpression::JawOpen,
                (0.4 + 0.4 * (t * 2.0).sin()) as f32,
            );
            let smile = (0.5 + 0.5 * (t * 0.5).sin()) as f32;
            data.set_shape(UnifiedExpression::MouthCornerPullLeft, smile);
            data.set_shape(UnifiedExpression::MouthCornerPullRight, smile);
        }

        if self.granted.eye {
            let openness = if t % BLINK_EVERY < BLINK_DURATION {
                0.0
            } else {
                1.0
            };
            let gaze = Vector2 {
                x: (0.3 * t.cos()) as f32,
                y: (0.2 * t.sin()) as f32,
            };
            for eye in [&mut data.eye.left, &mut data.eye.right] {
                eye.openness = openness;
                eye.gaze = gaze;
                eye.pupil_diameter_mm = 4.0;
            }
            data.eye.min_dilation = 2.0;
            data.eye.max_dilation = 7.0;
        }
        Ok(())
    }

    // A real module would release the device here.
    async fn teardown(&mut self) -> Result<()> {
        if let Some(logger) = &self.logger {
            logger.info("Demo module stopped", serde_json::Value::Null);
        }
        Ok(())
    }
}

fn main() {
    vrcftgo_sdk::serve(Demo {
        logger: None,
        granted: Support::default(),
        start: Instant::now(),
    });
}
