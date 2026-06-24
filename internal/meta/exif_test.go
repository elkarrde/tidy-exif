// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package meta

import (
	"bytes"
	"encoding/binary"
	"testing"

	"codeberg.org/elkarrde/exifscalpel/exif"
	"codeberg.org/elkarrde/exifscalpel/jpeg"
)

// The EXIF/TIFF parse and edit engine is tested in exifscalpel/exif (both byte
// orders, inline vs offset values, in-place vs rebuild). Here we test tidy-exif
// policy: the Adobe-only gate and the InspectJPEG/CleanJPEG orchestration.

// buildExifSeg constructs an Exif APP1 segment payload containing a single IFD0
// Software tag with the given value, via the library's rebuild path.
func buildExifSeg(order binary.ByteOrder, software string) []byte {
	val := append([]byte(software), 0) // ASCII values are NUL-terminated
	d := &exif.Data{
		ByteOrder: order,
		IFD0:      []exif.Entry{{Tag: exif.SoftwareTag, Type: 2, Count: uint32(len(val)), Value: val}},
	}
	payload, err := d.Build()
	if err != nil {
		panic(err)
	}
	return payload
}

// buildTestJPEGWithExif builds a minimal valid JPEG carrying one Exif APP1 segment.
func buildTestJPEGWithExif(software string) []byte {
	var buf bytes.Buffer
	segs := []jpeg.Segment{{Marker: 0xE1, Data: buildExifSeg(binary.LittleEndian, software)}}
	if err := jpeg.Write(&buf, segs, []byte{0xFF, 0xD9}); err != nil {
		panic(err)
	}
	return buf.Bytes()
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

func TestInspectAndCleanJPEG_EXIF(t *testing.T) {
	// Adobe Software tag: detected and cleaned.
	jpegData := buildTestJPEGWithExif("Adobe Photoshop CS6 (Windows)")
	rep, err := InspectJPEG(jpegData)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ExifSoftware != "Adobe Photoshop CS6 (Windows)" || !rep.HasAdobeData() {
		t.Fatalf("inspect missed Adobe EXIF Software: %+v", rep)
	}
	modified, out, err := CleanJPEG(jpegData, nil)
	if err != nil || !modified {
		t.Fatalf("CleanJPEG: modified=%v err=%v", modified, err)
	}
	if len(out) != len(jpegData) {
		t.Errorf("JPEG length changed: %d → %d", len(jpegData), len(out))
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
