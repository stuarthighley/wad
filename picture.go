package wad

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type binPatchImageHeader struct {
	Width, Height, LeftOffset, TopOffset int16
}

// Read a picture lump
func (w *WAD) ReadPicture(name string) (*Picture, error) {
	lumpNum, ok := w.lumpNums[name]
	if !ok {
		return nil, fmt.Errorf("%v lump not found", name)
	}

	lumpInfo := w.lumpInfos[lumpNum]
	if err := w.seek(int64(lumpInfo.Filepos)); err != nil {
		return nil, err
	}

	// Read lump
	lump := make([]byte, lumpInfo.Size)
	n, err := w.file.Read(lump)
	if err != nil {
		return nil, err
	}
	if n != lumpInfo.Size {
		return nil, fmt.Errorf("truncated lump")
	}

	// Read patch lump header
	reader := bytes.NewBuffer(lump)
	var header binPatchImageHeader
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// Initialise rectangular picture space to transparent
	columns := make([]Column, header.Width)
	for i := range columns {
		columns[i] = make(Column, header.Height)
		for j := range columns[i] {
			columns[i][j] = w.TransparentIndex
		}
	}

	// Read column offsets
	offsets := make([]int32, header.Width)
	if err := binary.Read(reader, binary.LittleEndian, offsets); err != nil {
		return nil, err
	}

	// For each column offset, expand out the posts into columns
	for columnIndex, offset := range offsets {
		for {
			topDelta := int(lump[offset])
			offset += 1
			if topDelta == 255 {
				break
			}
			numPixels := int(lump[offset])
			offset += 1
			offset += 1 // Padding
			for i := range numPixels {
				columns[columnIndex][topDelta+i] = lump[offset]
				offset += 1
			}
			offset += 1 // Padding
		}
	}
	return &Picture{Width: int(header.Width), Height: int(header.Height), Columns: columns}, nil
}
