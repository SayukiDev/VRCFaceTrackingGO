package main

// blendShape indexes the 72 ARKit-style weights PICO streams.
type blendShape int

const (
	eyeLookDownL blendShape = iota
	noseSneerL
	eyeLookInL
	browInnerUp
	browDownR
	mouthClose
	mouthLowerDownR
	jawOpen
	mouthUpperUpR
	mouthShrugUpper
	mouthFunnel
	eyeLookInR
	eyeLookDownR
	noseSneerR
	mouthRollUpper
	jawRight
	browDownL
	mouthShrugLower
	mouthRollLower
	mouthSmileL
	mouthPressL
	mouthSmileR
	mouthPressR
	mouthDimpleR
	mouthLeft
	jawForward
	eyeSquintL
	mouthFrownL
	eyeBlinkL
	cheekSquintL
	browOuterUpL
	eyeLookUpL
	jawLeft
	mouthStretchL
	mouthPucker
	eyeLookUpR
	browOuterUpR
	cheekSquintR
	eyeBlinkR
	mouthUpperUpL
	mouthFrownR
	eyeSquintR
	mouthStretchR
	cheekPuff
	eyeLookOutL
	eyeLookOutR
	eyeWideR
	eyeWideL
	mouthRight
	mouthDimpleL
	mouthLowerDownL
	tongueOut
	visemePP
	visemeCH
	visemeOLower
	visemeO
	visemeI
	visemeULower
	visemeRR
	visemeXX
	visemeAA
	visemeILower
	visemeFF
	visemeU
	visemeTH
	visemeKK
	visemeSS
	visemeE
	visemeDD
	visemeEUpper
	visemeNN
	visemeSil

	blendShapeCount
)
