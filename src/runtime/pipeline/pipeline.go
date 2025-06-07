package pipeline

import (
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/goki/vulkan"
)

type PushData struct {
	Transformation [16]float32
	TextureType    int32
	_              [15]float32
}

func NewLayout(
	device *device.Device,
	descriptorsLayout []vulkan.DescriptorSetLayout,
	constRanges []vulkan.PushConstantRange,
) vulkan.PipelineLayout {
	var layout vulkan.PipelineLayout
	if err := vulkan.Error(vulkan.CreatePipelineLayout(device.LogicalDevice, &vulkan.PipelineLayoutCreateInfo{
		SType:                  vulkan.StructureTypePipelineLayoutCreateInfo,
		PushConstantRangeCount: uint32(len(constRanges)),
		PPushConstantRanges:    constRanges,
		SetLayoutCount:         uint32(len(descriptorsLayout)),
		PSetLayouts:            descriptorsLayout,
	}, nil, &layout)); err != nil {
		panic("failed to create pipeline layout: " + err.Error())
	}

	return layout
}

type PipelineConfig struct {
	Layout     vulkan.PipelineLayout
	RenderPass vulkan.RenderPass
	VertShader vulkan.ShaderModule
	FragShader vulkan.ShaderModule
}

func New(
	device *device.Device,
	configs []PipelineConfig,
	vertexBindingDesc []vulkan.VertexInputBindingDescription,
	vertexBindingAttr []vulkan.VertexInputAttributeDescription,
) []vulkan.Pipeline {
	infos := []vulkan.GraphicsPipelineCreateInfo{}
	for _, config := range configs {
		infos = append(infos, vulkan.GraphicsPipelineCreateInfo{
			SType:      vulkan.StructureTypeGraphicsPipelineCreateInfo,
			StageCount: 2,
			PStages: []vulkan.PipelineShaderStageCreateInfo{
				{
					SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
					Stage:  vulkan.ShaderStageVertexBit,
					Module: config.VertShader,
					PName:  "main\x00",
				},
				{
					SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
					Stage:  vulkan.ShaderStageFragmentBit,
					Module: config.FragShader,
					PName:  "main\x00",
				},
			},
			PVertexInputState: &vulkan.PipelineVertexInputStateCreateInfo{
				SType:                           vulkan.StructureTypePipelineVertexInputStateCreateInfo,
				VertexBindingDescriptionCount:   uint32(len(vertexBindingDesc)),
				PVertexBindingDescriptions:      vertexBindingDesc,
				VertexAttributeDescriptionCount: uint32(len(vertexBindingAttr)),
				PVertexAttributeDescriptions:    vertexBindingAttr,
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
			Layout:             config.Layout,
			RenderPass:         config.RenderPass,
			Subpass:            0,
			BasePipelineIndex:  -1,
			BasePipelineHandle: vulkan.NullPipeline,
		})
	}

	pipelines := make([]vulkan.Pipeline, len(configs))
	if err := vulkan.Error(vulkan.CreateGraphicsPipelines(device.LogicalDevice, vulkan.NullPipelineCache, uint32(len(configs)),
		infos, nil, pipelines)); err != nil {
		panic("failed to create pipeline: " + err.Error())
	}

	return pipelines
}
