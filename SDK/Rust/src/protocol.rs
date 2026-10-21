//! Wire format of the module protocol (little endian).

use crate::expressions::EXPRESSION_COUNT;
use crate::tracking::TrackingData;

pub(crate) const OP_INITIALIZE: u8 = b'I';
pub(crate) const OP_UPDATE: u8 = b'U';
pub(crate) const OP_TEARDOWN: u8 = b'T';

pub(crate) const STATUS_OK: u8 = 0;
pub(crate) const STATUS_ERR: u8 = 1;

/// Error messages longer than this are cut.
pub(crate) const MAX_ERROR_LEN: usize = 64 * 1024;

/// Per eye (left, right) gaze X, gaze Y, pupil diameter and openness, then
/// max and min dilation, every shape weight and the head pose.
pub(crate) const DATA_FLOATS: usize = 2 * 4 + 2 + EXPRESSION_COUNT + 6;
pub(crate) const DATA_SIZE: usize = DATA_FLOATS * 4;

/// Appends the tracking data in wire order.
pub(crate) fn append_data(buf: &mut Vec<u8>, d: &TrackingData) {
    let mut put = |v: f32| buf.extend_from_slice(&v.to_le_bytes());
    for e in [&d.eye.left, &d.eye.right] {
        put(e.gaze.x);
        put(e.gaze.y);
        put(e.pupil_diameter_mm);
        put(e.openness);
    }
    put(d.eye.max_dilation);
    put(d.eye.min_dilation);
    for &w in &d.shapes {
        put(w);
    }
    let h = &d.head;
    for v in [h.yaw, h.pitch, h.roll, h.pos_x, h.pos_y, h.pos_z] {
        put(v);
    }
}

/// Appends a success response with `payload`.
pub(crate) fn append_ok(buf: &mut Vec<u8>, payload: &[u8]) {
    buf.push(STATUS_OK);
    buf.extend_from_slice(payload);
}

/// Appends an error response.
pub(crate) fn append_err(buf: &mut Vec<u8>, msg: &str) {
    let mut msg = msg.as_bytes();
    if msg.len() > MAX_ERROR_LEN {
        msg = &msg[..MAX_ERROR_LEN];
    }
    buf.push(STATUS_ERR);
    buf.extend_from_slice(&(msg.len() as u32).to_le_bytes());
    buf.extend_from_slice(msg);
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::UnifiedExpression;

    #[test]
    fn data_layout() {
        assert_eq!(DATA_SIZE, 416);
        let mut d = TrackingData::default();
        d.eye.left.gaze.x = 0.5;
        d.set_shape(UnifiedExpression::JawOpen, 0.25);
        d.head.pos_z = 2.0;
        let mut buf = Vec::new();
        append_data(&mut buf, &d);
        assert_eq!(buf.len(), DATA_SIZE);
        let f = |i: usize| f32::from_le_bytes(buf[i * 4..i * 4 + 4].try_into().unwrap());
        assert_eq!(f(0), 0.5);
        assert_eq!(f(3), 1.0, "left openness starts at 1");
        assert_eq!(f(9), 999.0, "min dilation starts at 999");
        assert_eq!(f(10 + 22), 0.25, "JawOpen is shape 22");
        assert_eq!(f(DATA_FLOATS - 1), 2.0);
    }
}
