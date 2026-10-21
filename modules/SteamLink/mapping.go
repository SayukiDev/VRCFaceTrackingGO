package main

import "github.com/SayukiDev/VRCFaceTrackingGO/pkg/tracking"

// OSC addresses Steam Link sends (see docs/SteamLinkSpec.md of LinkFT).
const (
	addrGazePoint   = "/sl/eyeTrackedGazePoint"
	addrEyesClosedL = "/sl/xrfb/facew/EyesClosedL"
	addrEyesClosedR = "/sl/xrfb/facew/EyesClosedR"
)

// faceMapping maps the XR_FB_face_tracking weights Steam Link forwards to
// Unified Expressions, based on
// https://docs.google.com/spreadsheets/d/118jo960co3Mgw8eREFVBsaJ7z0GtKNr52IB4Bz99VTA
var faceMapping = map[string][]tracking.UnifiedExpression{
	"/sl/xrfb/facew/UpperLidRaiserL":  {tracking.EyeWideLeft},
	"/sl/xrfb/facew/UpperLidRaiserR":  {tracking.EyeWideRight},
	"/sl/xrfb/facew/LidTightenerL":    {tracking.EyeSquintLeft},
	"/sl/xrfb/facew/LidTightenerR":    {tracking.EyeSquintRight},
	"/sl/xrfb/facew/InnerBrowRaiserL": {tracking.BrowInnerUpLeft},
	"/sl/xrfb/facew/InnerBrowRaiserR": {tracking.BrowInnerUpRight},
	"/sl/xrfb/facew/OuterBrowRaiserL": {tracking.BrowOuterUpLeft},
	"/sl/xrfb/facew/OuterBrowRaiserR": {tracking.BrowOuterUpRight},
	"/sl/xrfb/facew/BrowLowererL":     {tracking.BrowPinchLeft, tracking.BrowLowererLeft},
	"/sl/xrfb/facew/BrowLowererR":     {tracking.BrowPinchRight, tracking.BrowLowererRight},

	"/sl/xrfb/facew/JawDrop":          {tracking.JawOpen},
	"/sl/xrfb/facew/JawSidewaysLeft":  {tracking.JawLeft},
	"/sl/xrfb/facew/JawSidewaysRight": {tracking.JawRight},
	"/sl/xrfb/facew/JawThrust":        {tracking.JawForward},

	"/sl/xrfb/facew/MouthLeft":  {tracking.MouthLowerLeft, tracking.MouthUpperLeft},
	"/sl/xrfb/facew/MouthRight": {tracking.MouthLowerRight, tracking.MouthUpperRight},

	"/sl/xrfb/facew/ChinRaiserT": {tracking.MouthRaiserUpper},
	"/sl/xrfb/facew/ChinRaiserB": {tracking.MouthRaiserLower},

	"/sl/xrfb/facew/DimplerL": {tracking.MouthDimpleLeft},
	"/sl/xrfb/facew/DimplerR": {tracking.MouthDimpleRight},

	"/sl/xrfb/facew/LipsToward":          {tracking.MouthClosed},
	"/sl/xrfb/facew/LipCornerPullerL":    {tracking.MouthCornerPullLeft, tracking.MouthCornerSlantLeft},
	"/sl/xrfb/facew/LipCornerPullerR":    {tracking.MouthCornerPullRight, tracking.MouthCornerSlantRight},
	"/sl/xrfb/facew/LipCornerDepressorL": {tracking.MouthFrownLeft},
	"/sl/xrfb/facew/LipCornerDepressorR": {tracking.MouthFrownRight},
	"/sl/xrfb/facew/LowerLipDepressorL":  {tracking.MouthLowerDownLeft},
	"/sl/xrfb/facew/LowerLipDepressorR":  {tracking.MouthLowerDownRight},
	"/sl/xrfb/facew/UpperLipRaiserL":     {tracking.MouthUpperUpLeft},
	"/sl/xrfb/facew/UpperLipRaiserR":     {tracking.MouthUpperUpRight},
	"/sl/xrfb/facew/LipTightenerL":       {tracking.MouthTightenerLeft},
	"/sl/xrfb/facew/LipTightenerR":       {tracking.MouthTightenerRight},
	"/sl/xrfb/facew/LipPressorL":         {tracking.MouthPressLeft},
	"/sl/xrfb/facew/LipPressorR":         {tracking.MouthPressRight},
	"/sl/xrfb/facew/LipStretcherL":       {tracking.MouthStretchLeft},
	"/sl/xrfb/facew/LipStretcherR":       {tracking.MouthStretchRight},
	"/sl/xrfb/facew/LipPuckerL":          {tracking.LipPuckerLowerLeft, tracking.LipPuckerUpperLeft},
	"/sl/xrfb/facew/LipPuckerR":          {tracking.LipPuckerLowerRight, tracking.LipPuckerUpperRight},
	"/sl/xrfb/facew/LipFunnelerLB":       {tracking.LipFunnelLowerLeft},
	"/sl/xrfb/facew/LipFunnelerLT":       {tracking.LipFunnelUpperLeft},
	"/sl/xrfb/facew/LipFunnelerRB":       {tracking.LipFunnelLowerRight},
	"/sl/xrfb/facew/LipFunnelerRT":       {tracking.LipFunnelUpperRight},
	"/sl/xrfb/facew/LipSuckLB":           {tracking.LipSuckLowerLeft},
	"/sl/xrfb/facew/LipSuckLT":           {tracking.LipSuckUpperLeft},
	"/sl/xrfb/facew/LipSuckRB":           {tracking.LipSuckLowerRight},
	"/sl/xrfb/facew/LipSuckRT":           {tracking.LipSuckUpperRight},

	"/sl/xrfb/facew/CheekPuffL":   {tracking.CheekPuffLeft},
	"/sl/xrfb/facew/CheekPuffR":   {tracking.CheekPuffRight},
	"/sl/xrfb/facew/CheekSuckL":   {tracking.CheekSuckLeft},
	"/sl/xrfb/facew/CheekSuckR":   {tracking.CheekSuckRight},
	"/sl/xrfb/facew/CheekRaiserL": {tracking.CheekSquintLeft},
	"/sl/xrfb/facew/CheekRaiserR": {tracking.CheekSquintRight},

	"/sl/xrfb/facew/NoseWrinklerL": {tracking.NoseSneerLeft},
	"/sl/xrfb/facew/NoseWrinklerR": {tracking.NoseSneerRight},

	"/sl/xrfb/facew/TongueRetreat":     {tracking.TongueBendDown},
	"/sl/xrfb/facew/TongueTipAlveolar": {tracking.TongueCurlUp},
	"/sl/xrfb/facew/TongueOut":         {tracking.TongueOut},
}
