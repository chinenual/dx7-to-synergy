package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"

	"github.com/pkg/errors"

	"github.com/chinenual/synergize/data"
	"github.com/orcaman/writerseeker"
)

// Helper routines that may find their way back into the synergize/data module.

func helperBlankVce() (vce data.VCE, err error) {
	rdr := bytes.NewReader(data.VRAM_EDATA[data.Off_VRAM_EDATA:])
	if vce, err = data.ReadVce(rdr, false); err != nil {
		return
	}
	// re-allocate the Envelopes and each env Table to allow us to control size
	//and #osc simply by writing to VOITAB and NPOINTS params
	for i := 1; i < 16; i++ {
		// make a copy of the first osc:
		vce.Envelopes = append(vce.Envelopes, vce.Envelopes[0])
	}
	vce.Filters = make([][32]int8, 16)
	for i := 0; i < 16; i++ {
		// now re-allocate each envelope to their max possible length:
		vce.Envelopes[i].AmpEnvelope.Table = make([]byte, 4*16)
		vce.Envelopes[i].FreqEnvelope.Table = make([]byte, 4*16)
	}
	return
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func helperSetPatchType(vce *data.VCE, patchType int) {
	for i := range data.PatchTypePerOscTable[patchType-1] {
		vce.Envelopes[i].FreqEnvelope.OPTCH = data.PatchTypePerOscTable[patchType-1][i]
	}
}

var dxAlgoNoFeedbackPatchTypePerOscTable = [32][16]byte{
	{100, 97, 97, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 1 (algo 0)
	{100, 97, 97, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 2 (algo 1)
	{100, 97, 1, 100, 97, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 3 (algo 2)
	{100, 97, 1, 100, 97, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 4 (algo 3)
	{100, 1, 100, 1, 100, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4},   // DX 5 (algo 4)
	{100, 1, 100, 1, 100, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4},   // DX 6 (algo 5)
	{100, 97, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 7 (algo 6)
	{100, 97, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 8 (algo 7)
	{100, 97, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 9 (algo 8)
	{100, 76, 1, 100, 97, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 10 (algo 9)
	{100, 76, 1, 100, 97, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 11 (algo 10)
	{100, 76, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 12 (algo 11)
	{100, 76, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 13 (algo 12)
	{100, 76, 97, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 14 (algo 13)
	{100, 76, 97, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 15 (algo 14)
	{100, 161, 100, 145, 148, 2, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4}, // DX 16 (algo 15)
	{100, 161, 100, 145, 148, 2, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4}, // DX 17 (algo 16)
	{100, 97, 97, 76, 76, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},     // DX 18 (algo 17)
	{100, 1, 1, 100, 97, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},      // DX 19 (algo 18)
	{100, 76, 1, 100, 1, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},      // DX 20 (algo 19)
	{100, 1, 1, 100, 1, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},       // DX 21 (algo 20)
	{100, 1, 1, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},       // DX 22 (algo 21)
	{100, 1, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},       // DX 23 (algo 22)
	{100, 1, 1, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},         // DX 24 (algo 23)
	{100, 1, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},         // DX 25 (algo 24)
	{100, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},      // DX 26 (algo 25)
	{100, 76, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},      // DX 27 (algo 26)
	{4, 100, 97, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},      // DX 28 (algo 27)
	{100, 1, 100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},       // DX 29 (algo 28)
	{4, 100, 97, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},        // DX 30 (algo 29)
	{100, 1, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},         // DX 31 (algo 30)
	{4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},           // DX 32 (algo 31)
}

func helperSetAlgorithmPatchType(vce *data.VCE, dxAlgo byte, dxFeedback byte) (err error) {
	if dxFeedback != 0 {
		log.Printf("WARNING: Limitation: unhandled DX feedback: %d", dxFeedback)
	}
	if dxAlgo < 0 || dxAlgo > byte(len(dxAlgoNoFeedbackPatchTypePerOscTable)) {
		return errors.Errorf("Invalid Algorithm value %d - expected 0 .. 31", dxAlgo)
	}

	for i := range dxAlgoNoFeedbackPatchTypePerOscTable[dxAlgo] {
		vce.Envelopes[i].FreqEnvelope.OPTCH = dxAlgoNoFeedbackPatchTypePerOscTable[dxAlgo][i]
	}

	return
}

func helperCompactVCE(vce data.VCE) (compacted data.VCE, err error) {
	var writebuf = writerseeker.WriterSeeker{}

	if err = data.WriteVce(&writebuf, vce, data.VceName(vce.Head), false); err != nil {
		return
	}
	writeBytes, _ := ioutil.ReadAll(writebuf.Reader())

	var readbuf2 = bytes.NewReader(writeBytes)

	if compacted, err = data.ReadVce(readbuf2, false); err != nil {
		return
	}
	return
}

func helperVCEToJSON(vce data.VCE) (result string) {
	// compact the vce before printing it:
	var err error
	var compacted data.VCE
	if compacted, err = helperCompactVCE(vce); err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}

	b, _ := json.MarshalIndent(compacted, "", "\t")
	result = string(b)
	return
}

// translations of the javascript functions in viewVCE_envs.js

var _freqTimeScale = []int{0, 1, 2, 3, 4, 5, 6, 7,
	8, 9, 10, 11, 12, 13, 14, 15,
	25, 28, 32, 36, 40, 45, 51, 57,
	64, 72, 81, 91, 102, 115, 129, 145,
	163, 183, 205, 230, 258, 290, 326, 366,
	411, 461, 517, 581, 652, 732, 822, 922,
	1035, 1162, 1304, 1464, 1644, 1845, 2071, 2325,
	2609, 2929, 3288, 3691, 4143, 4650, 5219, 5859,
	6576, 7382, 8286, 9300, 10439, 11718, 13153, 14764,
	16572, 18600, 20078, 23436, 26306, 29528, 29529, 29530,
	29531, 29532, 29533, 29534, 29535}
var _ampTimeScale = []int{0, 1, 2, 3, 4, 5, 6, 7,
	8, 9, 10, 11, 12, 13, 14, 15,
	16, 17, 18, 19, 20, 21, 22, 23,
	24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39,
	40, 45, 51, 57, 64, 72, 81, 91,
	102, 115, 129, 145, 163, 183, 205, 230,
	258, 290, 326, 366, 411, 461, 517, 581,
	652, 732, 822, 922, 1035, 1162, 1304, 1464,
	1644, 1845, 2071, 2325, 2609, 2929, 3288, 3691,
	4143, 4650, 5219, 5859, 6576}

var _freqValues = []int{1, 2, 3, 4, 5, 6, 7, 7, 7, 8, 8, 9, 9, 10, 10, 11, 12, 12,
	13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 25, 26, 28, 30, 31, 33, 35, 37, 40,
	42, 44, 47, 50, 53, 56, 60, 63, 67, 71, 75, 80, 84, 89, 94, 100, 106, 112, 119,
	126, 134, 142, 150, 159, 169, 179, 189, 201, 212, 225, 238, 253, 268, 284, 300,
	318, 337, 357, 378, 401, 425, 450, 477, 505, 536, 568, 601, 637, 675, 715, 757,
	803, 850, 901, 955, 1011, 1072, 1135, 1203, 1274, 1350, 1430, 1515, 1605,
	1701, 1802, 1909, 2023, 2143, 2271, 2405, 2548, 2700, 2861, 3031, 3211,
	3402, 3605, 3818, 4046, 4286, 4541, 4811, 5097, 5400, 5722, 6061, 6422, 6804}

func _indexOfNearestValue(val int, array []int) (index int) {
	bestDiff := math.MaxInt64
	index = -1
	// brute force - we look at the whole array rather than return as soon as we find a minima
	// this is fine since the array is known to be short.
	for i, v := range array {
		diff := val - v
		if diff < 0 {
			// diffs are absolute values
			diff *= -1
		}
		if diff < bestDiff {
			bestDiff = diff
			index = i
		}
	}
	return
}

func helperNearestFreqTimeIndex(val int) (index int) {
	return _indexOfNearestValue(val, _freqTimeScale)
}
func helperNearestAmpTimeIndex(val int) (index int) {
	return _indexOfNearestValue(val, _ampTimeScale)
}

func helperNearestFreqValueIndex(val int) (index int) {
	return _indexOfNearestValue(val, _freqValues)
}

// translate a frequency time "as displayed" to "byte value as stored"
func helperUnscaleFreqTimeValue(time int) byte {
	// fixme: linear search is brute force - but the list is short - performance is "ok" as is...
	for i, v := range _freqTimeScale {
		if v >= time {
			return byte(i)
		}
	}
	// shouldnt happen!
	return byte(len(_freqTimeScale))
}

// translate a amplitude time "as displayed" to "byte value as stored"
func helperUnscaleAmpTimeValue(time int) byte {
	// fixme: linear search is brute force - but the list is short - performance is "ok" as is...
	for i, v := range _ampTimeScale {
		if v >= time {
			return byte(i)
		}
	}
	// shouldnt happen!
	return byte(len(_ampTimeScale))
}

// translate a frequency value "as displayed" to "byte value as stored"
func helperUnscaleFreqEnvValue(val byte) byte {
	return val
}

// translate a amplitude value "as displayed" to "byte value as stored"
func UnscaleAmpEnvValue(val byte) byte {
	return val + 55
}

// translate -- does not handle the "RAND" values, only the numeric ones
func helperUnscaleDetune(val int) int8 {
	// XREF: original source in TextToFDETUN() - javascript in viewVCE_voice.js

	// See FDETUNToText.  This "reverses" that attrocity

	if val >= (-32*3) && val <= (32*3) {
		// CASE B
		val /= 3
	} else if val > 0 {
		// CASE C
		val = ((val / 3) + 32) / 2
	} else {
		// CASE D
		val = ((val / 3) - 32) / 2
	}
	return int8(val)
}
