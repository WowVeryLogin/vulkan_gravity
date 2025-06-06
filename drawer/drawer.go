package drawer

import (
	"game/descriptors"
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
	descriptorsLayout []vulkan.DescriptorSetLayout,
) *Drawer {
	return &Drawer{
		pipeline:         pipeline.New(device, renderPass, descriptorsLayout),
		lastRenderedTime: time.Now(),
	}
}

func (d *Drawer) RenderGameObects(
	commandBuffer vulkan.CommandBuffer,
	gameObjects []*object.GameObject,
	worldDescriptorSet *descriptors.DescriptorsSet,
	frameDescriptorSet *descriptors.DescriptorsSet,
) {
	d.pipeline.Bind(commandBuffer)

	since := time.Since(d.lastRenderedTime)
	d.lastRenderedTime = time.Now()

	vulkan.CmdBindDescriptorSets(
		commandBuffer,
		vulkan.PipelineBindPointGraphics,
		d.pipeline.Layout,
		0,
		2,
		[]vulkan.DescriptorSet{
			worldDescriptorSet.Set,
			frameDescriptorSet.Set,
		},
		0,
		nil,
	)

	for _, obj := range gameObjects {
		data := obj.ToPushData(since)

		vulkan.CmdPushConstants(
			commandBuffer,
			d.pipeline.Layout,
			vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit|vulkan.ShaderStageFragmentBit),
			0,
			uint32(unsafe.Sizeof(pipeline.PushData{})),
			unsafe.Pointer(data),
		)

		obj.Model.Bind(commandBuffer)
		obj.Model.Draw(commandBuffer)
	}
}

func (d *Drawer) Close() {
	d.pipeline.Close()
}
