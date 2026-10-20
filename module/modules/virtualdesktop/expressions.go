package virtualdesktop

// expression indexes faceState.ExpressionWeights (OpenXR / Meta face
// tracking blend shape order as exposed by Virtual Desktop).
type expression int

const (
	browLowererL expression = iota
	browLowererR
	cheekPuffL
	cheekPuffR
	cheekRaiserL
	cheekRaiserR
	cheekSuckL
	cheekSuckR
	chinRaiserB
	chinRaiserT
	dimplerL
	dimplerR
	eyesClosedL
	eyesClosedR
	eyesLookDownL
	eyesLookDownR
	eyesLookLeftL
	eyesLookLeftR
	eyesLookRightL
	eyesLookRightR
	eyesLookUpL
	eyesLookUpR
	innerBrowRaiserL
	innerBrowRaiserR
	jawDrop
	jawSidewaysLeft
	jawSidewaysRight
	jawThrust
	lidTightenerL
	lidTightenerR
	lipCornerDepressorL
	lipCornerDepressorR
	lipCornerPullerL
	lipCornerPullerR
	lipFunnelerLB
	lipFunnelerLT
	lipFunnelerRB
	lipFunnelerRT
	lipPressorL
	lipPressorR
	lipPuckerL
	lipPuckerR
	lipStretcherL
	lipStretcherR
	lipSuckLB
	lipSuckLT
	lipSuckRB
	lipSuckRT
	lipTightenerL
	lipTightenerR
	lipsToward
	lowerLipDepressorL
	lowerLipDepressorR
	mouthLeft
	mouthRight
	noseWrinklerL
	noseWrinklerR
	outerBrowRaiserL
	outerBrowRaiserR
	upperLidRaiserL
	upperLidRaiserR
	upperLipRaiserL
	upperLipRaiserR
	tongueTipInterdental
	tongueTipAlveolar
	tongueFrontDorsalPalate
	tongueMidDorsalPalate
	tongueBackDorsalVelar
	tongueOut
	tongueRetreat
)
