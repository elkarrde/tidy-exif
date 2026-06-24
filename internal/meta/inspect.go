// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package meta

// Unified inspection and cleaning across both metadata blocks a JPEG may carry:
// the XMP APP1 segment and the EXIF APP1 Software tag. The byte-level segment
// parse/write, the XMP field surgery, and the EXIF tag edit are all provided by
// codeberg.org/elkarrde/exifscalpel (jpeg/xmp/exif). This file keeps tidy-exif's
// policy: the Adobe-only gate (isAdobeSoftware) and the orchestration of when to
// inspect, clean, and rewrite.

import (
	"bytes"
	"fmt"

	"codeberg.org/elkarrde/exifscalpel/exif"
	"codeberg.org/elkarrde/exifscalpel/jpeg"
	"codeberg.org/elkarrde/exifscalpel/xmp"
)

// FileReport summarises the Adobe-written metadata found in a single JPEG.
type FileReport struct {
	XMP          *xmp.Fields // nil if the file has no XMP segment
	ExifSoftware string      // IFD0 Software tag value, "" if absent
}

// HasAdobeData reports whether any targeted field (XMP or EXIF) holds a value.
func (r *FileReport) HasAdobeData() bool {
	if r.XMP != nil && r.XMP.Any() {
		return true
	}
	return r.ExifSoftware != ""
}

// InspectJPEG parses a JPEG's metadata segments without modifying anything.
func InspectJPEG(data []byte) (*FileReport, error) {
	segs, _, err := jpeg.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	r := &FileReport{}
	for _, seg := range segs {
		switch {
		case jpeg.IsXMP(seg):
			f, err := xmp.Parse(seg.Data)
			if err != nil {
				return nil, fmt.Errorf("XMP parse: %w", err)
			}
			r.XMP = f
		case jpeg.IsEXIF(seg):
			// Only Adobe Software signatures are in scope; leave camera/scanner
			// software (e.g. VueScan, firmware strings) untouched.
			if sw := exif.ReadValue(seg.Data, exif.SoftwareTag); isAdobeSoftware(sw) {
				r.ExifSoftware = sw
			}
		}
	}
	return r, nil
}

// CleanJPEG empties (or replaces) Adobe fields in both the XMP segment and the
// EXIF Software tag, in a single parse/write. Returns (modified, result, error);
// when nothing changes, modified is false and the original bytes are returned.
func CleanJPEG(data []byte, replacements map[string]string) (bool, []byte, error) {
	segs, tail, err := jpeg.Parse(bytes.NewReader(data))
	if err != nil {
		return false, nil, err
	}

	modified := false
	for i, seg := range segs {
		switch {
		case jpeg.IsXMP(seg):
			// xmp.Clean parses, applies replacements, and length-preserves via
			// xpacket padding; it returns changed=false (original bytes) when the
			// segment carries no Adobe data, so no separate guard is needed.
			out, changed, err := xmp.Clean(seg.Data, replacements)
			if err != nil {
				return false, nil, fmt.Errorf("XMP clean: %w", err)
			}
			if changed {
				segs[i].Data = out
				modified = true
			}
		case jpeg.IsEXIF(seg):
			if !isAdobeSoftware(exif.ReadValue(seg.Data, exif.SoftwareTag)) {
				continue // leave non-Adobe Software tags (firmware, VueScan, …)
			}
			repl := ""
			if v, ok := replacements["Software"]; ok {
				repl = v
			}
			// OverwriteValueInPlace edits seg.Data in place (a copy owned by the
			// segment list, not the caller's data), preserving its length.
			changed, err := exif.OverwriteValueInPlace(seg.Data, exif.SoftwareTag, []byte(repl))
			if err != nil {
				return false, nil, fmt.Errorf("EXIF Software clean: %w", err)
			}
			if changed {
				modified = true
			}
		}
	}

	if !modified {
		return false, data, nil
	}

	var buf bytes.Buffer
	if err := jpeg.Write(&buf, segs, tail); err != nil {
		return false, nil, err
	}
	return true, buf.Bytes(), nil
}

// ParseXMPFromJPEG extracts XMP fields from raw JPEG bytes, returning nil when
// the file has no XMP segment. An XMP-only convenience over jpeg.Parse + xmp.Parse.
func ParseXMPFromJPEG(data []byte) (*xmp.Fields, error) {
	segs, _, err := jpeg.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	for _, seg := range segs {
		if jpeg.IsXMP(seg) {
			return xmp.Parse(seg.Data)
		}
	}
	return nil, nil
}

// CleanXMPInJPEG cleans only the XMP segment of a JPEG, leaving EXIF untouched.
// Returns (modified, result, error); when nothing changes, modified is false and
// the original bytes are returned.
func CleanXMPInJPEG(data []byte, replacements map[string]string) (bool, []byte, error) {
	segs, tail, err := jpeg.Parse(bytes.NewReader(data))
	if err != nil {
		return false, nil, err
	}
	modified := false
	for i, seg := range segs {
		if !jpeg.IsXMP(seg) {
			continue
		}
		out, changed, err := xmp.Clean(seg.Data, replacements)
		if err != nil {
			return false, nil, fmt.Errorf("XMP clean: %w", err)
		}
		if changed {
			segs[i].Data = out
			modified = true
		}
	}
	if !modified {
		return false, data, nil
	}
	var buf bytes.Buffer
	if err := jpeg.Write(&buf, segs, tail); err != nil {
		return false, nil, err
	}
	return true, buf.Bytes(), nil
}
