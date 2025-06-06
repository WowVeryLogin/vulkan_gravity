package pipeline

import (
	"game/device"
	"game/model"
	"game/shader"
	"unsafe"

	"github.com/goki/vulkan"
)

type Pipeline struct {
	device        *device.Device
	shaderModules []vulkan.ShaderModule
	pipeline      vulkan.Pipeline
	Layout        vulkan.PipelineLayout
}

type PushData struct {
	Transformation [16]float32
	Color          [3]float32
	_              float32
}

func New(
	device *device.Device,
	renderPass vulkan.RenderPass,
	descriptorsLayout []vulkan.DescriptorSetLayout,
) *Pipeline {
	vertModule := shader.CreateShaderModule("shaders/vert.spv", device.LogicalDevice)
	fragModule := shader.CreateShaderModule("shaders/frag.spv", device.LogicalDevice)

	var layout vulkan.PipelineLayout
	if err := vulkan.Error(vulkan.CreatePipelineLayout(device.LogicalDevice, &vulkan.PipelineLayoutCreateInfo{
		SType:                  vulkan.StructureTypePipelineLayoutCreateInfo,
		PushConstantRangeCount: 1,
		PPushConstantRanges: []vulkan.PushConstantRange{
			{
				StageFlags: vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit | vulkan.ShaderStageFragmentBit),
				Offset:     0,
				Size:       uint32(unsafe.Sizeof(PushData{})),
			},
		},
		SetLayoutCount: uint32(len(descriptorsLayout)),
		PSetLayouts:    descriptorsLayout,
	}, nil, &layout)); err != nil {
		panic("failed to create pipeline layout: " + err.Error())
	}

	pipeline := make([]vulkan.Pipeline, 1)
	if err := vulkan.Error(vulkan.CreateGraphicsPipelines(device.LogicalDevice, vulkan.NullPipelineCache, 1, []vulkan.GraphicsPipelineCreateInfo{
		{
			SType:      vulkan.StructureTypeGraphicsPipelineCreateInfo,
			StageCount: 2,
			PStages: []vulkan.PipelineShaderStageCreateInfo{
				{
					SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
					Stage:  vulkan.ShaderStageVertexBit,
					Module: vertModule,
					PName:  "main\x00",
				},
				{
					SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
					Stage:  vulkan.ShaderStageFragmentBit,
					Module: fragModule,
					PName:  "main\x00",
				},
			},
			PVertexInputState: &vulkan.PipelineVertexInputStateCreateInfo{
				SType:                           vulkan.StructureTypePipelineVertexInputStateCreateInfo,
				VertexAttributeDescriptionCount: uint32(len(model.VertexAttributeDescription)),
				VertexBindingDescriptionCount:   uint32(len(model.VertexBindingDescription)),
				PVertexBindingDescriptions:      model.VertexBindingDescription,
				PVertexAttributeDescriptions:    model.VertexAttributeDescription,
			},
			PInputAssemblyState: &vulkan.PipelineInputAssemblyStateCreateInfo{
				SType:                  vulkan.StructureTypePipelineInputAssemblyStateCreateInfo,
				Topology:               vulkan.PrimitiveTopologyTriangleList,
				PrimitiveRestartEnable: vulkan.False,
			},
			PViewportState: &vulkan.PipelineViewportStateCreateInfo{
				SType:         vulkan.StructureTypePipelineViewportStateCreateInfo,
				ViewportCount: 1,
				ScissorCount:  1,
			},
			PRasterizationState: &vulkan.PipelineRasterizationStateCreateInfo{
				SType:       vulkan.StructureTypePipelineRasterizationStateCreateInfo,
				PolygonMode: vulkan.PolygonModeFill,
				LineWidth:   1.0,
				CullMode:    vulkan.CullModeFlags(vulkan.CullModeNone),
				FrontFace:   vulkan.FrontFaceClockwise,
			},
			PMultisampleState: &vulkan.PipelineMultisampleStateCreateInfo{
				SType:                vulkan.StructureTypePipelineMultisampleStateCreateInfo,
				RasterizationSamples: vulkan.SampleCount1Bit,
				MinSampleShading:     1.0,
			},
			PColorBlendState: &vulkan.PipelineColorBlendStateCreateInfo{
				SType:           vulkan.StructureTypePipelineColorBlendStateCreateInfo,
				LogicOp:         vulkan.LogicOpCopy,
				AttachmentCount: 1,
				PAttachments: []vulkan.PipelineColorBlendAttachmentState{
					{
						ColorWriteMask:      vulkan.ColorComponentFlags(vulkan.ColorComponentRBit | vulkan.ColorComponentGBit | vulkan.ColorComponentBBit | vulkan.ColorComponentABit),
						SrcColorBlendFactor: vulkan.BlendFactorOne,
						DstColorBlendFactor: vulkan.BlendFactorZero,
						ColorBlendOp:        vulkan.BlendOpAdd,
						SrcAlphaBlendFactor: vulkan.BlendFactorOne,
						DstAlphaBlendFactor: vulkan.BlendFactorZero,
						AlphaBlendOp:        vulkan.BlendOpAdd,
					},
				},
			},
			PDepthStencilState: &vulkan.PipelineDepthStencilStateCreateInfo{
				SType:            vulkan.StructureTypePipelineDepthStencilStateCreateInfo,
				DepthTestEnable:  vulkan.True,
				DepthWriteEnable: vulkan.True,
				DepthCompareOp:   vulkan.CompareOpLess,
				MinDepthBounds:   0,
				MaxDepthBounds:   1,
			},
			PDynamicState: &vulkan.PipelineDynamicStateCreateInfo{
				SType:             vulkan.StructureTypePipelineDynamicStateCreateInfo,
				DynamicStateCount: 2,
				PDynamicStates: []vulkan.DynamicState{
					vulkan.DynamicStateViewport,
					vulkan.DynamicStateScissor,
				},
			},
			Layout:             layout,
			RenderPass:         renderPass,
			Subpass:            0,
			BasePipelineIndex:  -1,
			BasePipelineHandle: vulkan.NullPipeline,
		},
	}, nil, pipeline)); err != nil {
		panic("failed to create pipeline: " + err.Error())
	}

	return &Pipeline{
		device:        device,
		pipeline:      pipeline[0],
		Layout:        layout,
		shaderModules: []vulkan.ShaderModule{vertModule, fragModule},
	}
}

func (p *Pipeline) Bind(commandBuffer vulkan.CommandBuffer) {
	vulkan.CmdBindPipeline(commandBuffer, vulkan.PipelineBindPointGraphics, p.pipeline)
}

func (p *Pipeline) Close() {
	for _, module := range p.shaderModules {
		vulkan.DestroyShaderModule(p.device.LogicalDevice, module, nil)
	}
	vulkan.DestroyPipelineLayout(p.device.LogicalDevice, p.Layout, nil)
	vulkan.DestroyPipeline(p.device.LogicalDevice, p.pipeline, nil)
}
