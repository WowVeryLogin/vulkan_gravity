package model

import (
	"fmt"
	"game/device"
	"game/model/builder"
	"unsafe"

	"github.com/goki/vulkan"
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
	UV     [3]float32
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

func NewWithGLTF(device *device.Device, gltfFile string) *Model {
	modelPoses, indexes, normals := builder.NewFromGLTF(gltfFile)
	vertices := make([]Vertex, 0, (len(modelPoses)+2)/3)
	for i := 0; i < len(modelPoses); i += 3 {
		vertices = append(vertices, Vertex{
			Pos: Position{
				X: modelPoses[i],
				Y: modelPoses[i+1],
				Z: modelPoses[i+2],
			},
			RGB: [3]float32{1, 1, 1}, // Default color, can be modified later
			Normal: [3]float32{
				normals[i],
				normals[i+1],
				normals[i+2],
			},
		})
	}
	fmt.Println("Model vertices count:", len(vertices), "indexes count:", len(indexes))

	return New(device, vertices, indexes)
}

func New(dev *device.Device, vertices []Vertex, indexes []uint32) *Model {
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

	return &Model{
		vertexCount:  uint32(len(vertices)),
		device:       dev,
		vertexBuffer: buffer,
		vertexMemory: memory,
		hasIndexes:   len(indexes) > 0,
		numIndexes:   uint32(len(indexes)),
		indexBuffer:  indexBuffer,
		indexMemory:  indexMemory,
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
}
