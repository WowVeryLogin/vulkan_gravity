package model

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"unsafe"

	_ "image/png"

	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/texture"
	"github.com/goki/vulkan"
	"github.com/qmuntal/gltf"
)

type Model struct {
	vertexCount  uint32
	device       *device.Device
	vertexBuffer vulkan.Buffer
	vertexMemory vulkan.DeviceMemory

	hasIndexes  bool
	numIndexes  uint32
	indexBuffer vulkan.Buffer
	indexMemory vulkan.DeviceMemory
	Texture     *texture.Texture
}

type Position struct {
	X float32
	Y float32
	Z float32
}

type Vertex struct {
	Pos    Position
	RGB    [3]float32
	Normal [3]float32
	UV     [2]float32
}

var VertexBindingDescription = []vulkan.VertexInputBindingDescription{
	{
		Binding:   0,
		Stride:    uint32(unsafe.Sizeof(Vertex{})),
		InputRate: vulkan.VertexInputRateVertex,
	},
}

var VertexAttributeDescription = []vulkan.VertexInputAttributeDescription{
	{
		Binding:  0,
		Location: 0,
		Format:   vulkan.FormatR32g32b32Sfloat,
		Offset:   uint32(unsafe.Offsetof(Vertex{}.Pos)),
	},
	{
		Binding:  0,
		Location: 1,
		Format:   vulkan.FormatR32g32b32Sfloat,
		Offset:   uint32(unsafe.Offsetof(Vertex{}.RGB)),
	},
	{
		Binding:  0,
		Location: 2,
		Format:   vulkan.FormatR32g32b32Sfloat,
		Offset:   uint32(unsafe.Offsetof(Vertex{}.Normal)),
	},
	{
		Binding:  0,
		Location: 3,
		Format:   vulkan.FormatR32g32Sfloat,
		Offset:   uint32(unsafe.Offsetof(Vertex{}.UV)),
	},
}

func modelFromGLTF(gltfFile string) ([]Vertex, []uint32, []*texture.TextureConfig) {
	doc, err := gltf.Open(gltfFile)
	if err != nil {
		log.Fatal(err)
	}

	vertices := []Vertex{}
	indexes := []uint32{}

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

			var uvData []byte
			var uvStride int
			uvAccessorIndex, hasUV := primitive.Attributes["TEXCOORD_0"]
			if hasUV {
				uvAccessor := doc.Accessors[uvAccessorIndex]
				uvBufferView := doc.BufferViews[*uvAccessor.BufferView]
				uvBuffer := doc.Buffers[uvBufferView.Buffer]

				uvData = uvBuffer.Data[uvBufferView.ByteOffset : uvBufferView.ByteOffset+uvBufferView.ByteLength]
				uvStride = uvBufferView.ByteStride
				if uvStride == 0 {
					uvStride = 8 // 2 * float32 for UV coords
				}
			}

			primitiveVertices := []Vertex{}
			for i := 0; i < vertexCount; i++ {
				offset := i*byteStride + int(posAccessor.ByteOffset)
				x := binary.LittleEndian.Uint32(posData[offset+0:])
				y := binary.LittleEndian.Uint32(posData[offset+4:])
				z := binary.LittleEndian.Uint32(posData[offset+8:])

				v := Vertex{
					Pos: Position{
						X: math.Float32frombits(x),
						Y: math.Float32frombits(y),
						Z: math.Float32frombits(z),
					},
					RGB:    [3]float32{1, 1, 1}, // Default color
					Normal: [3]float32{0, 0, 0}, // Default normal
					UV:     [2]float32{0, 0},    // Default UV
				}
				if hasUV {
					uvOffset := i*uvStride + int(doc.Accessors[primitive.Attributes["TEXCOORD_0"]].ByteOffset)
					ubits0 := binary.LittleEndian.Uint32(uvData[uvOffset : uvOffset+4])
					ubits1 := binary.LittleEndian.Uint32(uvData[uvOffset+4 : uvOffset+8])

					u := math.Float32frombits(ubits0)
					vv := math.Float32frombits(ubits1)
					v.UV = [2]float32{u, vv}
				}

				primitiveVertices = append(primitiveVertices, v)
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
					primitiveVertices[i].Normal = [3]float32{
						math.Float32frombits(binary.LittleEndian.Uint32(normData[offset+0:])),
						math.Float32frombits(binary.LittleEndian.Uint32(normData[offset+4:])),
						math.Float32frombits(binary.LittleEndian.Uint32(normData[offset+8:])),
					}
				}
			}
			vertices = append(vertices, primitiveVertices...)

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

	var textures []*texture.TextureConfig
	for i, tex := range doc.Textures {
		if tex.Source == nil || int(*tex.Source) >= len(doc.Images) {
			fmt.Printf("Texture %d has no valid source image\n", i)
			continue
		}

		img := doc.Images[*tex.Source]
		fmt.Printf("Texture %d: MIME Type = %s\n", i, img.MimeType)

		// Get image data
		textureRaw, err := extractImageData(doc, img)
		if err != nil {
			fmt.Printf("  Failed to extract image data: %v\n", err)
			continue
		}

		textures = append(textures, texture.TextureConfigFromPNG(bytes.NewReader(textureRaw)))
	}

	return vertices, indexes, textures
}

func extractImageData(doc *gltf.Document, img *gltf.Image) ([]byte, error) {
	if img.URI != "" {
		// Data URI (base64-encoded) or external file
		if strings.HasPrefix(img.URI, "data:") {
			// Parse data URI
			parts := strings.SplitN(img.URI, ",", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid data URI")
			}
			return base64.StdEncoding.DecodeString(parts[1])
		}

		// External image file — load separately if needed
		return os.ReadFile(img.URI)
	}

	// Image is stored in a bufferView
	if img.BufferView == nil || int(*img.BufferView) >= len(doc.BufferViews) {
		return nil, fmt.Errorf("no valid BufferView")
	}

	bv := doc.BufferViews[*img.BufferView]

	offset := int(bv.ByteOffset)
	length := int(bv.ByteLength)

	// For .glb, buffer is in the same file
	return doc.Buffers[bv.Buffer].Data[offset : offset+length], nil
}

func NewWithGLTF(device *device.Device, gltfFile string) *Model {
	vertices, indexes, textureData := modelFromGLTF(gltfFile)

	return New(device, vertices, indexes, textureData[0])
}

func New(dev *device.Device, vertices []Vertex, indexes []uint32, textureData *texture.TextureConfig) *Model {
	size := vulkan.DeviceSize(len(vertices) * int(unsafe.Sizeof(Vertex{})))

	buffer, memory := dev.CreateBuffer(
		size,
		vulkan.BufferUsageFlags(vulkan.BufferUsageVertexBufferBit|vulkan.BufferUsageTransferDstBit),
		vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit),
	)

	device.CopyWithStagingBufferGraphic(dev, vertices, func(cb vulkan.CommandBuffer, staging vulkan.Buffer) {
		vulkan.CmdCopyBuffer(cb, staging, buffer, 1, []vulkan.BufferCopy{
			{
				SrcOffset: 0,
				DstOffset: 0,
				Size:      size,
			},
		})
	})

	var (
		indexBuffer vulkan.Buffer
		indexMemory vulkan.DeviceMemory
	)
	if len(indexes) > 0 {
		indexSize := vulkan.DeviceSize(len(indexes) * int(unsafe.Sizeof(uint32(0))))

		indexBuffer, indexMemory = dev.CreateBuffer(
			indexSize,
			vulkan.BufferUsageFlags(vulkan.BufferUsageIndexBufferBit|vulkan.BufferUsageTransferDstBit),
			vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit),
		)

		device.CopyWithStagingBufferGraphic(dev, indexes, func(cb vulkan.CommandBuffer, staging vulkan.Buffer) {
			vulkan.CmdCopyBuffer(cb, staging, indexBuffer, 1, []vulkan.BufferCopy{
				{
					SrcOffset: 0,
					DstOffset: 0,
					Size:      indexSize,
				},
			})
		})
	}

	var tex *texture.Texture
	if textureData != nil {
		tex = texture.New(dev, textureData)
	}

	return &Model{
		vertexCount:  uint32(len(vertices)),
		device:       dev,
		vertexBuffer: buffer,
		vertexMemory: memory,
		hasIndexes:   len(indexes) > 0,
		numIndexes:   uint32(len(indexes)),
		indexBuffer:  indexBuffer,
		indexMemory:  indexMemory,
		Texture:      tex,
	}
}

func (m *Model) Bind(commandBuffer vulkan.CommandBuffer) {
	if m.hasIndexes {
		vulkan.CmdBindIndexBuffer(commandBuffer, m.indexBuffer, 0, vulkan.IndexTypeUint32)
	}
	vulkan.CmdBindVertexBuffers(commandBuffer, 0, 1, []vulkan.Buffer{m.vertexBuffer}, []vulkan.DeviceSize{0})
}

func (m *Model) Draw(commandBuffer vulkan.CommandBuffer) {
	if m.hasIndexes {
		vulkan.CmdDrawIndexed(commandBuffer, m.numIndexes, 1, 0, 0, 0)
		return
	}
	vulkan.CmdDraw(commandBuffer, m.vertexCount, 1, 0, 0)
}

func (m *Model) Close() {
	vulkan.DestroyBuffer(m.device.LogicalDevice, m.vertexBuffer, nil)
	vulkan.FreeMemory(m.device.LogicalDevice, m.vertexMemory, nil)
	if m.hasIndexes {
		vulkan.DestroyBuffer(m.device.LogicalDevice, m.indexBuffer, nil)
		vulkan.FreeMemory(m.device.LogicalDevice, m.indexMemory, nil)
	}
	if m.Texture != nil {
		m.Texture.Close()
	}
}
