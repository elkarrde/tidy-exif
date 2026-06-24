// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package meta

// The EXIF/TIFF parse and length-preserving edit engine now lives in
// codeberg.org/elkarrde/exifscalpel/exif (exif.ReadValue / exif.OverwriteValueInPlace
// on the IFD0 Software tag). What stays here is tidy-exif *policy*: deciding which
// Software values are in scope.

import "strings"

// isAdobeSoftware reports whether an EXIF Software value is an Adobe signature.
// This is the Adobe-only gate: non-Adobe values (camera firmware, scanner
// software such as VueScan, etc.) are legitimate metadata and are left
// untouched. The library deliberately does not make this judgement — it only
// reads and writes the tag — so the gate belongs to tidy-exif.
func isAdobeSoftware(s string) bool {
	return strings.Contains(strings.ToLower(s), "adobe")
}
