// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// XMP namespace URIs used to identify fields in the token stream.
const (
	nsXMP   = "http://ns.adobe.com/xap/1.0/"
	nsXMPMM = "http://ns.adobe.com/xap/1.0/mm/"
	nsStEvt = "http://ns.adobe.com/xap/1.0/sType/ResourceEvent#"
	nsRDF   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
)

// XMPData holds the Adobe-specific metadata fields targeted for removal.
type XMPData struct {
	CreatorTool        string
	MetadataDate       string
	DocumentID         string
	InstanceID         string
	OriginalDocumentID string
	SoftwareAgents     []string // one entry per xmpMM:History item
}

// HasAdobeData reports whether any target field contains a non-empty value.
func (x *XMPData) HasAdobeData() bool {
	if x.CreatorTool != "" || x.MetadataDate != "" || x.DocumentID != "" ||
		x.InstanceID != "" || x.OriginalDocumentID != "" {
		return true
	}
	for _, a := range x.SoftwareAgents {
		if a != "" {
			return true
		}
	}
	return false
}

// parseXMP extracts target fields from a raw XMP APP1 payload using an
// encoding/xml token decoder. Handles both attribute and element form of each field.
func parseXMP(segment []byte) (*XMPData, error) {
	if !bytes.HasPrefix(segment, xmpSig) {
		return nil, fmt.Errorf("not an XMP segment")
	}

	d := &XMPData{}
	dec := xml.NewDecoder(bytes.NewReader(segment[len(xmpSig):]))
	dec.Strict = false

	inHistory := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break // return whatever we extracted before the error
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Space == nsRDF && t.Name.Local == "Description":
				for _, a := range t.Attr {
					switch {
					case matchAttr(a, nsXMP, "CreatorTool", "xmp"):
						d.CreatorTool = a.Value
					case matchAttr(a, nsXMP, "MetadataDate", "xmp"):
						d.MetadataDate = a.Value
					case matchAttr(a, nsXMPMM, "DocumentID", "xmpMM"):
						d.DocumentID = a.Value
					case matchAttr(a, nsXMPMM, "InstanceID", "xmpMM"):
						d.InstanceID = a.Value
					case matchAttr(a, nsXMPMM, "OriginalDocumentID", "xmpMM"):
						d.OriginalDocumentID = a.Value
					}
				}

			// Element form of simple fields (less common but valid XMP)
			case t.Name.Space == nsXMP && t.Name.Local == "CreatorTool":
				d.CreatorTool = nextCharData(dec)
			case t.Name.Space == nsXMP && t.Name.Local == "MetadataDate":
				d.MetadataDate = nextCharData(dec)
			case t.Name.Space == nsXMPMM && t.Name.Local == "DocumentID":
				d.DocumentID = nextCharData(dec)
			case t.Name.Space == nsXMPMM && t.Name.Local == "InstanceID":
				d.InstanceID = nextCharData(dec)
			case t.Name.Space == nsXMPMM && t.Name.Local == "OriginalDocumentID":
				d.OriginalDocumentID = nextCharData(dec)

			case t.Name.Space == nsXMPMM && t.Name.Local == "History":
				inHistory = true
			case inHistory && t.Name.Space == nsStEvt && t.Name.Local == "softwareAgent":
				// element form: <stEvt:softwareAgent>value</stEvt:softwareAgent>
				d.SoftwareAgents = append(d.SoftwareAgents, nextCharData(dec))
			case inHistory:
				// attribute form (what Lightroom/Photoshop actually write):
				// <rdf:li ... stEvt:softwareAgent="value" .../>
				for _, a := range t.Attr {
					if matchAttr(a, nsStEvt, "softwareAgent", "stEvt") {
						d.SoftwareAgents = append(d.SoftwareAgents, a.Value)
					}
				}
			}

		case xml.EndElement:
			if t.Name.Space == nsXMPMM && t.Name.Local == "History" {
				inHistory = false
			}
		}
	}

	return d, nil
}

// matchAttr reports whether an xml.Attr matches the given namespace and local name.
// It handles both the URI-resolved form (proper XML namespace handling) and the
// prefixed local-name form that Go's encoding/xml may produce for attributes.
func matchAttr(a xml.Attr, ns, local, prefix string) bool {
	return (a.Name.Space == ns && a.Name.Local == local) ||
		(a.Name.Space == "" && a.Name.Local == prefix+":"+local)
}

// nextCharData reads the next decoder token and returns it as a trimmed string
// if it is character data. Returns "" for empty elements or on any error.
func nextCharData(dec *xml.Decoder) string {
	tok, err := dec.Token()
	if err != nil {
		return ""
	}
	if cd, ok := tok.(xml.CharData); ok {
		return strings.TrimSpace(string(cd))
	}
	return ""
}

// cleanXMP returns a new XMPData with all target fields replaced.
// replacements maps field names (e.g. "CreatorTool", "SoftwareAgent") to
// replacement values; any field absent from the map is set to "".
func cleanXMP(xmp *XMPData, replacements map[string]string) *XMPData {
	repl := func(key string) string {
		if v, ok := replacements[key]; ok {
			return v
		}
		return ""
	}
	agents := make([]string, len(xmp.SoftwareAgents))
	for i := range xmp.SoftwareAgents {
		agents[i] = repl("SoftwareAgent")
	}
	return &XMPData{
		CreatorTool:        repl("CreatorTool"),
		MetadataDate:       repl("MetadataDate"),
		DocumentID:         repl("DocumentID"),
		InstanceID:         repl("InstanceID"),
		OriginalDocumentID: repl("OriginalDocumentID"),
		SoftwareAgents:     agents,
	}
}

// marshalXMP applies cleaned field values back to the original raw segment bytes
// via targeted regexp replacement, then adjusts xpacket padding to preserve length.
func marshalXMP(original []byte, cleaned *XMPData) ([]byte, error) {
	if !bytes.HasPrefix(original, xmpSig) {
		return nil, fmt.Errorf("not an XMP segment")
	}

	xmlPart := make([]byte, len(original)-len(xmpSig))
	copy(xmlPart, original[len(xmpSig):])

	fields := []struct{ name, value string }{
		{"xmp:CreatorTool", cleaned.CreatorTool},
		{"xmp:MetadataDate", cleaned.MetadataDate},
		{"xmpMM:DocumentID", cleaned.DocumentID},
		{"xmpMM:InstanceID", cleaned.InstanceID},
		{"xmpMM:OriginalDocumentID", cleaned.OriginalDocumentID},
	}
	for _, f := range fields {
		xmlPart = patchField(xmlPart, f.name, f.value)
	}
	// All history entries are replaced with the same value (see cleanXMP), so a
	// single global replacement across attribute and element forms suffices.
	agentRepl := ""
	if len(cleaned.SoftwareAgents) > 0 {
		agentRepl = cleaned.SoftwareAgents[0]
	}
	xmlPart = patchAll(xmlPart, "stEvt:softwareAgent", agentRepl)

	patched := append(append([]byte(nil), xmpSig...), xmlPart...)
	return adjustPadding(patched, len(original))
}

// patchField replaces the value of a named XMP field in raw XML bytes.
// Handles attribute form (name="value") and element form (<name>value</name>).
func patchField(xmlBytes []byte, name, replacement string) []byte {
	qn := regexp.QuoteMeta(name)
	// attribute, double-quoted
	if re := regexp.MustCompile(qn + `="[^"]*"`); re.Match(xmlBytes) {
		return re.ReplaceAll(xmlBytes, []byte(name+`="`+replacement+`"`))
	}
	// attribute, single-quoted
	if re := regexp.MustCompile(qn + `='[^']*'`); re.Match(xmlBytes) {
		return re.ReplaceAll(xmlBytes, []byte(name+`='`+replacement+`'`))
	}
	// element form
	if re := regexp.MustCompile(`<` + qn + `>[^<]*</` + qn + `>`); re.Match(xmlBytes) {
		return re.ReplaceAll(xmlBytes, []byte(`<`+name+`>`+replacement+`</`+name+`>`))
	}
	return xmlBytes
}

// patchAll replaces the value of every occurrence of a field in raw XML bytes,
// in both attribute form (name="value" / name='value') and element form
// (<name>value</name>). Used for repeated fields such as history softwareAgent.
func patchAll(xmlBytes []byte, name, replacement string) []byte {
	qn := regexp.QuoteMeta(name)
	out := xmlBytes
	out = regexp.MustCompile(qn+`="[^"]*"`).ReplaceAll(out, []byte(name+`="`+replacement+`"`))
	out = regexp.MustCompile(qn+`='[^']*'`).ReplaceAll(out, []byte(name+`='`+replacement+`'`))
	out = regexp.MustCompile(`<`+qn+`>[^<]*</`+qn+`>`).ReplaceAll(out, []byte(`<`+name+`>`+replacement+`</`+name+`>`))
	return out
}

// adjustPadding expands the xpacket padding in data to reach exactly targetLen bytes.
// XMP padding lives between </x:xmpmeta> and <?xpacket end=...?>.
func adjustPadding(data []byte, targetLen int) ([]byte, error) {
	diff := targetLen - len(data)
	if diff == 0 {
		return data, nil
	}
	if diff < 0 {
		return nil, fmt.Errorf("patched XMP is %d bytes larger than original segment", -diff)
	}

	extra := bytes.Repeat([]byte{' '}, diff)

	endMetaIdx := bytes.Index(data, []byte("</x:xmpmeta>"))
	endPktIdx := bytes.Index(data, []byte("<?xpacket end"))

	insertAt := -1
	switch {
	case endMetaIdx >= 0 && endPktIdx > endMetaIdx:
		insertAt = endPktIdx // standard xpacket padding area
	case endPktIdx >= 0:
		insertAt = endPktIdx // no end-meta tag; insert before end-packet anyway
	}

	if insertAt >= 0 {
		out := make([]byte, 0, targetLen)
		out = append(out, data[:insertAt]...)
		out = append(out, extra...)
		out = append(out, data[insertAt:]...)
		return out, nil
	}

	// No xpacket structure found; pad at end as a last resort.
	return append(data, extra...), nil
}

// ParseXMPFromJPEG extracts XMP data from raw JPEG bytes.
// Returns nil, nil if no XMP segment is present.
func ParseXMPFromJPEG(jpegData []byte) (*XMPData, error) {
	segs, _, err := parseJPEG(bytes.NewReader(jpegData))
	if err != nil {
		return nil, err
	}
	for _, seg := range segs {
		if isXMPSeg(seg) {
			return parseXMP(seg.data)
		}
	}
	return nil, nil
}

// CleanXMPInJPEG applies XMP field cleaning to raw JPEG bytes.
// Returns (modified, resultBytes, error). If no XMP segment is found or all
// target fields are already empty, modified is false and original bytes are returned.
func CleanXMPInJPEG(jpegData []byte, replacements map[string]string) (bool, []byte, error) {
	segs, tail, err := parseJPEG(bytes.NewReader(jpegData))
	if err != nil {
		return false, nil, err
	}

	modified := false
	for i, seg := range segs {
		if !isXMPSeg(seg) {
			continue
		}
		xmp, err := parseXMP(seg.data)
		if err != nil {
			return false, nil, fmt.Errorf("XMP parse: %w", err)
		}
		if !xmp.HasAdobeData() {
			continue
		}
		cleaned := cleanXMP(xmp, replacements)
		patched, err := marshalXMP(seg.data, cleaned)
		if err != nil {
			return false, nil, fmt.Errorf("XMP marshal: %w", err)
		}
		segs[i].data = patched
		modified = true
	}

	if !modified {
		return false, jpegData, nil
	}

	var buf bytes.Buffer
	if err := writeJPEG(&buf, segs, tail); err != nil {
		return false, nil, err
	}
	return true, buf.Bytes(), nil
}
