package drawer

import (
	"game/camera"
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
) *Drawer {
	return &Drawer{
		pipeline:         pipeline.New(device, renderPass),
		lastRenderedTime: time.Now(),
	}
}

func (d *Drawer) RenderGameObects(
	commandBuffer vulkan.CommandBuffer,
	gameObjects []*object.GameObject,
	camera *camera.Camera,
) {
	d.pipeline.Bind(commandBuffer)

	since := time.Since(d.lastRenderedTime)
	d.lastRenderedTime = time.Now()

	view, projection := camera.ToMatrix()

	for _, obj := range gameObjects {
		data := obj.ToPushData(since)
		data.View = view
		data.Projection = projection

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
