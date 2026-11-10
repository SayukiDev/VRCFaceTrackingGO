//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"
)

// loaderNames are tried in order. The loader then finds the active runtime,
// which WiVRn registers for as long as its server runs.
var loaderNames = []string{"libopenxr_loader.so.1", "libopenxr_loader.so"}

// xrError is a failed OpenXR call.
type xrError struct {
	call   string
	result int32
}

func (e *xrError) Error() string {
	name := map[int32]string{
		xrErrorRuntimeFailure:        "XR_ERROR_RUNTIME_FAILURE",
		xrErrorFeatureUnsupported:    "XR_ERROR_FEATURE_UNSUPPORTED",
		xrErrorInstanceLost:          "XR_ERROR_INSTANCE_LOST",
		xrErrorSessionLost:           "XR_ERROR_SESSION_LOST",
		xrErrorFormFactorUnavailable: "XR_ERROR_FORM_FACTOR_UNAVAILABLE (no headset connected)",
		xrErrorRuntimeUnavailable:    "XR_ERROR_RUNTIME_UNAVAILABLE (no active OpenXR runtime; start WiVRn and connect the headset)",
	}[e.result]
	if name == "" {
		name = fmt.Sprintf("XrResult %d", e.result)
	}
	return fmt.Sprintf("WiVRn: %s: %s", e.call, name)
}

func (e *xrError) Is(target error) bool {
	return target == errLost && (e.result == xrErrorInstanceLost || e.result == xrErrorSessionLost)
}

// check turns an XrResult into an error. Positive results are qualified
// successes.
func check(call string, result int32) error {
	if result >= 0 {
		return nil
	}
	return &xrError{call: call, result: result}
}

// openxr is xrAPI on top of the OpenXR loader, called through purego.
type openxr struct {
	// Core functions, exported by the loader.
	enumerateInstanceExtensionProperties func(layer *byte, capacity uint32, count *uint32, props *xrExtensionProperties) int32
	createInstance                       func(info *xrInstanceCreateInfo, instance *uint64) int32
	destroyInstance                      func(instance uint64) int32
	getInstanceProperties                func(instance uint64, props *xrInstanceProperties) int32
	getInstanceProcAddr                  func(instance uint64, name *byte, fn *uintptr) int32
	getSystem                            func(instance uint64, info *xrSystemGetInfo, system *uint64) int32
	getSystemProperties                  func(instance, system uint64, props *xrSystemProperties) int32
	createSession                        func(instance uint64, info *xrSessionCreateInfo, session *uint64) int32
	beginSession                         func(session uint64, info *xrSessionBeginInfo) int32
	endSession                           func(session uint64) int32
	pollEventFn                          func(instance uint64, buf *xrEventDataBuffer) int32
	stringToPath                         func(instance uint64, s *byte, path *uint64) int32
	createActionSet                      func(instance uint64, info *xrActionSetCreateInfo, set *uint64) int32
	createAction                         func(set uint64, info *xrActionCreateInfo, action *uint64) int32
	suggestInteractionProfileBindings    func(instance uint64, info *xrInteractionProfileSuggestedBinding) int32
	attachSessionActionSets              func(session uint64, info *xrSessionActionSetsAttachInfo) int32
	createActionSpace                    func(session uint64, info *xrActionSpaceCreateInfo, space *uint64) int32
	createReferenceSpace                 func(session uint64, info *xrReferenceSpaceCreateInfo, space *uint64) int32
	syncActions                          func(session uint64, info *xrActionsSyncInfo) int32
	locateSpace                          func(space, base uint64, time int64, loc *xrSpaceLocation) int32

	// Extension functions, looked up per instance.
	convertTimespecTimeToTime func(instance uint64, ts *xrTimespec, time *int64) int32
	createFaceTracker2FB      func(session uint64, info *xrFaceTrackerCreateInfo2FB, tracker *uint64) int32
	getFaceExpressionWeights2 func(tracker uint64, info *xrFaceExpressionInfo2FB, weights *xrFaceExpressionWeights2FB) int32
	createFacialTrackerHTC    func(session uint64, info *xrFacialTrackerCreateInfoHTC, tracker *uint64) int32
	getFacialExpressionsHTC   func(tracker uint64, expr *xrFacialExpressionsHTC) int32
	createFaceTrackerAndroid  func(session uint64, info *xrFaceTrackerCreateInfoAndroid, tracker *uint64) int32
	getFaceStateAndroid       func(tracker uint64, info *xrFaceStateGetInfoAndroid, state *xrFaceStateAndroid) int32

	instance uint64
	session  uint64

	gazeSet   uint64
	gazeSpace uint64
	viewSpace uint64

	fb2Tracker     uint64
	htcEyeTracker  uint64
	htcLipTracker  uint64
	androidTracker uint64

	// buf holds what the runtime writes into. Keeping it in this long-lived
	// value means the pointers handed to C stay put between calls.
	buf struct {
		event xrEventDataBuffer

		fb2Weights     [fbFace2Count]float32
		fb2Confidences [xrFaceConfidenceCount2FB]float32
		htcEye         [htcEyesCount]float32
		htcLip         [htcLipCount]float32
		android        [androidCount]float32
		androidRegions [xrFaceRegionConfidenceCountAndroid]float32
	}
}

// openXR loads the OpenXR loader. It does not talk to a runtime yet.
func openXR() (xrAPI, error) {
	// The loader prints every failure to stderr, which is this module's log
	// channel; while no headset is connected that would be several lines per
	// retry. The failures are reported through the returned errors instead.
	if _, set := os.LookupEnv("XR_LOADER_DEBUG"); !set {
		os.Setenv("XR_LOADER_DEBUG", "none")
	}

	var (
		lib uintptr
		err error
	)
	for _, name := range loaderNames {
		if lib, err = purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("WiVRn: load the OpenXR loader (install the openxr package): %w", err)
	}

	x := &openxr{}
	for name, fn := range map[string]any{
		"xrEnumerateInstanceExtensionProperties": &x.enumerateInstanceExtensionProperties,
		"xrCreateInstance":                       &x.createInstance,
		"xrDestroyInstance":                      &x.destroyInstance,
		"xrGetInstanceProperties":                &x.getInstanceProperties,
		"xrGetInstanceProcAddr":                  &x.getInstanceProcAddr,
		"xrGetSystem":                            &x.getSystem,
		"xrGetSystemProperties":                  &x.getSystemProperties,
		"xrCreateSession":                        &x.createSession,
		"xrBeginSession":                         &x.beginSession,
		"xrEndSession":                           &x.endSession,
		"xrPollEvent":                            &x.pollEventFn,
		"xrStringToPath":                         &x.stringToPath,
		"xrCreateActionSet":                      &x.createActionSet,
		"xrCreateAction":                         &x.createAction,
		"xrSuggestInteractionProfileBindings":    &x.suggestInteractionProfileBindings,
		"xrAttachSessionActionSets":              &x.attachSessionActionSets,
		"xrCreateActionSpace":                    &x.createActionSpace,
		"xrCreateReferenceSpace":                 &x.createReferenceSpace,
		"xrSyncActions":                          &x.syncActions,
		"xrLocateSpace":                          &x.locateSpace,
	} {
		sym, err := purego.Dlsym(lib, name)
		if err != nil {
			return nil, fmt.Errorf("WiVRn: OpenXR loader: %w", err)
		}
		purego.RegisterFunc(fn, sym)
	}
	return x, nil
}

// cstr returns s as a NUL terminated C string.
func cstr(s string) *byte {
	b := append([]byte(s), 0)
	return &b[0]
}

// gostr reads a NUL terminated string out of a fixed size C array.
func gostr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// extensions lists what the active runtime supports.
func (x *openxr) extensions() (map[string]bool, error) {
	var count uint32
	if err := check("xrEnumerateInstanceExtensionProperties", x.enumerateInstanceExtensionProperties(nil, 0, &count, nil)); err != nil {
		return nil, err
	}
	found := make(map[string]bool, count)
	if count == 0 {
		return found, nil
	}
	props := make([]xrExtensionProperties, count)
	for i := range props {
		props[i].Type = xrTypeExtensionProperties
	}
	if err := check("xrEnumerateInstanceExtensionProperties", x.enumerateInstanceExtensionProperties(nil, count, &count, &props[0])); err != nil {
		return nil, err
	}
	for i := range props[:count] {
		found[gostr(props[i].ExtensionName[:])] = true
	}
	return found, nil
}

// bind looks up an extension function of the current instance.
func (x *openxr) bind(fn any, name string) error {
	var addr uintptr
	if err := check("xrGetInstanceProcAddr("+name+")", x.getInstanceProcAddr(x.instance, cstr(name), &addr)); err != nil {
		return err
	}
	if addr == 0 {
		return fmt.Errorf("WiVRn: runtime has no %s", name)
	}
	purego.RegisterFunc(fn, addr)
	return nil
}

func (x *openxr) path(s string) (uint64, error) {
	var p uint64
	err := check("xrStringToPath("+s+")", x.stringToPath(x.instance, cstr(s), &p))
	return p, err
}

func (x *openxr) connect() (features, error) {
	var feat features

	available, err := x.extensions()
	if err != nil {
		return feat, err
	}
	// Without a headless session this module would need a graphics binding
	// and would compete with the game for the display.
	for _, required := range []string{extHeadless, extTimespec} {
		if !available[required] {
			return feat, fmt.Errorf("WiVRn: the active OpenXR runtime lacks %s; is WiVRn the active runtime?", required)
		}
	}
	enabled := []string{extHeadless, extTimespec}
	for _, optional := range []string{extEyeGaze, extFaceFB2, extFaceHTC, extFaceAndroid} {
		if available[optional] {
			enabled = append(enabled, optional)
		}
	}
	names := make([]*byte, len(enabled))
	for i, name := range enabled {
		names[i] = cstr(name)
	}

	create := xrInstanceCreateInfo{
		Type:                  xrTypeInstanceCreateInfo,
		EnabledExtensionCount: uint32(len(names)),
		EnabledExtensionNames: &names[0],
	}
	copy(create.ApplicationInfo.ApplicationName[:], "VRCFaceTrackingGO")
	create.ApplicationInfo.ApplicationVersion = 1
	create.ApplicationInfo.APIVersion = xrAPIVersion1_0
	if err := check("xrCreateInstance", x.createInstance(&create, &x.instance)); err != nil {
		x.instance = 0
		return feat, err
	}

	props := xrInstanceProperties{Type: xrTypeInstanceProperties}
	if err := check("xrGetInstanceProperties", x.getInstanceProperties(x.instance, &props)); err != nil {
		return feat, err
	}
	feat.runtime = gostr(props.RuntimeName[:])

	if err := x.bind(&x.convertTimespecTimeToTime, "xrConvertTimespecTimeToTimeKHR"); err != nil {
		return feat, err
	}

	// The system only exists while a headset is connected.
	var system uint64
	get := xrSystemGetInfo{Type: xrTypeSystemGetInfo, FormFactor: xrFormFactorHeadMountedDisplay}
	if err := check("xrGetSystem", x.getSystem(x.instance, &get, &system)); err != nil {
		return feat, err
	}

	// Ask only about the extensions that were enabled; a runtime may reject
	// structures it does not know.
	var (
		gazeProps    = xrSystemEyeGazeInteractionProperties{Type: xrTypeSystemEyeGazeInteractionProps}
		fb2Props     = xrSystemFaceTrackingProperties2FB{Type: xrTypeSystemFaceTrackingProps2FB}
		htcProps     = xrSystemFacialTrackingPropertiesHTC{Type: xrTypeSystemFacialTrackingPropsHTC}
		androidProps = xrSystemFaceTrackingPropertiesAndroid{Type: xrTypeSystemFaceTrackingPropsAndroid}
	)
	sys := xrSystemProperties{Type: xrTypeSystemProperties}
	if available[extEyeGaze] {
		gazeProps.Next, sys.Next = sys.Next, unsafe.Pointer(&gazeProps)
	}
	if available[extFaceFB2] {
		fb2Props.Next, sys.Next = sys.Next, unsafe.Pointer(&fb2Props)
	}
	if available[extFaceHTC] {
		htcProps.Next, sys.Next = sys.Next, unsafe.Pointer(&htcProps)
	}
	if available[extFaceAndroid] {
		androidProps.Next, sys.Next = sys.Next, unsafe.Pointer(&androidProps)
	}
	if err := check("xrGetSystemProperties", x.getSystemProperties(x.instance, system, &sys)); err != nil {
		return feat, err
	}
	feat.system = gostr(sys.SystemName[:])
	feat.eyeGaze = gazeProps.SupportsEyeGazeInteraction != 0
	feat.fb2 = fb2Props.SupportsVisualFaceTracking != 0
	feat.htcEye = htcProps.SupportEyeFacialTracking != 0
	feat.htcLip = htcProps.SupportLipFacialTracking != 0
	feat.android = androidProps.SupportsFaceTracking != 0
	if !feat.any() {
		return feat, nil
	}

	// No graphics binding: XR_MND_headless makes this a headless session.
	session := xrSessionCreateInfo{Type: xrTypeSessionCreateInfo, SystemID: system}
	if err := check("xrCreateSession", x.createSession(x.instance, &session, &x.session)); err != nil {
		return feat, err
	}

	if feat.eyeGaze {
		if err := x.setupGaze(); err != nil {
			return feat, err
		}
	}
	if feat.fb2 {
		if err := errors.Join(
			x.bind(&x.createFaceTracker2FB, "xrCreateFaceTracker2FB"),
			x.bind(&x.getFaceExpressionWeights2, "xrGetFaceExpressionWeights2FB"),
		); err != nil {
			return feat, err
		}
		source := uint32(xrFaceTrackingDataSource2VisualFB)
		info := xrFaceTrackerCreateInfo2FB{
			Type:                     xrTypeFaceTrackerCreateInfo2FB,
			RequestedDataSourceCount: 1,
			RequestedDataSources:     &source,
		}
		if err := check("xrCreateFaceTracker2FB", x.createFaceTracker2FB(x.session, &info, &x.fb2Tracker)); err != nil {
			return feat, err
		}
	}
	if feat.htcEye || feat.htcLip {
		if err := errors.Join(
			x.bind(&x.createFacialTrackerHTC, "xrCreateFacialTrackerHTC"),
			x.bind(&x.getFacialExpressionsHTC, "xrGetFacialExpressionsHTC"),
		); err != nil {
			return feat, err
		}
		for _, t := range []struct {
			on      bool
			kind    uint32
			tracker *uint64
		}{
			{feat.htcEye, xrFacialTrackingTypeEyeDefaultHTC, &x.htcEyeTracker},
			{feat.htcLip, xrFacialTrackingTypeLipDefaultHTC, &x.htcLipTracker},
		} {
			if !t.on {
				continue
			}
			info := xrFacialTrackerCreateInfoHTC{Type: xrTypeFacialTrackerCreateInfoHTC, FacialTrackingType: t.kind}
			if err := check("xrCreateFacialTrackerHTC", x.createFacialTrackerHTC(x.session, &info, t.tracker)); err != nil {
				return feat, err
			}
		}
	}
	if feat.android {
		if err := errors.Join(
			x.bind(&x.createFaceTrackerAndroid, "xrCreateFaceTrackerANDROID"),
			x.bind(&x.getFaceStateAndroid, "xrGetFaceStateANDROID"),
		); err != nil {
			return feat, err
		}
		info := xrFaceTrackerCreateInfoAndroid{Type: xrTypeFaceTrackerCreateInfoAndroid}
		if err := check("xrCreateFaceTrackerANDROID", x.createFaceTrackerAndroid(x.session, &info, &x.androidTracker)); err != nil {
			return feat, err
		}
	}
	return feat, nil
}

// setupGaze binds a pose action to the eye gaze and creates the spaces to
// locate it in.
func (x *openxr) setupGaze() error {
	setInfo := xrActionSetCreateInfo{Type: xrTypeActionSetCreateInfo}
	copy(setInfo.ActionSetName[:], "vrcfacetracking")
	copy(setInfo.LocalizedActionSetName[:], "VRCFaceTrackingGO")
	if err := check("xrCreateActionSet", x.createActionSet(x.instance, &setInfo, &x.gazeSet)); err != nil {
		return err
	}

	var action uint64
	actionInfo := xrActionCreateInfo{Type: xrTypeActionCreateInfo, ActionType: xrActionTypePoseInput}
	copy(actionInfo.ActionName[:], "eye_gaze")
	copy(actionInfo.LocalizedActionName[:], "Eye gaze")
	if err := check("xrCreateAction", x.createAction(x.gazeSet, &actionInfo, &action)); err != nil {
		return err
	}

	profile, err := x.path("/interaction_profiles/ext/eye_gaze_interaction")
	if err != nil {
		return err
	}
	pose, err := x.path("/user/eyes_ext/input/gaze_ext/pose")
	if err != nil {
		return err
	}
	binding := xrActionSuggestedBinding{Action: action, Binding: pose}
	suggest := xrInteractionProfileSuggestedBinding{
		Type:                   xrTypeInteractionProfileSuggestedBind,
		InteractionProfile:     profile,
		CountSuggestedBindings: 1,
		SuggestedBindings:      &binding,
	}
	if err := check("xrSuggestInteractionProfileBindings", x.suggestInteractionProfileBindings(x.instance, &suggest)); err != nil {
		return err
	}

	attach := xrSessionActionSetsAttachInfo{
		Type:            xrTypeSessionActionSetsAttachInfo,
		CountActionSets: 1,
		ActionSets:      &x.gazeSet,
	}
	if err := check("xrAttachSessionActionSets", x.attachSessionActionSets(x.session, &attach)); err != nil {
		return err
	}

	identity := xrPosef{Orientation: [4]float32{0, 0, 0, 1}}
	gaze := xrActionSpaceCreateInfo{Type: xrTypeActionSpaceCreateInfo, Action: action, PoseInActionSpace: identity}
	if err := check("xrCreateActionSpace", x.createActionSpace(x.session, &gaze, &x.gazeSpace)); err != nil {
		return err
	}
	view := xrReferenceSpaceCreateInfo{
		Type:                 xrTypeReferenceSpaceCreateInfo,
		ReferenceSpaceType:   xrReferenceSpaceTypeView,
		PoseInReferenceSpace: identity,
	}
	return check("xrCreateReferenceSpace", x.createReferenceSpace(x.session, &view, &x.viewSpace))
}

func (x *openxr) pollEvent() (sessionEvent, error) {
	x.buf.event = xrEventDataBuffer{Type: xrTypeEventDataBuffer}
	result := x.pollEventFn(x.instance, &x.buf.event)
	if result == xrEventUnavailable {
		return eventNone, nil
	}
	if err := check("xrPollEvent", result); err != nil {
		return eventNone, err
	}
	switch x.buf.event.Type {
	case xrTypeEventDataInstanceLossPending:
		return eventEnded, nil
	case xrTypeEventDataSessionStateChanged:
		changed := (*xrEventDataSessionStateChanged)(unsafe.Pointer(&x.buf.event))
		if changed.Session != x.session {
			return eventOther, nil
		}
		switch changed.State {
		case xrSessionStateReady:
			return eventReady, nil
		case xrSessionStateStopping:
			return eventStopping, nil
		case xrSessionStateLossPending, xrSessionStateExiting:
			return eventEnded, nil
		}
	}
	return eventOther, nil
}

func (x *openxr) begin() error {
	info := xrSessionBeginInfo{
		Type:                         xrTypeSessionBeginInfo,
		PrimaryViewConfigurationType: xrViewConfigurationTypePrimaryStereo,
	}
	return check("xrBeginSession", x.beginSession(x.session, &info))
}

func (x *openxr) end() error {
	return check("xrEndSession", x.endSession(x.session))
}

// now returns the current time on the runtime's clock.
func (x *openxr) now() (int64, error) {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0, fmt.Errorf("WiVRn: clock_gettime: %w", err)
	}
	var t int64
	c := xrTimespec{Sec: ts.Sec, Nsec: ts.Nsec}
	err := check("xrConvertTimespecTimeToTimeKHR", x.convertTimespecTimeToTime(x.instance, &c, &t))
	return t, err
}

func (x *openxr) read(f *frame) error {
	now, err := x.now()
	if err != nil {
		return err
	}
	*f = frame{}

	if x.gazeSpace != 0 {
		active := xrActiveActionSet{ActionSet: x.gazeSet}
		sync := xrActionsSyncInfo{Type: xrTypeActionsSyncInfo, CountActiveActionSets: 1, ActiveActionSets: &active}
		if err := check("xrSyncActions", x.syncActions(x.session, &sync)); err != nil {
			return err
		}
		loc := xrSpaceLocation{Type: xrTypeSpaceLocation}
		if err := check("xrLocateSpace", x.locateSpace(x.gazeSpace, x.viewSpace, now, &loc)); err != nil {
			return err
		}
		if loc.LocationFlags&xrSpaceLocationOrientationValid != 0 {
			f.gaze = loc.Pose.Orientation
			f.gazeValid = true
		}
	}

	if x.fb2Tracker != 0 {
		info := xrFaceExpressionInfo2FB{Type: xrTypeFaceExpressionInfo2FB, Time: now}
		weights := xrFaceExpressionWeights2FB{
			Type:            xrTypeFaceExpressionWeights2FB,
			WeightCount:     uint32(len(x.buf.fb2Weights)),
			Weights:         &x.buf.fb2Weights[0],
			ConfidenceCount: uint32(len(x.buf.fb2Confidences)),
			Confidences:     &x.buf.fb2Confidences[0],
		}
		if err := check("xrGetFaceExpressionWeights2FB", x.getFaceExpressionWeights2(x.fb2Tracker, &info, &weights)); err != nil {
			return err
		}
		if weights.IsValid != 0 {
			f.fb2 = x.buf.fb2Weights
			f.fb2Valid = true
		}
	}

	if x.htcEyeTracker != 0 {
		expr := xrFacialExpressionsHTC{
			Type:                 xrTypeFacialExpressionsHTC,
			SampleTime:           now,
			ExpressionCount:      uint32(len(x.buf.htcEye)),
			ExpressionWeightings: &x.buf.htcEye[0],
		}
		if err := check("xrGetFacialExpressionsHTC(eye)", x.getFacialExpressionsHTC(x.htcEyeTracker, &expr)); err != nil {
			return err
		}
		if expr.IsActive != 0 {
			f.htcEye = x.buf.htcEye
			f.htcEyeValid = true
		}
	}
	if x.htcLipTracker != 0 {
		expr := xrFacialExpressionsHTC{
			Type:                 xrTypeFacialExpressionsHTC,
			SampleTime:           now,
			ExpressionCount:      uint32(len(x.buf.htcLip)),
			ExpressionWeightings: &x.buf.htcLip[0],
		}
		if err := check("xrGetFacialExpressionsHTC(lip)", x.getFacialExpressionsHTC(x.htcLipTracker, &expr)); err != nil {
			return err
		}
		if expr.IsActive != 0 {
			f.htcLip = x.buf.htcLip
			f.htcLipValid = true
		}
	}

	if x.androidTracker != 0 {
		info := xrFaceStateGetInfoAndroid{Type: xrTypeFaceStateGetInfoAndroid, Time: now}
		state := xrFaceStateAndroid{
			Type:                           xrTypeFaceStateAndroid,
			ParametersCapacityInput:        uint32(len(x.buf.android)),
			Parameters:                     &x.buf.android[0],
			RegionConfidencesCapacityInput: uint32(len(x.buf.androidRegions)),
			RegionConfidences:              &x.buf.androidRegions[0],
		}
		if err := check("xrGetFaceStateANDROID", x.getFaceStateAndroid(x.androidTracker, &info, &state)); err != nil {
			return err
		}
		if state.IsValid != 0 && state.FaceTrackingState == xrFaceTrackingStateTrackingAndroid &&
			state.ParametersCountOutput >= uint32(len(x.buf.android)) {
			f.android = x.buf.android
			f.androidValid = true
		}
	}
	return nil
}

// disconnect destroys the instance, which takes the session, spaces,
// actions and trackers with it.
func (x *openxr) disconnect() {
	if x.instance != 0 {
		x.destroyInstance(x.instance)
	}
	x.instance, x.session = 0, 0
	x.gazeSet, x.gazeSpace, x.viewSpace = 0, 0, 0
	x.fb2Tracker, x.htcEyeTracker, x.htcLipTracker, x.androidTracker = 0, 0, 0, 0
}
