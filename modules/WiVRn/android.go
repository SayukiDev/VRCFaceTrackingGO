package main

import "github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"

// XR_ANDROID_face_tracking lays out its first fbFace1Count parameters exactly
// like XR_FB_face_tracking2; only the tongue differs.
const (
	androidTongueOut = int(fbFace1Count) + iota
	androidTongueLeft
	androidTongueRight
	androidTongueUp
	androidTongueDown

	androidCount
)

func setFaceAndroid(p []float32, data *tracking.TrackingData) {
	setFaceFb(p, data)
	data.Shapes[tracking.TongueOut].Weight = p[androidTongueOut]
	data.Shapes[tracking.TongueLeft].Weight = p[androidTongueLeft]
	data.Shapes[tracking.TongueRight].Weight = p[androidTongueRight]
	data.Shapes[tracking.TongueUp].Weight = p[androidTongueUp]
	data.Shapes[tracking.TongueDown].Weight = p[androidTongueDown]
}
