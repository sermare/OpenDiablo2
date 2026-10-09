// Package d2compression holds the decompressors the MPQ reader needs for sound: an
// adaptive Huffman decoder (huffman.go) and WavDecompress, the ADPCM step of the compressed WAV
// sector format (wav.go). d2mpq calls it for blocks that carry the WAV compression flags.
// Ported from community MPQ documentation; correctness is shown indirectly by the real
// archives' sounds decoding, there is no byte comparison test against the original decoder.
package d2compression
