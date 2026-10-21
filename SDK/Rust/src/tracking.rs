use crate::expressions::{EXPRESSION_COUNT, UnifiedExpression};

/// A 2D vector. Gaze is a direction in tangent space: `x` = tan(yaw), right
/// positive; `y` = tan(pitch), up positive.
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct Vector2 {
    pub x: f32,
    pub y: f32,
}

/// One eye.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SingleEyeData {
    pub gaze: Vector2,
    /// Pupil diameter in millimetres.
    pub pupil_diameter_mm: f32,
    /// 0 (closed) to 1 (open).
    pub openness: f32,
}

impl Default for SingleEyeData {
    fn default() -> Self {
        Self {
            gaze: Vector2::default(),
            pupil_diameter_mm: 0.0,
            openness: 1.0,
        }
    }
}

/// Both eyes plus the pupil diameter range seen so far, which the host uses
/// to normalise the pupil size.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct EyeData {
    pub left: SingleEyeData,
    pub right: SingleEyeData,
    pub max_dilation: f32,
    pub min_dilation: f32,
}

impl Default for EyeData {
    fn default() -> Self {
        Self {
            left: SingleEyeData::default(),
            right: SingleEyeData::default(),
            max_dilation: 0.0,
            min_dilation: 999.0,
        }
    }
}

/// Head rotation and position.
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct HeadData {
    pub yaw: f32,
    pub pitch: f32,
    pub roll: f32,
    pub pos_x: f32,
    pub pos_y: f32,
    pub pos_z: f32,
}

/// Everything a module reports in one Update. Fields the module does not
/// track keep their initial values.
#[derive(Clone, Debug, PartialEq)]
pub struct TrackingData {
    pub eye: EyeData,
    /// Shape weights from 0 to 1, indexed by [`UnifiedExpression`].
    pub shapes: [f32; EXPRESSION_COUNT],
    pub head: HeadData,
}

impl Default for TrackingData {
    fn default() -> Self {
        Self {
            eye: EyeData::default(),
            shapes: [0.0; EXPRESSION_COUNT],
            head: HeadData::default(),
        }
    }
}

impl TrackingData {
    /// Returns the weight of one shape.
    pub fn shape(&self, e: UnifiedExpression) -> f32 {
        self.shapes[e as usize]
    }

    /// Sets the weight of one shape.
    pub fn set_shape(&mut self, e: UnifiedExpression, weight: f32) {
        self.shapes[e as usize] = weight;
    }
}
