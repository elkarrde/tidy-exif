package main

// Unified inspection and cleaning across both metadata blocks a JPEG may carry:
// the XMP APP1 segment (xmp.go) and the Exif APP1 Software tag (exif.go).

import (
	"bytes"
	"fmt"
)

// FileReport summarises the Adobe-written metadata found in a single JPEG.
type FileReport struct {
	XMP          *XMPData // nil if the file has no XMP segment
	ExifSoftware string   // IFD0 Software tag value, "" if absent
}

// HasAdobeData reports whether any targeted field (XMP or EXIF) holds a value.
func (r *FileReport) HasAdobeData() bool {
	if r.XMP != nil && r.XMP.HasAdobeData() {
		return true
	}
	return r.ExifSoftware != ""
}

// InspectJPEG parses a JPEG's metadata segments without modifying anything.
func InspectJPEG(data []byte) (*FileReport, error) {
	segs, _, err := parseJPEG(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	r := &FileReport{}
	for _, seg := range segs {
		switch {
		case isXMPSeg(seg):
			xmp, err := parseXMP(seg.data)
			if err != nil {
				return nil, fmt.Errorf("XMP parse: %w", err)
			}
			r.XMP = xmp
		case isExifSeg(seg):
			// Only Adobe Software signatures are in scope; leave camera/scanner
			// software (e.g. VueScan, firmware strings) untouched.
			if sw := readExifSoftware(seg.data); isAdobeSoftware(sw) {
				r.ExifSoftware = sw
			}
		}
	}
	return r, nil
}

// CleanJPEG empties (or replaces) Adobe fields in both the XMP segment and the
// Exif Software tag, in a single parse/write. Returns (modified, result, error);
// when nothing changes, modified is false and the original bytes are returned.
func CleanJPEG(data []byte, replacements map[string]string) (bool, []byte, error) {
	segs, tail, err := parseJPEG(bytes.NewReader(data))
	if err != nil {
		return false, nil, err
	}

	modified := false
	for i, seg := range segs {
		switch {
		case isXMPSeg(seg):
			xmp, err := parseXMP(seg.data)
			if err != nil {
				return false, nil, fmt.Errorf("XMP parse: %w", err)
			}
			if !xmp.HasAdobeData() {
				continue
			}
			patched, err := marshalXMP(seg.data, cleanXMP(xmp, replacements))
			if err != nil {
				return false, nil, fmt.Errorf("XMP marshal: %w", err)
			}
			segs[i].data = patched
			modified = true
		case isExifSeg(seg):
			if !isAdobeSoftware(readExifSoftware(seg.data)) {
				continue // leave non-Adobe Software tags (firmware, VueScan, …)
			}
			repl := ""
			if v, ok := replacements["Software"]; ok {
				repl = v
			}
			// cleanExifSoftware edits seg.data in place (a copy owned by the
			// segment list, not the caller's data), preserving its length.
			if cleanExifSoftware(seg.data, repl) {
				modified = true
			}
		}
	}

	if !modified {
		return false, data, nil
	}

	var buf bytes.Buffer
	if err := writeJPEG(&buf, segs, tail); err != nil {
		return false, nil, err
	}
	return true, buf.Bytes(), nil
}
