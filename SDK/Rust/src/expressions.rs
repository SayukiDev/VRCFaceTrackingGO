//! Unified Expressions shape indices, in the order of the wire format.
//!
//! Generated from `pkg/tracking/expressions.go`; the order mirrors the
//! Unified Expressions specification of VRCFaceTracking and must not change.

/// A Unified Expressions shape. Use it to index [`crate::TrackingData::shapes`]
/// or with [`crate::TrackingData::set_shape`].
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
#[repr(usize)]
pub enum UnifiedExpression {
    // Eye
    EyeSquintRight,
    EyeSquintLeft,
    EyeWideRight,
    EyeWideLeft,

    // Eyebrow
    BrowPinchRight,
    BrowPinchLeft,
    BrowLowererRight,
    BrowLowererLeft,
    BrowInnerUpRight,
    BrowInnerUpLeft,
    BrowOuterUpRight,
    BrowOuterUpLeft,

    // Nose
    NasalDilationRight,
    NasalDilationLeft,
    NasalConstrictRight,
    NasalConstrictLeft,

    // Cheek
    CheekSquintRight,
    CheekSquintLeft,
    CheekPuffRight,
    CheekPuffLeft,
    CheekSuckRight,
    CheekSuckLeft,

    // Jaw
    JawOpen,
    JawRight,
    JawLeft,
    JawForward,
    JawBackward,
    JawClench,
    JawMandibleRaise,
    MouthClosed,

    // Lip push/pull
    LipSuckUpperRight,
    LipSuckUpperLeft,
    LipSuckLowerRight,
    LipSuckLowerLeft,
    LipSuckCornerRight,
    LipSuckCornerLeft,
    LipFunnelUpperRight,
    LipFunnelUpperLeft,
    LipFunnelLowerRight,
    LipFunnelLowerLeft,
    LipPuckerUpperRight,
    LipPuckerUpperLeft,
    LipPuckerLowerRight,
    LipPuckerLowerLeft,

    // Upper lip raiser
    MouthUpperUpRight,
    MouthUpperUpLeft,
    MouthUpperDeepenRight,
    MouthUpperDeepenLeft,
    NoseSneerRight,
    NoseSneerLeft,

    // Lower lip depressor
    MouthLowerDownRight,
    MouthLowerDownLeft,

    // Mouth direction
    MouthUpperRight,
    MouthUpperLeft,
    MouthLowerRight,
    MouthLowerLeft,

    // Smile
    MouthCornerPullRight,
    MouthCornerPullLeft,
    MouthCornerSlantRight,
    MouthCornerSlantLeft,

    // Sad
    MouthFrownRight,
    MouthFrownLeft,
    MouthStretchRight,
    MouthStretchLeft,

    MouthDimpleRight,
    MouthDimpleLeft,

    MouthRaiserUpper,
    MouthRaiserLower,
    MouthPressRight,
    MouthPressLeft,
    MouthTightenerRight,
    MouthTightenerLeft,

    // Tongue
    TongueOut,
    TongueUp,
    TongueDown,
    TongueRight,
    TongueLeft,
    TongueRoll,
    TongueBendDown,
    TongueCurlUp,
    TongueSquish,
    TongueFlat,
    TongueTwistRight,
    TongueTwistLeft,

    // Throat / neck
    SoftPalateClose,
    ThroatSwallow,
    NeckFlexRight,
    NeckFlexLeft,
}

/// Number of Unified Expressions shapes.
pub const EXPRESSION_COUNT: usize = 88;
