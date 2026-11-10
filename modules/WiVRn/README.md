# WiVRn Module

Face and eye tracking from headsets connected through [WiVRn](https://github.com/WiVRn/WiVRn).

WiVRn offers tracking data to OpenXR applications only, so this module opens its own headless OpenXR session next to the game and reads the data from it. It does not use cgo; the OpenXR loader is called through [purego](https://github.com/ebitengine/purego).

## Requirements

- Linux with the OpenXR loader installed (`libopenxr_loader.so.1`, package `openxr` on Arch, `libopenxr-loader1` on Debian/Ubuntu).
- Face and eye tracking enabled in the WiVRn headset app, with the headset's permissions granted. WiVRn decides what it offers when the headset connects, so reconnect after changing this.

The module can start before WiVRn or the headset. It retries every 5 seconds until the headset is streaming, and connects again after a disconnect.

## Supported data

| OpenXR extension | Headsets |
|---|---|
| `XR_ANDROID_face_tracking` | Samsung Galaxy XR |
| `XR_FB_face_tracking2` | Meta Quest Pro, PICO 4 Pro / Enterprise |
| `XR_HTC_facial_tracking` | HTC headsets with the eye or lip tracker |
| `XR_EXT_eye_gaze_interaction` | All of the above |

Tested with a Samsung Galaxy XR on WiVRn 26.9: the session connects and delivers face and gaze data. The resulting expressions have not been checked on an avatar yet. The Quest Pro, PICO and HTC paths have not been run on a headset; their structure layouts and mappings are covered by tests.

WiVRn reports one combined gaze, so both eyes look the same way. Eye openness comes from the face tracking data; with eye gaze alone the eyes stay open.

## Troubleshooting

- `XR_ERROR_RUNTIME_UNAVAILABLE`: WiVRn is not the active OpenXR runtime. It registers itself once the headset connects. To use a specific runtime, set `XR_RUNTIME_JSON` to its manifest.
- "the headset reports no face or eye tracking": enable them in the WiVRn headset app and reconnect.
- Loader messages are silenced because stderr carries the module's log. Set `XR_LOADER_DEBUG=error` to see them.

## Testing against a running WiVRn

```sh
WIVRN_LIVE=1 go test -run TestLive -v ./modules/WiVRn/
```

## Credits

The expression mappings are taken from this repository's [ALVR module](../ALVR), a port of [alvr-org/VRCFT-ALVR](https://github.com/alvr-org/VRCFT-ALVR) licensed under the Apache License 2.0; see [LICENSE](LICENSE). The approach of reading WiVRn through a headless OpenXR session follows [oscavmgr](https://github.com/galister/oscavmgr). The icon is the WiVRn logo.
