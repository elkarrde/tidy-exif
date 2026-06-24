// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package meta

import (
	"bytes"
	"testing"

	"codeberg.org/elkarrde/exifscalpel/jpeg"
)

// The XMP field parser/marshaller (both attribute and element forms, padding
// math) is tested in exifscalpel/xmp. Here we test tidy-exif's JPEG-level
// orchestration: that CleanJPEG / CleanXMPInJPEG drive xmp.Clean correctly and
// preserve segment length. The attribute-form history case below is the
// mandatory regression — Lightroom/Photoshop write softwareAgent as an
// attribute, which the original element-only parser silently missed.

// xmpSig is the Adobe xap namespace signature that prefixes an XMP APP1 payload
// (= jpeg.Segment.Data for an XMP segment). Declared locally to build fixtures.
var xmpSig = []byte("http://ns.adobe.com/xap/1.0/\x00")

// sampleXMP is a representative Lightroom-style XMP block with all six target fields.
const sampleXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:xmp="http://ns.adobe.com/xap/1.0/"
        xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/"
        xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#"
        xmp:CreatorTool="Adobe Lightroom Classic 13.0"
        xmp:MetadataDate="2024-01-15T12:00:00+01:00"
        xmpMM:DocumentID="xmp.did:abc123"
        xmpMM:InstanceID="xmp.iid:def456"
        xmpMM:OriginalDocumentID="xmp.did:xyz789">
      <xmpMM:History>
        <rdf:Seq>
          <rdf:li rdf:parseType="Resource">
            <stEvt:action>saved</stEvt:action>
            <stEvt:softwareAgent>Adobe Lightroom Classic 13.0</stEvt:softwareAgent>
          </rdf:li>
          <rdf:li rdf:parseType="Resource">
            <stEvt:action>saved</stEvt:action>
            <stEvt:softwareAgent>Adobe Photoshop 2024</stEvt:softwareAgent>
          </rdf:li>
        </rdf:Seq>
      </xmpMM:History>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

// attrHistoryXMP uses the attribute form of history entries that Lightroom and
// Photoshop actually write (<rdf:li stEvt:softwareAgent="..."/>). Regression
// fixture: the original parser only handled the element form and missed these.
const attrHistoryXMP = `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:xmp="http://ns.adobe.com/xap/1.0/"
        xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/"
        xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#"
        xmp:CreatorTool="">
      <xmpMM:History>
        <rdf:Seq>
          <rdf:li stEvt:action="saved" stEvt:softwareAgent="Adobe Photoshop Lightroom 5.0 (Windows)" stEvt:when="2017-10-25T00:50:38+02:00"/>
          <rdf:li stEvt:action="saved" stEvt:softwareAgent="Adobe Photoshop CS6 (Windows)" stEvt:when="2017-10-25T01:06:07+02:00"/>
        </rdf:Seq>
      </xmpMM:History>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

func buildTestJPEGWithXMP(xmpContent string) []byte {
	payload := append(append([]byte(nil), xmpSig...), []byte(xmpContent)...)
	var buf bytes.Buffer
	_ = jpeg.Write(&buf, []jpeg.Segment{{Marker: 0xE1, Data: payload}}, []byte{0xFF, 0xD9})
	return buf.Bytes()
}

func TestCleanAttributeHistoryEmptiesAgents(t *testing.T) {
	jpegData := buildTestJPEGWithXMP(attrHistoryXMP)
	modified, out, err := CleanJPEG(jpegData, nil)
	if err != nil || !modified {
		t.Fatalf("CleanJPEG: modified=%v err=%v", modified, err)
	}
	if len(out) != len(jpegData) {
		t.Errorf("length changed: %d → %d", len(jpegData), len(out))
	}
	if bytes.Contains(out, []byte("Adobe Photoshop")) {
		t.Error("attribute-form softwareAgent not emptied")
	}
	rep, _ := InspectJPEG(out)
	if rep.HasAdobeData() {
		t.Errorf("still reports Adobe data after clean: %+v", rep.XMP)
	}
}

func TestCleanXMPInJPEG(t *testing.T) {
	jpegData := buildTestJPEGWithXMP(sampleXMP)

	modified, result, err := CleanXMPInJPEG(jpegData, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !modified {
		t.Fatal("expected modified=true")
	}
	if len(result) != len(jpegData) {
		t.Errorf("JPEG size changed: got %d, want %d", len(result), len(jpegData))
	}

	f, err := ParseXMPFromJPEG(result)
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("no XMP segment in result")
	}
	if f.CreatorTool != "" {
		t.Errorf("CreatorTool = %q after clean", f.CreatorTool)
	}
	if len(f.SoftwareAgents) != 2 {
		t.Errorf("SoftwareAgents len = %d, want 2", len(f.SoftwareAgents))
	}
	for i, a := range f.SoftwareAgents {
		if a != "" {
			t.Errorf("SoftwareAgents[%d] = %q after clean", i, a)
		}
	}
}

func TestCleanXMPInJPEGNoXMP(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xD9}

	modified, result, err := CleanXMPInJPEG(jpegData, nil)
	if err != nil {
		t.Fatal(err)
	}
	if modified {
		t.Error("expected modified=false for JPEG with no XMP")
	}
	if !bytes.Equal(result, jpegData) {
		t.Error("result differs from input for no-XMP JPEG")
	}
}
