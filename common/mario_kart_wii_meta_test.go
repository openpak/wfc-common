package common

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

// putBits writes value into buf as an MSB-first bit field starting at byteOffset*8+bitOffset,
// the layout the RKG documents use. It is deliberately independent of RKGhostData.GetBits.
func putBits(buf []byte, byteOffset, bitOffset, bitLength int, value uint32) {
	start := byteOffset*8 + bitOffset
	for i := 0; i < bitLength; i++ {
		bit := (value >> uint(bitLength-1-i)) & 1
		pos := start + i
		if bit == 1 {
			buf[pos/8] |= 0x80 >> uint(pos%8)
		} else {
			buf[pos/8] &^= 0x80 >> uint(pos%8)
		}
	}
}

func putTime(buf []byte, byteOffset, bitOffset, m, s, ms int) {
	putBits(buf, byteOffset, bitOffset, 7, uint32(m))
	putBits(buf, byteOffset, bitOffset+7, 7, uint32(s))
	putBits(buf, byteOffset, bitOffset+14, 10, uint32(ms))
}

// syntheticRKG builds a header-only RKG from the documented layout.
func syntheticRKG() []byte {
	f := make([]byte, 0x8C+0x10)
	copy(f, "RKGD")
	putTime(f, 0x04, 0, 1, 23, 456)
	putBits(f, 0x07, 0, 6, 0x0D) // Rainbow Road
	putBits(f, 0x08, 0, 6, 0x0F) // Blue Falcon
	putBits(f, 0x08, 6, 6, 0x07) // Luigi
	putBits(f, 0x09, 4, 7, 24)   // 2024
	putBits(f, 0x0A, 3, 4, 11)
	putBits(f, 0x0A, 7, 5, 9)
	putBits(f, 0x0B, 4, 4, 2) // Classic Controller
	putBits(f, 0x0C, 4, 1, 1)
	putBits(f, 0x0C, 7, 7, 1)
	putBits(f, 0x0D, 6, 1, 1)
	putBits(f, 0x0E, 0, 16, 0x1234)
	putBits(f, 0x10, 0, 8, 3)
	putTime(f, 0x11, 0, 0, 27, 100)
	putTime(f, 0x14, 0, 0, 28, 200)
	putTime(f, 0x17, 0, 0, 27, 156)
	f[0x34], f[0x35] = 78, 2
	binary.BigEndian.PutUint16(f[0x36:], 0x0102)
	for i, u := range utf16.Encode([]rune("Ana")) {
		binary.BigEndian.PutUint16(f[0x3C+2+i*2:], u)
	}
	return f
}

func TestParseRKGMetadata(t *testing.T) {
	m, err := ParseRKGMetadata(syntheticRKG())
	if err != nil {
		t.Fatal(err)
	}
	if m.FinishTimeMS != 83456 || FormatMKWTime(m.FinishTimeMS) != "1:23.456" {
		t.Fatalf("time %d", m.FinishTimeMS)
	}
	if len(m.LapTimesMS) != 3 || m.LapTimesMS[0] != 27100 || m.LapTimesMS[1] != 28200 || m.LapTimesMS[2] != 27156 {
		t.Fatalf("laps %v", m.LapTimesMS)
	}
	if m.Course != RainbowRoad || m.Course.Name() != "Rainbow Road" {
		t.Fatalf("course %v", m.Course)
	}
	if m.Vehicle != BlueFalcon || m.Vehicle.Name() != "Blue Falcon" || m.Character != Luigi || m.Character.Name() != "Luigi" {
		t.Fatalf("kart %v %v", m.Vehicle, m.Character)
	}
	if m.Controller != Classic || m.Controller.Name() != "Classic Controller" {
		t.Fatalf("controller %v", m.Controller)
	}
	if m.Year != 2024 || m.Month != 11 || m.Day != 9 {
		t.Fatalf("date %d-%d-%d", m.Year, m.Month, m.Day)
	}
	if !m.Compressed || m.GhostType != 1 || m.DriftType != 1 || m.InputDataLength != 0x1234 || m.LapCount != 3 {
		t.Fatalf("flags %+v", m)
	}
	if m.CountryCode != 78 || m.StateCode != 2 || m.LocationCode != 0x0102 {
		t.Fatalf("location %+v", m)
	}
	if m.MiiName != "Ana" {
		t.Fatalf("name %q", m.MiiName)
	}
}

func TestParseRKGMetadataRejects(t *testing.T) {
	if _, err := ParseRKGMetadata(make([]byte, 10)); err != ErrRKGTooShort {
		t.Fatal(err)
	}
	if _, err := ParseRKGMetadata(make([]byte, 0x90)); err != ErrRKGMagic {
		t.Fatal(err)
	}
}
