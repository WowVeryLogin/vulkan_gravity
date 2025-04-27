package model

import (
	"game/device"
	"unsafe"

	"github.com/goki/vulkan"
)

type Model struct {
	vertexCount uint32
	device      *device.Device
	buffer      vulkan.Buffer
	memory      vulkan.DeviceMemory
}

type Position struct {
	X float32
	Y float32
}

type Vertex struct {
	Pos Position
	RGB [3]float32
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
		Format:   vulkan.FormatR32g32Sfloat,
		Offset:   uint32(unsafe.Offsetof(Vertex{}.Pos)),
	},
	{
		Binding:  0,
		Location: 1,
		Format:   vulkan.FormatR32g32b32Sfloat,
		Offset:   uint32(unsafe.Offsetof(Vertex{}.RGB)),
	},
}

func New(device *device.Device, vertices []Vertex) *Model {
	size := vulkan.DeviceSize(len(vertices) * int(unsafe.Sizeof(Vertex{})))

	buffer, memory := device.CreateBuffer(
		size,
		vulkan.BufferUsageFlags(vulkan.BufferUsageVertexBufferBit),
		vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit),
	)

	var data unsafe.Pointer
	if err := vulkan.Error(vulkan.MapMemory(device.LogicalDevice, memory, 0, size, 0, &data)); err != nil {
		panic("failed to map buffer memory: " + err.Error())
	}
	slice := unsafe.Slice((*Vertex)(data), len(vertices))
	copy(slice, vertices)
	vulkan.UnmapMemory(device.LogicalDevice, memory)

	return &Model{
		vertexCount: uint32(len(vertices)),
		device:      device,
		buffer:      buffer,
		memory:      memory,
	}
}

func (m *Model) Bind(commandBuffer vulkan.CommandBuffer) {
	vulkan.CmdBindVertexBuffers(commandBuffer, 0, 1, []vulkan.Buffer{m.buffer}, []vulkan.DeviceSize{0})
}

func (m *Model) Draw(commandBuffer vulkan.CommandBuffer) {
	vulkan.CmdDraw(commandBuffer, m.vertexCount, 1, 0, 0)
}

func (m *Model) Close() {
	vulkan.DestroyBuffer(m.device.LogicalDevice, m.buffer, nil)
	vulkan.FreeMemory(m.device.LogicalDevice, m.memory, nil)
}
