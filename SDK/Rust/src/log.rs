use std::io::Write;
use std::time::{SystemTime, UNIX_EPOCH};

use serde_json::{Map, Value};

/// Writes JSON log lines to stderr in the format the host re-logs
/// (`level`, `ts`, `logger`, `msg` plus extra fields).
#[derive(Clone, Debug)]
pub struct Logger {
    name: String,
}

impl Logger {
    /// A logger whose lines are tagged with `name`.
    pub fn new(name: impl Into<String>) -> Self {
        Self { name: name.into() }
    }

    /// A logger tagged `<name>.<sub>`.
    pub fn named(&self, sub: &str) -> Self {
        Self::new(format!("{}.{}", self.name, sub))
    }

    pub fn debug(&self, msg: &str, fields: Value) {
        self.log("debug", msg, fields)
    }

    pub fn info(&self, msg: &str, fields: Value) {
        self.log("info", msg, fields)
    }

    pub fn warn(&self, msg: &str, fields: Value) {
        self.log("warn", msg, fields)
    }

    pub fn error(&self, msg: &str, fields: Value) {
        self.log("error", msg, fields)
    }

    /// Writes one line. `fields` is a JSON object (e.g. `json!({"port": 1})`)
    /// or `Value::Null` for none.
    pub fn log(&self, level: &str, msg: &str, fields: Value) {
        let ts = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_or(0.0, |d| d.as_secs_f64());
        let mut line = Map::new();
        line.insert("level".into(), level.into());
        line.insert("ts".into(), ts.into());
        line.insert("logger".into(), self.name.clone().into());
        line.insert("msg".into(), msg.into());
        if let Value::Object(extra) = fields {
            line.extend(extra);
        }
        let mut out = std::io::stderr().lock();
        let _ = serde_json::to_writer(&mut out, &line);
        let _ = out.write_all(b"\n");
    }
}
