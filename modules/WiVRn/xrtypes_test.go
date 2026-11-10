package main

import (
	"testing"
	"unsafe"
)

// The expected numbers were printed by a C program using sizeof and
// offsetof on openxr.h 1.1.60 for x86_64; arm64 lays these out the same.
func TestLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("layout is only defined for 64-bit platforms")
	}
	sizes := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"XrExtensionProperties", unsafe.Sizeof(xrExtensionProperties{}), 152},
		{"XrApplicationInfo", unsafe.Sizeof(xrApplicationInfo{}), 272},
		{"XrInstanceCreateInfo", unsafe.Sizeof(xrInstanceCreateInfo{}), 328},
		{"XrInstanceProperties", unsafe.Sizeof(xrInstanceProperties{}), 152},
		{"XrSystemGetInfo", unsafe.Sizeof(xrSystemGetInfo{}), 24},
		{"XrSystemProperties", unsafe.Sizeof(xrSystemProperties{}), 304},
		{"XrSystemEyeGazeInteractionPropertiesEXT", unsafe.Sizeof(xrSystemEyeGazeInteractionProperties{}), 24},
		{"XrSystemFaceTrackingProperties2FB", unsafe.Sizeof(xrSystemFaceTrackingProperties2FB{}), 24},
		{"XrSystemFacialTrackingPropertiesHTC", unsafe.Sizeof(xrSystemFacialTrackingPropertiesHTC{}), 24},
		{"XrSystemFaceTrackingPropertiesANDROID", unsafe.Sizeof(xrSystemFaceTrackingPropertiesAndroid{}), 24},
		{"XrSessionCreateInfo", unsafe.Sizeof(xrSessionCreateInfo{}), 32},
		{"XrSessionBeginInfo", unsafe.Sizeof(xrSessionBeginInfo{}), 24},
		{"XrEventDataBuffer", unsafe.Sizeof(xrEventDataBuffer{}), 4016},
		{"XrEventDataSessionStateChanged", unsafe.Sizeof(xrEventDataSessionStateChanged{}), 40},
		{"XrActionSetCreateInfo", unsafe.Sizeof(xrActionSetCreateInfo{}), 216},
		{"XrActionCreateInfo", unsafe.Sizeof(xrActionCreateInfo{}), 224},
		{"XrActionSuggestedBinding", unsafe.Sizeof(xrActionSuggestedBinding{}), 16},
		{"XrInteractionProfileSuggestedBinding", unsafe.Sizeof(xrInteractionProfileSuggestedBinding{}), 40},
		{"XrSessionActionSetsAttachInfo", unsafe.Sizeof(xrSessionActionSetsAttachInfo{}), 32},
		{"XrActionSpaceCreateInfo", unsafe.Sizeof(xrActionSpaceCreateInfo{}), 64},
		{"XrReferenceSpaceCreateInfo", unsafe.Sizeof(xrReferenceSpaceCreateInfo{}), 48},
		{"XrActiveActionSet", unsafe.Sizeof(xrActiveActionSet{}), 16},
		{"XrActionsSyncInfo", unsafe.Sizeof(xrActionsSyncInfo{}), 32},
		{"XrSpaceLocation", unsafe.Sizeof(xrSpaceLocation{}), 56},
		{"XrFaceTrackerCreateInfo2FB", unsafe.Sizeof(xrFaceTrackerCreateInfo2FB{}), 32},
		{"XrFaceExpressionInfo2FB", unsafe.Sizeof(xrFaceExpressionInfo2FB{}), 24},
		{"XrFaceExpressionWeights2FB", unsafe.Sizeof(xrFaceExpressionWeights2FB{}), 72},
		{"XrFacialTrackerCreateInfoHTC", unsafe.Sizeof(xrFacialTrackerCreateInfoHTC{}), 24},
		{"XrFacialExpressionsHTC", unsafe.Sizeof(xrFacialExpressionsHTC{}), 48},
		{"XrFaceTrackerCreateInfoANDROID", unsafe.Sizeof(xrFaceTrackerCreateInfoAndroid{}), 16},
		{"XrFaceStateGetInfoANDROID", unsafe.Sizeof(xrFaceStateGetInfoAndroid{}), 24},
		{"XrFaceStateANDROID", unsafe.Sizeof(xrFaceStateAndroid{}), 72},
		{"timespec", unsafe.Sizeof(xrTimespec{}), 16},

		{"XrInstanceCreateInfo.enabledExtensionNames", unsafe.Offsetof(xrInstanceCreateInfo{}.EnabledExtensionNames), 320},
		{"XrSystemProperties.trackingProperties", unsafe.Offsetof(xrSystemProperties{}.Tracking), 296},
		{"XrActionCreateInfo.localizedActionName", unsafe.Offsetof(xrActionCreateInfo{}.LocalizedActionName), 96},
		{"XrActionSpaceCreateInfo.poseInActionSpace", unsafe.Offsetof(xrActionSpaceCreateInfo{}.PoseInActionSpace), 32},
		{"XrReferenceSpaceCreateInfo.poseInReferenceSpace", unsafe.Offsetof(xrReferenceSpaceCreateInfo{}.PoseInReferenceSpace), 20},
		{"XrSpaceLocation.pose", unsafe.Offsetof(xrSpaceLocation{}.Pose), 24},
		{"XrEventDataSessionStateChanged.state", unsafe.Offsetof(xrEventDataSessionStateChanged{}.State), 24},
		{"XrFaceExpressionWeights2FB.isValid", unsafe.Offsetof(xrFaceExpressionWeights2FB{}.IsValid), 48},
		{"XrFaceExpressionWeights2FB.time", unsafe.Offsetof(xrFaceExpressionWeights2FB{}.Time), 64},
		{"XrFacialExpressionsHTC.expressionWeightings", unsafe.Offsetof(xrFacialExpressionsHTC{}.ExpressionWeightings), 40},
		{"XrFaceStateANDROID.isValid", unsafe.Offsetof(xrFaceStateAndroid{}.IsValid), 48},
		{"XrFaceStateANDROID.regionConfidences", unsafe.Offsetof(xrFaceStateAndroid{}.RegionConfidences), 64},
	}
	for _, s := range sizes {
		if s.got != s.want {
			t.Errorf("%s = %d, want %d", s.name, s.got, s.want)
		}
	}
}

func TestCounts(t *testing.T) {
	if fbFace1Count != 63 || fbFace2Count != 70 || fbTongueOut != 68 {
		t.Errorf("fb counts = %d/%d, tongue %d", fbFace1Count, fbFace2Count, fbTongueOut)
	}
	if htcEyesCount != 14 || htcLipCount != 37 {
		t.Errorf("htc eyes %d, lip %d", htcEyesCount, htcLipCount)
	}
	if androidTongueOut != 63 || androidTongueDown != 67 || androidCount != xrFaceParameterCountAndroid {
		t.Errorf("android tongue %d..%d, count %d", androidTongueOut, androidTongueDown, androidCount)
	}
}
