package renderer

import (
	"time"
	"unsafe"

	"github.com/WowVeryLogin/vulkan_engine/src/object"
	"github.com/WowVeryLogin/vulkan_engine/src/object/model"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/descriptors"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/pipeline"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/shader"

	"github.com/goki/vulkan"
)

type Renderer struct {
	pipeline         vulkan.Pipeline
	lastRenderedTime time.Time

	device     *device.Device
	vertShader vulkan.ShaderModule
	fragShader vulkan.ShaderModule
	layout     vulkan.PipelineLayout
}

func New(
	device *device.Device,
	renderPass vulkan.RenderPass,
	descriptorsLayout []vulkan.DescriptorSetLayout,
) *Renderer {
	vertModule := shader.CreateShaderModule("shaders/vert.spv", device.LogicalDevice)
	fragModule := shader.CreateShaderModule("shaders/frag.spv", device.LogicalDevice)

	layout := pipeline.NewLayout(device, descriptorsLayout, []vulkan.PushConstantRange{
		{
			StageFlags: vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit | vulkan.ShaderStageFragmentBit),
			Offset:     0,
			Size:       uint32(unsafe.Sizeof(pipeline.PushData{})),
		},
	})
	pipelines := pipeline.New(device, []pipeline.PipelineConfig{
		{
			Layout:     layout,
			RenderPass: renderPass,
			FragShader: fragModule,
			VertShader: vertModule,
		},
	}, model.VertexBindingDescription, model.VertexAttributeDescription)

	return &Renderer{
		pipeline:         pipelines[0],
		lastRenderedTime: time.Now(),
		layout:           layout,
		vertShader:       vertModule,
		fragShader:       fragModule,
		device:           device,
	}
}

func (d *Renderer) Render(
	commandBuffer vulkan.CommandBuffer,
	gameObjects []*object.GameObject,
	frameDescriptorSet *descriptors.DescriptorSet,
) {
	vulkan.CmdBindPipeline(commandBuffer, vulkan.PipelineBindPointGraphics, d.pipeline)

	since := time.Since(d.lastRenderedTime)
	d.lastRenderedTime = time.Now()

	vulkan.CmdBindDescriptorSets(
		commandBuffer,
		vulkan.PipelineBindPointGraphics,
		d.layout,
		0,
		1,
		[]vulkan.DescriptorSet{
			frameDescriptorSet.Set,
		},
		0,
		nil,
	)

	for _, obj := range gameObjects {
		data := obj.ToPushData(since)

		vulkan.CmdPushConstants(
			commandBuffer,
			d.layout,
			vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit|vulkan.ShaderStageFragmentBit),
			0,
			uint32(unsafe.Sizeof(pipeline.PushData{})),
			unsafe.Pointer(data),
		)

		obj.Model.Bind(commandBuffer)
		obj.Model.Draw(commandBuffer)
	}
}

func (d *Renderer) Close() {
	vulkan.DestroyShaderModule(d.device.LogicalDevice, d.vertShader, nil)
	vulkan.DestroyShaderModule(d.device.LogicalDevice, d.fragShader, nil)
	vulkan.DestroyPipeline(d.device.LogicalDevice, d.pipeline, nil)
	vulkan.DestroyPipelineLayout(d.device.LogicalDevice, d.layout, nil)
}
