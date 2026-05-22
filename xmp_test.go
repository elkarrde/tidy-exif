package main

import (
	"bytes"
	"testing"
)

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

func makeXMPSeg(content string) []byte {
	return append(append([]byte(nil), xmpSig...), []byte(content)...)
}

func buildTestJPEGWithXMP(xmpContent string) []byte {
	payload := append(append([]byte(nil), xmpSig...), []byte(xmpContent)...)
	segLen := uint16(len(payload) + 2)
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xD8})
	buf.Write([]byte{0xFF, 0xE1})
	buf.WriteByte(byte(segLen >> 8))
	buf.WriteByte(byte(segLen))
	buf.Write(payload)
	buf.Write([]byte{0xFF, 0xD9})
	return buf.Bytes()
}

func TestParseXMP(t *testing.T) {
	seg := makeXMPSeg(sampleXMP)
	d, err := parseXMP(seg)
	if err != nil {
		t.Fatal(err)
	}
	if d.CreatorTool != "Adobe Lightroom Classic 13.0" {
		t.Errorf("CreatorTool = %q", d.CreatorTool)
	}
	if d.MetadataDate != "2024-01-15T12:00:00+01:00" {
		t.Errorf("MetadataDate = %q", d.MetadataDate)
	}
	if d.DocumentID != "xmp.did:abc123" {
		t.Errorf("DocumentID = %q", d.DocumentID)
	}
	if d.InstanceID != "xmp.iid:def456" {
		t.Errorf("InstanceID = %q", d.InstanceID)
	}
	if d.OriginalDocumentID != "xmp.did:xyz789" {
		t.Errorf("OriginalDocumentID = %q", d.OriginalDocumentID)
	}
	if len(d.SoftwareAgents) != 2 {
		t.Fatalf("SoftwareAgents len = %d, want 2", len(d.SoftwareAgents))
	}
	if d.SoftwareAgents[0] != "Adobe Lightroom Classic 13.0" {
		t.Errorf("SoftwareAgents[0] = %q", d.SoftwareAgents[0])
	}
	if d.SoftwareAgents[1] != "Adobe Photoshop 2024" {
		t.Errorf("SoftwareAgents[1] = %q", d.SoftwareAgents[1])
	}
}

func TestParseXMPNotXMP(t *testing.T) {
	_, err := parseXMP([]byte("not an xmp segment"))
	if err == nil {
		t.Error("expected error for non-XMP input")
	}
}

func TestCleanXMP(t *testing.T) {
	xmp := &XMPData{
		CreatorTool:        "Adobe Lightroom Classic 13.0",
		MetadataDate:       "2024-01-15",
		DocumentID:         "xmp.did:abc",
		InstanceID:         "xmp.iid:def",
		OriginalDocumentID: "xmp.did:xyz",
		SoftwareAgents:     []string{"Adobe Lightroom Classic 13.0", "Adobe Photoshop 2024"},
	}

	cleaned := cleanXMP(xmp, nil)

	if cleaned.CreatorTool != "" || cleaned.MetadataDate != "" || cleaned.DocumentID != "" ||
		cleaned.InstanceID != "" || cleaned.OriginalDocumentID != "" {
		t.Errorf("cleanXMP did not zero all simple fields: %+v", cleaned)
	}
	for i, a := range cleaned.SoftwareAgents {
		if a != "" {
			t.Errorf("SoftwareAgents[%d] = %q, want empty", i, a)
		}
	}
	// original must be unchanged
	if xmp.CreatorTool == "" {
		t.Error("cleanXMP mutated the input")
	}
}

func TestCleanXMPWithReplacements(t *testing.T) {
	xmp := &XMPData{
		CreatorTool: "Adobe Lightroom Classic 13.0",
		DocumentID:  "xmp.did:abc",
	}
	cleaned := cleanXMP(xmp, map[string]string{"CreatorTool": "cleaned"})
	if cleaned.CreatorTool != "cleaned" {
		t.Errorf("CreatorTool = %q, want %q", cleaned.CreatorTool, "cleaned")
	}
	if cleaned.DocumentID != "" {
		t.Errorf("DocumentID = %q, want empty", cleaned.DocumentID)
	}
}

func TestMarshalXMPPreservesLength(t *testing.T) {
	original := makeXMPSeg(sampleXMP)
	xmp, err := parseXMP(original)
	if err != nil {
		t.Fatal(err)
	}
	result, err := marshalXMP(original, cleanXMP(xmp, nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != len(original) {
		t.Errorf("length: got %d, want %d", len(result), len(original))
	}
}

func TestMarshalXMPZeroesValues(t *testing.T) {
	original := makeXMPSeg(sampleXMP)
	xmp, err := parseXMP(original)
	if err != nil {
		t.Fatal(err)
	}
	result, err := marshalXMP(original, cleanXMP(xmp, nil))
	if err != nil {
		t.Fatal(err)
	}

	reparsed, err := parseXMP(result)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.CreatorTool != "" {
		t.Errorf("CreatorTool after clean = %q", reparsed.CreatorTool)
	}
	if reparsed.DocumentID != "" {
		t.Errorf("DocumentID after clean = %q", reparsed.DocumentID)
	}
	if reparsed.InstanceID != "" {
		t.Errorf("InstanceID after clean = %q", reparsed.InstanceID)
	}
	if reparsed.OriginalDocumentID != "" {
		t.Errorf("OriginalDocumentID after clean = %q", reparsed.OriginalDocumentID)
	}
	for i, a := range reparsed.SoftwareAgents {
		if a != "" {
			t.Errorf("SoftwareAgents[%d] after clean = %q", i, a)
		}
	}
}

func TestCleanXMPInJPEG(t *testing.T) {
	jpeg := buildTestJPEGWithXMP(sampleXMP)

	modified, result, err := CleanXMPInJPEG(jpeg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !modified {
		t.Fatal("expected modified=true")
	}
	if len(result) != len(jpeg) {
		t.Errorf("JPEG size changed: got %d, want %d", len(result), len(jpeg))
	}

	xmp, err := ParseXMPFromJPEG(result)
	if err != nil {
		t.Fatal(err)
	}
	if xmp == nil {
		t.Fatal("no XMP segment in result")
	}
	if xmp.CreatorTool != "" {
		t.Errorf("CreatorTool = %q after clean", xmp.CreatorTool)
	}
	if len(xmp.SoftwareAgents) != 2 {
		t.Errorf("SoftwareAgents len = %d, want 2", len(xmp.SoftwareAgents))
	}
	for i, a := range xmp.SoftwareAgents {
		if a != "" {
			t.Errorf("SoftwareAgents[%d] = %q after clean", i, a)
		}
	}
}

func TestCleanXMPInJPEGNoXMP(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xD9}

	modified, result, err := CleanXMPInJPEG(jpeg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if modified {
		t.Error("expected modified=false for JPEG with no XMP")
	}
	if !bytes.Equal(result, jpeg) {
		t.Error("result differs from input for no-XMP JPEG")
	}
}

func TestHasAdobeData(t *testing.T) {
	empty := &XMPData{}
	if empty.HasAdobeData() {
		t.Error("empty XMPData.HasAdobeData() = true")
	}

	withData := &XMPData{CreatorTool: "Lightroom"}
	if !withData.HasAdobeData() {
		t.Error("XMPData with CreatorTool.HasAdobeData() = false")
	}

	withAgents := &XMPData{SoftwareAgents: []string{"", "Photoshop"}}
	if !withAgents.HasAdobeData() {
		t.Error("XMPData with non-empty SoftwareAgent.HasAdobeData() = false")
	}
}
