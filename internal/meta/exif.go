// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package meta

// EXIF (TIFF) handling for the IFD0 Software tag (0x0131), which Adobe tools
// write alongside the XMP signature (e.g. "Adobe Photoshop CS6 (Windows)").
//
// Like the XMP path, edits are length-preserving: the tag's value bytes are
// overwritten in place (NUL-padded) so no TIFF or JPEG offsets need rewriting.

import (
	"bytes"
	"encoding/binary"
	"strings"
)

// exifSig is the prefix of an Exif APP1 segment payload; the TIFF header begins
// immediately after it.
var exifSig = []byte("Exif\x00\x00")

const exifSoftwareTag = 0x0131 // TIFF/EXIF IFD0 "Software" tag

// isExifSeg reports whether a JPEG segment is the Exif APP1 segment.
func isExifSeg(s jpegSeg) bool {
	return s.marker == 0xE1 && bytes.HasPrefix(s.data, exifSig)
}

// exifTIFF locates the TIFF block within an Exif APP1 payload and returns the
// byte order plus the absolute offset (within payload) of the IFD0 entry list.
// found is false when the payload is not a parseable Exif/TIFF structure.
func exifTIFF(payload []byte) (order binary.ByteOrder, ifd0 int, found bool) {
	if !bytes.HasPrefix(payload, exifSig) {
		return nil, 0, false
	}
	tiff := len(exifSig) // TIFF header starts here; all TIFF offsets are relative to it
	if len(payload) < tiff+8 {
		return nil, 0, false
	}
	switch {
	case payload[tiff] == 'I' && payload[tiff+1] == 'I':
		order = binary.LittleEndian
	case payload[tiff] == 'M' && payload[tiff+1] == 'M':
		order = binary.BigEndian
	default:
		return nil, 0, false
	}
	if order.Uint16(payload[tiff+2:]) != 42 {
		return nil, 0, false
	}
	ifd0 = tiff + int(order.Uint32(payload[tiff+4:]))
	if ifd0 < tiff || ifd0+2 > len(payload) {
		return nil, 0, false
	}
	return order, ifd0, true
}

// softwareValueRange returns the byte range [start,end) within payload that
// holds the IFD0 Software tag's ASCII value (including its NUL terminator), or
// found=false if the tag is absent or unparseable.
func softwareValueRange(payload []byte) (start, end int, found bool) {
	order, ifd0, ok := exifTIFF(payload)
	if !ok {
		return 0, 0, false
	}
	tiff := len(exifSig)
	n := int(order.Uint16(payload[ifd0:]))
	for e := 0; e < n; e++ {
		off := ifd0 + 2 + e*12
		if off+12 > len(payload) {
			break
		}
		if order.Uint16(payload[off:]) != exifSoftwareTag {
			continue
		}
		typ := order.Uint16(payload[off+2:])
		count := int(order.Uint32(payload[off+4:]))
		if typ != 2 || count == 0 { // type 2 == ASCII
			return 0, 0, false
		}
		// Values up to 4 bytes are stored inline in the value/offset field;
		// larger values live at an offset relative to the TIFF header.
		if count <= 4 {
			start = off + 8
		} else {
			start = tiff + int(order.Uint32(payload[off+8:]))
		}
		end = start + count
		if start < tiff || end > len(payload) {
			return 0, 0, false
		}
		return start, end, true
	}
	return 0, 0, false
}

// isAdobeSoftware reports whether an EXIF Software value is an Adobe signature.
// Non-Adobe values (camera firmware, scanner software such as VueScan, etc.)
// are legitimate metadata and are left untouched.
func isAdobeSoftware(s string) bool {
	return strings.Contains(strings.ToLower(s), "adobe")
}

// readExifSoftware returns the IFD0 Software value (NUL trimmed) from an Exif
// APP1 payload, or "" if the tag is absent.
func readExifSoftware(payload []byte) string {
	start, end, found := softwareValueRange(payload)
	if !found {
		return ""
	}
	v := payload[start:end]
	if i := bytes.IndexByte(v, 0); i >= 0 {
		v = v[:i]
	}
	return string(bytes.TrimSpace(v))
}

// cleanExifSoftware overwrites the IFD0 Software value in place with replacement
// (NUL-padded to the original length). Returns whether anything changed. The
// replacement is truncated if it would not fit in the original value region.
func cleanExifSoftware(payload []byte, replacement string) bool {
	start, end, found := softwareValueRange(payload)
	if !found {
		return false
	}
	repl := []byte(replacement)
	if len(repl) > (end-start)-1 { // leave room for a NUL terminator
		repl = repl[:(end-start)-1]
	}
	want := make([]byte, end-start)
	copy(want, repl)
	if bytes.Equal(payload[start:end], want) {
		return false // already at target value
	}
	copy(payload[start:end], want)
	return true
}
