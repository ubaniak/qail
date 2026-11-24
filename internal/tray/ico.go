package tray

import (
	"bytes"
	"encoding/binary"
	"image/png"
)

// pngToICO wraps a PNG in a single-image .ico container, which is what
// the Windows tray expects. ICO has carried PNG payloads since Vista.
func pngToICO(p []byte) []byte {
	w, h := 0, 0
	if cfg, err := png.DecodeConfig(bytes.NewReader(p)); err == nil {
		w, h = cfg.Width, cfg.Height
	}
	dim := func(n int) byte {
		if n <= 0 || n >= 256 {
			return 0 // 0 means 256 (or unknown) in an ICO directory entry
		}
		return byte(n)
	}

	var buf bytes.Buffer
	// ICONDIR: reserved, type (1 = icon), count.
	_ = binary.Write(&buf, binary.LittleEndian, [3]uint16{0, 1, 1})
	// ICONDIRENTRY.
	buf.WriteByte(dim(w))
	buf.WriteByte(dim(h))
	buf.WriteByte(0)                                        // palette size
	buf.WriteByte(0)                                        // reserved
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // colour planes
	_ = binary.Write(&buf, binary.LittleEndian, uint16(32)) // bits per pixel
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p)))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(6+16)) // offset of the image data
	buf.Write(p)
	return buf.Bytes()
}
