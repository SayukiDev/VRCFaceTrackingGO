package main

import "unsafe"

// The types below mirror the C layout of the OpenXR structures this module
// passes to the runtime (openxr.h 1.1, 64-bit). Go lays out these fields the
// same way a C compiler does on amd64 and arm64; xrtypes_test.go checks the
// sizes and offsets against values printed by a C program.
//
// Handles and atoms are 64-bit on every supported platform.

// Structure types (XrStructureType), followed by the enum values, flags and
// counts this module uses.
const (
	xrTypeExtensionProperties                    = 2
	xrTypeInstanceCreateInfo                     = 3
	xrTypeSystemGetInfo                          = 4
	xrTypeSystemProperties                       = 5
	xrTypeSessionCreateInfo                      = 8
	xrTypeSessionBeginInfo                       = 10
	xrTypeEventDataBuffer                        = 16
	xrTypeEventDataInstanceLossPending           = 17
	xrTypeEventDataSessionStateChanged           = 18
	xrTypeActionSetCreateInfo                    = 28
	xrTypeActionCreateInfo                       = 29
	xrTypeInstanceProperties                     = 32
	xrTypeReferenceSpaceCreateInfo               = 37
	xrTypeActionSpaceCreateInfo                  = 38
	xrTypeSpaceLocation                          = 42
	xrTypeInteractionProfileSuggestedBind        = 51
	xrTypeSessionActionSetsAttachInfo            = 60
	xrTypeActionsSyncInfo                        = 61
	xrTypeSystemEyeGazeInteractionProps          = 1000030000
	xrTypeSystemFacialTrackingPropsHTC           = 1000104000
	xrTypeFacialTrackerCreateInfoHTC             = 1000104001
	xrTypeFacialExpressionsHTC                   = 1000104002
	xrTypeSystemFaceTrackingProps2FB             = 1000287013
	xrTypeFaceTrackerCreateInfo2FB               = 1000287014
	xrTypeFaceExpressionInfo2FB                  = 1000287015
	xrTypeFaceExpressionWeights2FB               = 1000287016
	xrTypeFaceTrackerCreateInfoAndroid           = 1000458000
	xrTypeFaceStateGetInfoAndroid                = 1000458001
	xrTypeFaceStateAndroid                       = 1000458002
	xrTypeSystemFaceTrackingPropsAndroid         = 1000458003
	xrFormFactorHeadMountedDisplay               = 1
	xrViewConfigurationTypePrimaryStereo         = 2
	xrReferenceSpaceTypeView                     = 1
	xrActionTypePoseInput                        = 4
	xrFacialTrackingTypeEyeDefaultHTC            = 1
	xrFacialTrackingTypeLipDefaultHTC            = 2
	xrFaceTrackingDataSource2VisualFB            = 0
	xrFaceTrackingStateTrackingAndroid           = 2
	xrSpaceLocationOrientationValid              = 1
	xrFaceConfidenceCount2FB                     = 2
	xrFaceRegionConfidenceCountAndroid           = 3
	xrFaceParameterCountAndroid                  = 68
	xrAPIVersion1_0                       uint64 = 1 << 48
)

// Results (XrResult). Negative values are errors.
const (
	xrSuccess                    = 0
	xrEventUnavailable           = 4
	xrErrorRuntimeFailure        = -2
	xrErrorFeatureUnsupported    = -8
	xrErrorInstanceLost          = -13
	xrErrorSessionLost           = -17
	xrErrorFormFactorUnavailable = -35
	xrErrorRuntimeUnavailable    = -51
)

// Session states (XrSessionState).
const (
	xrSessionStateReady       = 2
	xrSessionStateStopping    = 6
	xrSessionStateLossPending = 7
	xrSessionStateExiting     = 8
)

// Extension names.
const (
	extHeadless    = "XR_MND_headless"
	extTimespec    = "XR_KHR_convert_timespec_time"
	extEyeGaze     = "XR_EXT_eye_gaze_interaction"
	extFaceFB2     = "XR_FB_face_tracking2"
	extFaceHTC     = "XR_HTC_facial_tracking"
	extFaceAndroid = "XR_ANDROID_face_tracking"
)

type xrPosef struct {
	Orientation [4]float32 // x, y, z, w
	Position    [3]float32
}

type xrExtensionProperties struct {
	Type             uint32
	Next             unsafe.Pointer
	ExtensionName    [128]byte
	ExtensionVersion uint32
}

type xrApplicationInfo struct {
	ApplicationName    [128]byte
	ApplicationVersion uint32
	EngineName         [128]byte
	EngineVersion      uint32
	APIVersion         uint64
}

type xrInstanceCreateInfo struct {
	Type                  uint32
	Next                  unsafe.Pointer
	CreateFlags           uint64
	ApplicationInfo       xrApplicationInfo
	EnabledAPILayerCount  uint32
	EnabledAPILayerNames  **byte
	EnabledExtensionCount uint32
	EnabledExtensionNames **byte
}

type xrInstanceProperties struct {
	Type           uint32
	Next           unsafe.Pointer
	RuntimeVersion uint64
	RuntimeName    [128]byte
}

type xrSystemGetInfo struct {
	Type       uint32
	Next       unsafe.Pointer
	FormFactor uint32
}

type xrSystemProperties struct {
	Type       uint32
	Next       unsafe.Pointer
	SystemID   uint64
	VendorID   uint32
	SystemName [256]byte
	Graphics   [3]uint32
	Tracking   [2]uint32
}

type xrSystemEyeGazeInteractionProperties struct {
	Type                       uint32
	Next                       unsafe.Pointer
	SupportsEyeGazeInteraction uint32
}

type xrSystemFaceTrackingProperties2FB struct {
	Type                       uint32
	Next                       unsafe.Pointer
	SupportsVisualFaceTracking uint32
	SupportsAudioFaceTracking  uint32
}

type xrSystemFacialTrackingPropertiesHTC struct {
	Type                     uint32
	Next                     unsafe.Pointer
	SupportEyeFacialTracking uint32
	SupportLipFacialTracking uint32
}

type xrSystemFaceTrackingPropertiesAndroid struct {
	Type                 uint32
	Next                 unsafe.Pointer
	SupportsFaceTracking uint32
}

type xrSessionCreateInfo struct {
	Type        uint32
	Next        unsafe.Pointer
	CreateFlags uint64
	SystemID    uint64
}

type xrSessionBeginInfo struct {
	Type                         uint32
	Next                         unsafe.Pointer
	PrimaryViewConfigurationType uint32
}

type xrEventDataBuffer struct {
	Type    uint32
	Next    unsafe.Pointer
	Varying [4000]byte
}

type xrEventDataSessionStateChanged struct {
	Type    uint32
	Next    unsafe.Pointer
	Session uint64
	State   uint32
	Time    int64
}

type xrActionSetCreateInfo struct {
	Type                   uint32
	Next                   unsafe.Pointer
	ActionSetName          [64]byte
	LocalizedActionSetName [128]byte
	Priority               uint32
}

type xrActionCreateInfo struct {
	Type                uint32
	Next                unsafe.Pointer
	ActionName          [64]byte
	ActionType          uint32
	CountSubactionPaths uint32
	SubactionPaths      *uint64
	LocalizedActionName [128]byte
}

type xrActionSuggestedBinding struct {
	Action  uint64
	Binding uint64
}

type xrInteractionProfileSuggestedBinding struct {
	Type                   uint32
	Next                   unsafe.Pointer
	InteractionProfile     uint64
	CountSuggestedBindings uint32
	SuggestedBindings      *xrActionSuggestedBinding
}

type xrSessionActionSetsAttachInfo struct {
	Type            uint32
	Next            unsafe.Pointer
	CountActionSets uint32
	ActionSets      *uint64
}

type xrActionSpaceCreateInfo struct {
	Type              uint32
	Next              unsafe.Pointer
	Action            uint64
	SubactionPath     uint64
	PoseInActionSpace xrPosef
}

type xrReferenceSpaceCreateInfo struct {
	Type                 uint32
	Next                 unsafe.Pointer
	ReferenceSpaceType   uint32
	PoseInReferenceSpace xrPosef
}

type xrActiveActionSet struct {
	ActionSet     uint64
	SubactionPath uint64
}

type xrActionsSyncInfo struct {
	Type                  uint32
	Next                  unsafe.Pointer
	CountActiveActionSets uint32
	ActiveActionSets      *xrActiveActionSet
}

type xrSpaceLocation struct {
	Type          uint32
	Next          unsafe.Pointer
	LocationFlags uint64
	Pose          xrPosef
}

type xrFaceTrackerCreateInfo2FB struct {
	Type                     uint32
	Next                     unsafe.Pointer
	FaceExpressionSet        uint32
	RequestedDataSourceCount uint32
	RequestedDataSources     *uint32
}

type xrFaceExpressionInfo2FB struct {
	Type uint32
	Next unsafe.Pointer
	Time int64
}

type xrFaceExpressionWeights2FB struct {
	Type                           uint32
	Next                           unsafe.Pointer
	WeightCount                    uint32
	Weights                        *float32
	ConfidenceCount                uint32
	Confidences                    *float32
	IsValid                        uint32
	IsEyeFollowingBlendshapesValid uint32
	DataSource                     uint32
	Time                           int64
}

type xrFacialTrackerCreateInfoHTC struct {
	Type               uint32
	Next               unsafe.Pointer
	FacialTrackingType uint32
}

type xrFacialExpressionsHTC struct {
	Type                 uint32
	Next                 unsafe.Pointer
	IsActive             uint32
	SampleTime           int64
	ExpressionCount      uint32
	ExpressionWeightings *float32
}

type xrFaceTrackerCreateInfoAndroid struct {
	Type uint32
	Next unsafe.Pointer
}

type xrFaceStateGetInfoAndroid struct {
	Type uint32
	Next unsafe.Pointer
	Time int64
}

type xrFaceStateAndroid struct {
	Type                           uint32
	Next                           unsafe.Pointer
	ParametersCapacityInput        uint32
	ParametersCountOutput          uint32
	Parameters                     *float32
	FaceTrackingState              uint32
	SampleTime                     int64
	IsValid                        uint32
	RegionConfidencesCapacityInput uint32
	RegionConfidencesCountOutput   uint32
	RegionConfidences              *float32
}

// xrTimespec is struct timespec on 64-bit Linux.
type xrTimespec struct {
	Sec  int64
	Nsec int64
}
