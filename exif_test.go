package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildExifSeg constructs an Exif APP1 segment payload (the bytes after the JPEG
// marker+length) containing a single IFD0 Software tag with the given value.
func buildExifSeg(order binary.ByteOrder, software string) []byte {
	val := append([]byte(software), 0) // ASCII values are NUL-terminated

	tiff := new(bytes.Buffer)
	if order == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	put16 := func(v uint16) { b := make([]byte, 2); order.PutUint16(b, v); tiff.Write(b) }
	put32 := func(v uint32) { b := make([]byte, 4); order.PutUint32(b, v); tiff.Write(b) }

	put16(42)
	put32(8) // IFD0 begins immediately after the 8-byte TIFF header
	// IFD0 (at TIFF offset 8): count, one 12-byte entry, next-IFD offset, value
	put16(1)
	put16(exifSoftwareTag)
	put16(2)                  // type ASCII
	put32(uint32(len(val)))   // count (includes NUL)
	put32(uint32(8 + 2 + 12 + 4)) // value offset (relative to TIFF start) = 26
	put32(0)                  // no next IFD
	tiff.Write(val)

	return append(append([]byte(nil), exifSig...), tiff.Bytes()...)
}

func TestReadExifSoftware(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		seg := buildExifSeg(order, "Adobe Photoshop CS6 (Windows)")
		if got := readExifSoftware(seg); got != "Adobe Photoshop CS6 (Windows)" {
			t.Errorf("readExifSoftware (%v) = %q", order, got)
		}
	}
}

func TestIsAdobeSoftware(t *testing.T) {
	cases := map[string]bool{
		"Adobe Photoshop CS6 (Windows)": true,
		"Adobe Photoshop Lightroom 5.0": true,
		"VueScan x64 9.5.60":            false,
		"GIMP 2.10":                     false,
		"":                              false,
	}
	for in, want := range cases {
		if got := isAdobeSoftware(in); got != want {
			t.Errorf("isAdobeSoftware(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCleanExifSoftware_LengthPreserved(t *testing.T) {
	seg := buildExifSeg(binary.LittleEndian, "Adobe Photoshop CS6 (Windows)")
	orig := len(seg)

	if !cleanExifSoftware(seg, "") {
		t.Fatal("cleanExifSoftware reported no change")
	}
	if len(seg) != orig {
		t.Errorf("segment length changed: %d → %d", orig, len(seg))
	}
	if got := readExifSoftware(seg); got != "" {
		t.Errorf("Software not emptied: %q", got)
	}
	// idempotent: a second pass changes nothing
	if cleanExifSoftware(seg, "") {
		t.Error("second cleanExifSoftware reported a change")
	}
}

func TestInspectAndCleanJPEG_EXIF(t *testing.T) {
	// Adobe Software tag: detected and cleaned.
	jpeg := buildTestJPEGWithExif("Adobe Photoshop CS6 (Windows)")
	rep, err := InspectJPEG(jpeg)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ExifSoftware != "Adobe Photoshop CS6 (Windows)" || !rep.HasAdobeData() {
		t.Fatalf("inspect missed Adobe EXIF Software: %+v", rep)
	}
	modified, out, err := CleanJPEG(jpeg, nil)
	if err != nil || !modified {
		t.Fatalf("CleanJPEG: modified=%v err=%v", modified, err)
	}
	if len(out) != len(jpeg) {
		t.Errorf("JPEG length changed: %d → %d", len(jpeg), len(out))
	}
	rep2, _ := InspectJPEG(out)
	if rep2.HasAdobeData() {
		t.Errorf("EXIF Software not cleaned: %+v", rep2)
	}

	// Non-Adobe Software tag: left untouched.
	scan := buildTestJPEGWithExif("VueScan x64 9.5.60")
	rep3, _ := InspectJPEG(scan)
	if rep3.HasAdobeData() {
		t.Errorf("non-Adobe Software wrongly flagged: %+v", rep3)
	}
	modified, _, _ = CleanJPEG(scan, nil)
	if modified {
		t.Error("CleanJPEG modified a non-Adobe Software tag")
	}
}

// buildTestJPEGWithExif builds a minimal valid JPEG carrying one Exif APP1 segment.
func buildTestJPEGWithExif(software string) []byte {
	var buf bytes.Buffer
	segs := []jpegSeg{{marker: 0xE1, data: buildExifSeg(binary.LittleEndian, software)}}
	if err := writeJPEG(&buf, segs, []byte{0xFF, 0xD9}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
