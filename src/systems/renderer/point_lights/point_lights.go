package pointlights

import (
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/descriptors"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/pipeline"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/shader"
	"github.com/goki/vulkan"
)

type PointLightsRenderer struct {
	pipeline vulkan.Pipeline

	device     *device.Device
	vertShader vulkan.ShaderModule
	fragShader vulkan.ShaderModule
	layout     vulkan.PipelineLayout
}

func New(
	device *device.Device,
	renderPass vulkan.RenderPass,
	descriptorsLayout []vulkan.DescriptorSetLayout,
) *PointLightsRenderer {
	vertModule := shader.CreateShaderModule("shaders/point_light.vert.spv", device.LogicalDevice)
	fragModule := shader.CreateShaderModule("shaders/point_light.frag.spv", device.LogicalDevice)

	layout := pipeline.NewLayout(device, descriptorsLayout, nil)
	pipelines := pipeline.New(device, []pipeline.PipelineConfig{
		{
			Layout:     layout,
			RenderPass: renderPass,
			FragShader: fragModule,
			VertShader: vertModule,
		},
	}, nil, nil)

	return &PointLightsRenderer{
		pipeline:   pipelines[0],
		layout:     layout,
		vertShader: vertModule,
		fragShader: fragModule,
		device:     device,
	}
}

func (d *PointLightsRenderer) Render(
	commandBuffer vulkan.CommandBuffer,
	frameDescriptorSet *descriptors.DescriptorSet,
) {
	vulkan.CmdBindPipeline(commandBuffer, vulkan.PipelineBindPointGraphics, d.pipeline)
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
	vulkan.CmdDraw(commandBuffer, 6, 1, 0, 0)
}

func (d *PointLightsRenderer) Close() {
	vulkan.DestroyShaderModule(d.device.LogicalDevice, d.vertShader, nil)
	vulkan.DestroyShaderModule(d.device.LogicalDevice, d.fragShader, nil)
	vulkan.DestroyPipeline(d.device.LogicalDevice, d.pipeline, nil)
	vulkan.DestroyPipelineLayout(d.device.LogicalDevice, d.layout, nil)
}
