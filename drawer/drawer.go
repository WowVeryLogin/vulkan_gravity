package drawer

import (
	"game/device"
	"game/object"
	"game/pipeline"
	"time"
	"unsafe"

	"github.com/goki/vulkan"
)

type Drawer struct {
	pipeline         *pipeline.Pipeline
	lastRenderedTime time.Time
}

func New(
	device *device.Device,
	renderPass vulkan.RenderPass,
	descriptorsLayout vulkan.DescriptorSetLayout,
) *Drawer {
	return &Drawer{
		pipeline:         pipeline.New(device, renderPass, descriptorsLayout),
		lastRenderedTime: time.Now(),
	}
}

func (d *Drawer) RenderGameObects(
	commandBuffer vulkan.CommandBuffer,
	descriptors vulkan.DescriptorSet,
	gameObjects []*object.GameObject,
) {
	d.pipeline.Bind(commandBuffer, descriptors)

	since := time.Since(d.lastRenderedTime)
	d.lastRenderedTime = time.Now()

	for _, obj := range gameObjects {
		vulkan.CmdPushConstants(
			commandBuffer,
			d.pipeline.Layout,
			vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit|vulkan.ShaderStageFragmentBit),
			0,
			uint32(unsafe.Sizeof(pipeline.PushData{})),
			unsafe.Pointer(obj.ToPushData(since)),
		)

		obj.Model.Bind(commandBuffer)
		obj.Model.Draw(commandBuffer)
	}
}

func (d *Drawer) Close() {
	d.pipeline.Close()
}
