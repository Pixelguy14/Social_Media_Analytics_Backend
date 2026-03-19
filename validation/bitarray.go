package validation

import "errors"

// ValidateBitArray ensures the binary blob matches the InkToChat constraints.
// 256 * 192 pixels = 49152 bits.
// Since 1 byte = 8 bits, 49152 bits / 8 = 6144 bytes.
// So the byte array must be exactly 6144 bytes long.
func ValidateBitArray(data []byte) error {
	const expectedSize = 6144 // 256 * 192 / 8

	if len(data) != expectedSize {
		return errors.New("invalid drawing dimensions: expected exactly 49152 bits (6144 bytes)")
	}
	return nil
}
