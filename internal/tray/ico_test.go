package tray

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPNGToICO(t *testing.T) {
	ico := pngToICO(iconPNG)
	var hdr [3]uint16
	if err := binary.Read(bytes.NewReader(ico[:6]), binary.LittleEndian, &hdr); err != nil {
		t.Fatal(err)
	}
	if hdr != [3]uint16{0, 1, 1} {
		t.Fatalf("header = %v, want [0 1 1]", hdr)
	}
	if ico[6] != 44 || ico[7] != 44 {
		t.Fatalf("size = %dx%d, want 44x44", ico[6], ico[7])
	}
	size := binary.LittleEndian.Uint32(ico[14:18])
	off := binary.LittleEndian.Uint32(ico[18:22])
	if int(size) != len(iconPNG) || off != 22 {
		t.Fatalf("size=%d off=%d, want %d 22", size, off, len(iconPNG))
	}
	if !bytes.Equal(ico[22:], iconPNG) {
		t.Fatal("payload is not the PNG")
	}
}
