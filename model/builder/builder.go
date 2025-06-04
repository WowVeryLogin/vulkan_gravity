package builder

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/qmuntal/gltf"
)

func NewFromGLTF(gltfFile string) ([]float32, []uint32, []float32) {
	doc, err := gltf.Open(gltfFile)
	if err != nil {
		log.Fatal(err)
	}

	vertices := []float32{}
	indexes := []uint32{}
	normals := []float32{}
	// Load buffer data
	for _, mesh := range doc.Meshes {
		for _, primitive := range mesh.Primitives {
			// Get accessor for POSITION
			posAccessor := doc.Accessors[primitive.Attributes["POSITION"]]
			posBufferView := doc.BufferViews[*posAccessor.BufferView]
			positionBuffer := doc.Buffers[posBufferView.Buffer]

			// Get actual vertex data
			posData := positionBuffer.Data[posBufferView.ByteOffset : posBufferView.ByteOffset+posBufferView.ByteLength]
			vertexCount := int(posAccessor.Count)
			byteStride := posBufferView.ByteStride
			if byteStride == 0 {
				byteStride = 12 // 3 * 4 bytes (float32)
			}

			for i := 0; i < vertexCount; i++ {
				offset := i*byteStride + int(posAccessor.ByteOffset)
				x := binary.LittleEndian.Uint32(posData[offset+0:])
				y := binary.LittleEndian.Uint32(posData[offset+4:])
				z := binary.LittleEndian.Uint32(posData[offset+8:])
				vertices = append(vertices, math.Float32frombits(x), math.Float32frombits(y), math.Float32frombits(z))
			}

			normAccessorIdx, ok := primitive.Attributes["NORMAL"]
			if ok {
				normAccessor := doc.Accessors[normAccessorIdx]
				normView := doc.BufferViews[*normAccessor.BufferView]
				normBuffer := doc.Buffers[normView.Buffer].Data
				normStride := normView.ByteStride
				if normStride == 0 {
					normStride = 12
				}
				normData := normBuffer[normView.ByteOffset : normView.ByteOffset+normView.ByteLength]
				for i := 0; i < int(normAccessor.Count); i++ {
					offset := i*normStride + int(normAccessor.ByteOffset)
					x := binary.LittleEndian.Uint32(normData[offset+0:])
					y := binary.LittleEndian.Uint32(normData[offset+4:])
					z := binary.LittleEndian.Uint32(normData[offset+8:])
					normals = append(normals, math.Float32frombits(x), math.Float32frombits(y), math.Float32frombits(z))
				}
			}

			// Get indices (if available)
			if primitive.Indices != nil {
				idxAccessor := doc.Accessors[*primitive.Indices]
				idxBufferView := doc.BufferViews[*idxAccessor.BufferView]
				idxBuffer := doc.Buffers[idxBufferView.Buffer]
				idxData := idxBuffer.Data[idxBufferView.ByteOffset : idxBufferView.ByteOffset+idxBufferView.ByteLength]

				for i := 0; i < int(idxAccessor.Count); i++ {
					var index uint32
					offset := i * 2 // assuming unsigned short (uint16)
					if idxAccessor.ComponentType == gltf.ComponentUshort {
						index = uint32(binary.LittleEndian.Uint16(idxData[offset:]))
					} else if idxAccessor.ComponentType == gltf.ComponentUint {
						offset = i * 4
						index = binary.LittleEndian.Uint32(idxData[offset:])
					}
					indexes = append(indexes, index)
				}
			}
		}
	}

	return vertices, indexes, normals
}
