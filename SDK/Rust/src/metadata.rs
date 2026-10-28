use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use serde::{Serialize, Serializer};

/// Which kinds of tracking a module provides.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize)]
pub struct Support {
    pub eye: bool,
    pub expression: bool,
}

impl Support {
    /// Eye and expression tracking.
    pub const BOTH: Support = Support {
        eye: true,
        expression: true,
    };

    pub(crate) fn to_byte(self) -> u8 {
        u8::from(self.eye) | u8::from(self.expression) << 1
    }

    pub(crate) fn from_byte(b: u8) -> Self {
        Support {
            eye: b & 1 != 0,
            expression: b & 2 != 0,
        }
    }

    /// The kinds present in both.
    pub fn intersect(self, other: Support) -> Support {
        Support {
            eye: self.eye && other.eye,
            expression: self.expression && other.expression,
        }
    }
}

/// User facing information about a module, shown on the Modules page.
#[derive(Clone, Debug, Default, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Metadata {
    pub name: String,
    /// What the module provides, in one sentence.
    pub description: String,
    pub author: String,
    pub version: String,
    /// Home page of the module or of the software it needs.
    pub page_url: String,
    /// What the user must set up before activating the module.
    pub usage_instructions: String,
    /// PNG encoded images, e.g. `include_bytes!("icon.png").to_vec()`.
    #[serde(serialize_with = "base64_list")]
    pub icons: Vec<Vec<u8>>,
    /// Operating systems the module is for (`windows`, `linux`, `darwin`).
    /// Empty means all.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub platforms: Vec<String>,
}

fn base64_list<S: Serializer>(icons: &[Vec<u8>], s: S) -> Result<S::Ok, S::Error> {
    s.collect_seq(icons.iter().map(|i| STANDARD.encode(i)))
}

/// What `--info` prints.
#[derive(Clone, Debug, PartialEq, Serialize)]
pub struct Description {
    /// `Some("driver")` for a driver program, `None` for a module.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub kind: Option<String>,
    pub id: String,
    pub metadata: Metadata,
    pub supported: Support,
    /// The modules a driver program contains.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub modules: Vec<Description>,
}
