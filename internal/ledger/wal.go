package ledger

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	walMagic   = []byte{0x41, 0x55, 0x44, 0x46} // "AUDF"
	walVersion = byte(0x01)
	walTrailer = []byte{0x5A, 0x5A} // "ZZ"

	ErrCorruptFrame   = errors.New("corrupt wal frame: magic or boundary trailer mismatch")
	ErrChecksumFailed = errors.New("wal frame checksum verification failed")
	ErrTruncatedFrame = errors.New("unexpected end of wal file while reading frame")
)

// WAL represents a thread-safe, strictly append-only Write-Ahead Log file
type WAL struct {
	mu           sync.RWMutex
	file         *os.File
	path         string
	lastOffset   int64
	currentBlock uint64
}

// OpenWAL opens or initializes a WAL file, validating all existing frames upon startup
func OpenWAL(path string) (*WAL, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed creating wal directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed opening wal file: %w", err)
	}

	wal := &WAL{
		file: file,
		path: path,
	}

	// Validate existing frames and locate last valid offset
	validOffset, count, err := wal.scanAndValidate()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("wal validation failed at offset %d: %w", validOffset, err)
	}

	wal.lastOffset = validOffset
	wal.currentBlock = count

	// Position write pointer at the end of valid data
	if _, err := file.Seek(validOffset, io.SeekStart); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed seeking to valid wal end: %w", err)
	}

	return wal, nil
}

// scanAndValidate reads from offset 0 to end, checking magic bytes, CRC32, and trailers
func (w *WAL) scanAndValidate() (validOffset int64, blockCount uint64, err error) {
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return 0, 0, err
	}

	var offset int64 = 0
	var count uint64 = 0

	for {
		header := make([]byte, 13) // 4 magic + 1 version + 4 length + 4 crc
		n, readErr := io.ReadFull(w.file, header)
		if readErr == io.EOF || (readErr == io.ErrUnexpectedEOF && n == 0) {
			break
		}
		if readErr != nil {
			return offset, count, fmt.Errorf("%w: header read at offset %d", ErrTruncatedFrame, offset)
		}

		if !bytes.Equal(header[0:4], walMagic) {
			return offset, count, fmt.Errorf("%w: invalid magic at offset %d", ErrCorruptFrame, offset)
		}

		length := binary.BigEndian.Uint32(header[5:9])
		expectedCRC := binary.BigEndian.Uint32(header[9:13])

		payload := make([]byte, length)
		if _, readErr := io.ReadFull(w.file, payload); readErr != nil {
			return offset, count, fmt.Errorf("%w: payload read at offset %d", ErrTruncatedFrame, offset)
		}

		calculatedCRC := crc32.ChecksumIEEE(payload)
		if calculatedCRC != expectedCRC {
			return offset, count, fmt.Errorf("%w: calculated %d, expected %d at offset %d", ErrChecksumFailed, calculatedCRC, expectedCRC, offset)
		}

		trailer := make([]byte, 2)
		if _, readErr := io.ReadFull(w.file, trailer); readErr != nil {
			return offset, count, fmt.Errorf("%w: trailer read at offset %d", ErrTruncatedFrame, offset)
		}

		if !bytes.Equal(trailer, walTrailer) {
			return offset, count, fmt.Errorf("%w: invalid trailer at offset %d", ErrCorruptFrame, offset)
		}

		frameSize := int64(13 + length + 2)
		offset += frameSize
		count++
	}

	return offset, count, nil
}

// AppendBlock serializes the block, frames it with checksum and boundary markers, and fsyncs to disk
func (w *WAL) AppendBlock(block *Block) (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	payload, err := json.Marshal(block)
	if err != nil {
		return 0, fmt.Errorf("failed marshaling block %d: %w", block.Header.Index, err)
	}

	length := uint32(len(payload))
	checksum := crc32.ChecksumIEEE(payload)

	var frame bytes.Buffer
	frame.Grow(13 + len(payload) + 2)

	// Header: Magic (4) + Version (1) + Length (4) + CRC (4)
	frame.Write(walMagic)
	frame.WriteByte(walVersion)

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, length)
	frame.Write(lenBuf)

	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, checksum)
	frame.Write(crcBuf)

	// Payload
	frame.Write(payload)

	// Trailer
	frame.Write(walTrailer)

	writeOffset := w.lastOffset
	if _, err := w.file.WriteAt(frame.Bytes(), writeOffset); err != nil {
		return 0, fmt.Errorf("failed writing block frame to wal: %w", err)
	}

	// Guarantee immediate durability
	if err := w.file.Sync(); err != nil {
		return 0, fmt.Errorf("failed to fsync wal file: %w", err)
	}

	w.lastOffset = writeOffset + int64(frame.Len())
	w.currentBlock++

	return writeOffset, nil
}

// ReadAllBlocks reads and parses all blocks sequentially from the WAL
func (w *WAL) ReadAllBlocks() ([]*Block, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed seeking to wal start: %w", err)
	}

	var blocks []*Block
	var offset int64 = 0

	for {
		header := make([]byte, 13)
		n, err := io.ReadFull(w.file, header)
		if err == io.EOF || (err == io.ErrUnexpectedEOF && n == 0) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w at offset %d", ErrTruncatedFrame, offset)
		}

		if !bytes.Equal(header[0:4], walMagic) {
			return nil, fmt.Errorf("%w at offset %d", ErrCorruptFrame, offset)
		}

		length := binary.BigEndian.Uint32(header[5:9])
		expectedCRC := binary.BigEndian.Uint32(header[9:13])

		payload := make([]byte, length)
		if _, err := io.ReadFull(w.file, payload); err != nil {
			return nil, fmt.Errorf("%w payload at offset %d", ErrTruncatedFrame, offset)
		}

		if crc32.ChecksumIEEE(payload) != expectedCRC {
			return nil, fmt.Errorf("%w at offset %d", ErrChecksumFailed, offset)
		}

		trailer := make([]byte, 2)
		if _, err := io.ReadFull(w.file, trailer); err != nil {
			return nil, fmt.Errorf("%w trailer at offset %d", ErrTruncatedFrame, offset)
		}

		if !bytes.Equal(trailer, walTrailer) {
			return nil, fmt.Errorf("%w at offset %d", ErrCorruptFrame, offset)
		}

		var b Block
		if err := json.Unmarshal(payload, &b); err != nil {
			return nil, fmt.Errorf("failed unmarshaling block at offset %d: %w", offset, err)
		}

		blocks = append(blocks, &b)
		offset += int64(13 + length + 2)
	}

	return blocks, nil
}

// BlockCount returns the number of blocks committed to the WAL
func (w *WAL) BlockCount() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentBlock
}

// Close flushes and closes the WAL file
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Sync()
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
