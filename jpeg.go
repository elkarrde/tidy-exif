// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

// JPEG segment parser/writer. Adapted from codeberg.org/elkarrde/lapis.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// xmpSig is the namespace URI that identifies an XMP APP1 segment payload.
var xmpSig = []byte("http://ns.adobe.com/xap/1.0/\x00")

type jpegSeg struct {
	marker byte
	data   []byte
}

func isXMPSeg(s jpegSeg) bool {
	return s.marker == 0xE1 && bytes.HasPrefix(s.data, xmpSig)
}

// parseJPEG reads a complete JPEG and returns its APP segments plus the raw
// tail starting at the SOS payload (compressed image data through EOI).
func parseJPEG(r io.Reader) ([]jpegSeg, []byte, error) {
	all, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	if len(all) < 2 || all[0] != 0xFF || all[1] != 0xD8 {
		return nil, nil, fmt.Errorf("not a JPEG file")
	}

	var segs []jpegSeg
	i := 2 // skip SOI
	for i < len(all) {
		if all[i] != 0xFF {
			return nil, nil, fmt.Errorf("JPEG parse error at offset %d: expected 0xFF", i)
		}
		// skip legal 0xFF padding before marker byte
		for i < len(all) && all[i] == 0xFF {
			i++
		}
		if i >= len(all) {
			break
		}
		m := all[i]
		i++

		// standalone markers (no payload): SOI, EOI, RST0-RST7
		if m == 0xD8 || m == 0xD9 || (m >= 0xD0 && m <= 0xD7) {
			if m == 0xD9 {
				return segs, []byte{0xFF, 0xD9}, nil
			}
			continue
		}

		if i+2 > len(all) {
			return nil, nil, fmt.Errorf("truncated JPEG at marker 0xFF%02X", m)
		}
		length := int(binary.BigEndian.Uint16(all[i:]))
		if length < 2 || i+length > len(all) {
			return nil, nil, fmt.Errorf("invalid segment length at 0xFF%02X: %d", m, length)
		}
		payload := make([]byte, length-2)
		copy(payload, all[i+2:i+length])

		if m == 0xDA { // SOS: compressed image data follows the header
			segs = append(segs, jpegSeg{marker: m, data: payload})
			return segs, all[i+length:], nil
		}

		segs = append(segs, jpegSeg{marker: m, data: payload})
		i += length
	}
	return segs, nil, nil
}

// writeJPEG emits a JPEG from its segment list and raw tail (image data + EOI).
func writeJPEG(w io.Writer, segs []jpegSeg, tail []byte) error {
	if _, err := w.Write([]byte{0xFF, 0xD8}); err != nil {
		return err
	}
	hdr := make([]byte, 4)
	for _, s := range segs {
		hdr[0] = 0xFF
		hdr[1] = s.marker
		binary.BigEndian.PutUint16(hdr[2:], uint16(len(s.data)+2))
		if _, err := w.Write(hdr); err != nil {
			return err
		}
		if _, err := w.Write(s.data); err != nil {
			return err
		}
	}
	_, err := w.Write(tail)
	return err
}
