package store

import "crypto/rand"

func newTestUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatTestUUID(b)
}

func formatTestUUID(b []byte) string {
	return hexTest(b[0:4]) + "-" + hexTest(b[4:6]) + "-" + hexTest(b[6:8]) + "-" + hexTest(b[8:10]) + "-" + hexTest(b[10:16])
}

func hexTest(b []byte) string {
	const hextable = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, x := range b {
		out = append(out, hextable[x>>4], hextable[x&0x0f])
	}
	return string(out)
}
