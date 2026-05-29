package uint256

// GetByte returns the n-th least significant byte of v (n=0 is LSB).
func GetByte(v uint64, n uint) byte {
	return byte(v >> (n * 8))
}

// SetByte returns v with its n-th least significant byte replaced by b.
func SetByte(v uint64, n uint, b byte) uint64 {
	shift := n * 8
	return (v &^ (0xFF << shift)) | (uint64(b) << shift)
}
