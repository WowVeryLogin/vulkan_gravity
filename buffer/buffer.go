package buffer

import (
	"game/device"
	"unsafe"

	"github.com/goki/vulkan"
)

type Buffer[T any] struct {
	device      *device.Device
	Buffer      vulkan.Buffer
	memory      vulkan.DeviceMemory
	elementSize int
	mapped      bool
	data        unsafe.Pointer
}

func New[T any](
	dev *device.Device,
	initialLen int,
	mapped bool,
	usage vulkan.BufferUsageFlags,
	memoryProps vulkan.MemoryPropertyFlags,
) *Buffer[T] {
	var t T
	size := vulkan.DeviceSize(initialLen * int(unsafe.Sizeof(t)))
	buffer, memory := dev.CreateBuffer(size, usage, memoryProps)

	var data unsafe.Pointer
	if mapped {
		vulkan.MapMemory(dev.LogicalDevice, memory, 0, size, 0, &data)
	}

	return &Buffer[T]{
		device:      dev,
		Buffer:      buffer,
		memory:      memory,
		mapped:      mapped,
		elementSize: int(unsafe.Sizeof(t)),
		data:        data,
	}
}

func (b *Buffer[T]) InitWithStaging(data []T) {
	size := vulkan.DeviceSize(len(data) * b.elementSize)
	device.CopyWithStagingBufferGraphic(b.device, data, func(cb vulkan.CommandBuffer, staging vulkan.Buffer) {
		vulkan.CmdCopyBuffer(cb, staging, b.Buffer, 1, []vulkan.BufferCopy{
			{
				SrcOffset: 0,
				DstOffset: 0,
				Size:      size,
			},
		})
	})
}

func (b *Buffer[T]) WriteBuffer(copyBuffer []T) {
	if !b.mapped {
		panic("Buffer is not mapped, cannot write data")
	}
	slice := unsafe.Slice((*T)(b.data), len(copyBuffer))
	copy(slice, copyBuffer)
}

func (b *Buffer[T]) Close() {
	if b.mapped {
		vulkan.UnmapMemory(b.device.LogicalDevice, b.memory)
		b.data = nil
	}
	vulkan.DestroyBuffer(b.device.LogicalDevice, b.Buffer, nil)
	vulkan.FreeMemory(b.device.LogicalDevice, b.memory, nil)
}
