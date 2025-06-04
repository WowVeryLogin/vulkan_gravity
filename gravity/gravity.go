package gravity

import (
	"game/device"
	"game/object"
	"game/shader"
	"game/swapchain"
	"time"
	"unsafe"

	"github.com/goki/vulkan"
)

type ObjectWithMass struct {
	position [3]float32
	velocity [3]float32
	mass     float32
	_        [3]float32 // padding to make struct 24 bytes (multiple of 8)
}

type ForceField struct {
	force [3]float32
	_     [1]float32
}

type VectorField struct {
	position [3]float32
	_        [1]float32
}

type pushMassData struct {
	timeSince   float32
	numElements uint32
	_           [2]float32
}

type pushFieldData struct {
	totalElements uint32
	numElements   uint32
	_             [2]float32
}

type Buffers struct {
	massBuffer  vulkan.Buffer
	massMemory  vulkan.DeviceMemory
	forceBuffer vulkan.Buffer
	forceMemory vulkan.DeviceMemory
}

type Gravity struct {
	device                     *device.Device
	pipelines                  []vulkan.Pipeline
	pipelinesLayout            vulkan.PipelineLayout
	DescriptorsSets            []vulkan.DescriptorSet
	DescriptorsLayout          vulkan.DescriptorSetLayout
	lastRenderedTime           time.Time
	buffers                    []Buffers
	vecBuffer                  vulkan.Buffer
	vecMemory                  vulkan.DeviceMemory
	massModule                 vulkan.ShaderModule
	fieldModule                vulkan.ShaderModule
	descriptorsPool            vulkan.DescriptorPool
	computeFinished            []vulkan.Semaphore
	computeFinishedForGraphics []vulkan.Semaphore

	massElementsCount  int
	fieldElementsCount int
}

func createSyncObjects(
	device *device.Device,
) ([]vulkan.Semaphore, []vulkan.Semaphore) {
	computeFinished := make([]vulkan.Semaphore, swapchain.MAX_FRAMES_IN_FLIGHT)
	computeFinishedForGraphics := make([]vulkan.Semaphore, swapchain.MAX_FRAMES_IN_FLIGHT)

	for i := range computeFinished {
		if err := vulkan.Error(vulkan.CreateSemaphore(device.LogicalDevice, &vulkan.SemaphoreCreateInfo{
			SType: vulkan.StructureTypeSemaphoreCreateInfo,
		}, nil, &computeFinished[i])); err != nil {
			panic("failed to create image semaphore: " + err.Error())
		}

		if err := vulkan.Error(vulkan.CreateSemaphore(device.LogicalDevice, &vulkan.SemaphoreCreateInfo{
			SType: vulkan.StructureTypeSemaphoreCreateInfo,
		}, nil, &computeFinishedForGraphics[i])); err != nil {
			panic("failed to create image semaphore: " + err.Error())
		}
	}

	return computeFinished, computeFinishedForGraphics
}

func New(device *device.Device) *Gravity {
	massModule := shader.CreateShaderModule("shaders/gravity.comp.spv", device.LogicalDevice)
	forceModule := shader.CreateShaderModule("shaders/field.comp.spv", device.LogicalDevice)

	var descriptorsLayout vulkan.DescriptorSetLayout
	if err := vulkan.Error(vulkan.CreateDescriptorSetLayout(device.LogicalDevice, &vulkan.DescriptorSetLayoutCreateInfo{
		SType:        vulkan.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 4,
		PBindings: []vulkan.DescriptorSetLayoutBinding{
			{ // mass previous frame (in)
				Binding:         0,
				DescriptorCount: 1,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageComputeBit),
			},
			{ // mass current frame (out)
				Binding:         1,
				DescriptorCount: 1,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit | vulkan.ShaderStageComputeBit),
			},
			{ // vector field (in)
				Binding:         2,
				DescriptorCount: 1,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageComputeBit),
			},
			{ // force field (out)
				Binding:         3,
				DescriptorCount: 1,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit | vulkan.ShaderStageComputeBit),
			},
		},
	}, nil, &descriptorsLayout)); err != nil {
		panic("failed to create descriptor set layout: " + err.Error())
	}

	var descriptorsPool vulkan.DescriptorPool
	if err := vulkan.Error(vulkan.CreateDescriptorPool(device.LogicalDevice, &vulkan.DescriptorPoolCreateInfo{
		SType:         vulkan.StructureTypeDescriptorPoolCreateInfo,
		PoolSizeCount: 1,
		PPoolSizes: []vulkan.DescriptorPoolSize{
			{
				Type:            vulkan.DescriptorTypeStorageBuffer,
				DescriptorCount: 4 * swapchain.MAX_FRAMES_IN_FLIGHT,
			},
		},
		MaxSets: swapchain.MAX_FRAMES_IN_FLIGHT,
	}, nil, &descriptorsPool)); err != nil {
		panic("failed to create descriptor pool: " + err.Error())
	}

	descriptorsSets := make([]vulkan.DescriptorSet, swapchain.MAX_FRAMES_IN_FLIGHT)
	for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
		if err := vulkan.Error(vulkan.AllocateDescriptorSets(device.LogicalDevice, &vulkan.DescriptorSetAllocateInfo{
			SType:              vulkan.StructureTypeDescriptorSetAllocateInfo,
			DescriptorPool:     descriptorsPool,
			DescriptorSetCount: 1,
			PSetLayouts: []vulkan.DescriptorSetLayout{
				descriptorsLayout,
			},
		}, &descriptorsSets[i])); err != nil {
			panic("failed to allocate descriptor sets: " + err.Error())
		}
	}

	var layout vulkan.PipelineLayout
	if err := vulkan.Error(vulkan.CreatePipelineLayout(device.LogicalDevice, &vulkan.PipelineLayoutCreateInfo{
		SType:                  vulkan.StructureTypePipelineLayoutCreateInfo,
		PushConstantRangeCount: 1,
		PPushConstantRanges: []vulkan.PushConstantRange{
			{
				StageFlags: vulkan.ShaderStageFlags(vulkan.ShaderStageComputeBit),
				Offset:     0,
				Size:       uint32(unsafe.Sizeof(pushFieldData{})),
			},
		},
		SetLayoutCount: 1,
		PSetLayouts: []vulkan.DescriptorSetLayout{
			descriptorsLayout,
		},
	}, nil, &layout)); err != nil {
		panic("failed to create pipeline layout: " + err.Error())
	}

	pipelines := make([]vulkan.Pipeline, 2)
	if err := vulkan.Error(vulkan.CreateComputePipelines(device.LogicalDevice, nil, 2, []vulkan.ComputePipelineCreateInfo{
		{
			SType: vulkan.StructureTypeComputePipelineCreateInfo,
			Stage: vulkan.PipelineShaderStageCreateInfo{
				SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
				Stage:  vulkan.ShaderStageComputeBit,
				Module: massModule,
				PName:  "main\x00",
			},
			Layout: layout,
		},
		{
			SType: vulkan.StructureTypeComputePipelineCreateInfo,
			Stage: vulkan.PipelineShaderStageCreateInfo{
				SType:  vulkan.StructureTypePipelineShaderStageCreateInfo,
				Stage:  vulkan.ShaderStageComputeBit,
				Module: forceModule,
				PName:  "main\x00",
			},
			Layout: layout,
		},
	}, nil, pipelines)); err != nil {
		panic("failed to create compute pipeline: " + err.Error())
	}

	computeFinished, computeFinishedForGraphics := createSyncObjects(device)
	if err := vulkan.Error(vulkan.QueueSubmit(device.ComputeQueue, 1, []vulkan.SubmitInfo{
		{
			SType:                vulkan.StructureTypeSubmitInfo,
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vulkan.Semaphore{
				computeFinished[swapchain.MAX_FRAMES_IN_FLIGHT-1],
			},
		},
	}, nil)); err != nil {
		panic("failed to submit compute command buffer: " + err.Error())
	}

	return &Gravity{
		pipelines:         pipelines,
		pipelinesLayout:   layout,
		DescriptorsSets:   descriptorsSets,
		lastRenderedTime:  time.Now(),
		DescriptorsLayout: descriptorsLayout,

		buffers: make([]Buffers, swapchain.MAX_FRAMES_IN_FLIGHT),
		device:  device,

		massModule:                 massModule,
		fieldModule:                forceModule,
		computeFinished:            computeFinished,
		computeFinishedForGraphics: computeFinishedForGraphics,

		descriptorsPool: descriptorsPool,
	}
}

func (g *Gravity) UploadMassObjects(
	dev *device.Device,
	objects []*object.GameObject,
) {
	var massObjects []ObjectWithMass
	for _, object := range objects {
		if object.Mass != nil {
			massObjects = append(massObjects, ObjectWithMass{
				position: object.GetPosition(),
				velocity: object.Mass.Velocity,
				mass:     object.Mass.Mass,
			})
		}
	}
	g.massElementsCount = len(massObjects)
	bufferSize := int(unsafe.Sizeof(ObjectWithMass{})) * g.massElementsCount

	for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
		g.buffers[i].massBuffer, g.buffers[i].massMemory = dev.CreateBuffer(
			vulkan.DeviceSize(bufferSize),
			vulkan.BufferUsageFlags(vulkan.BufferUsageVertexBufferBit|vulkan.BufferUsageStorageBufferBit|vulkan.BufferUsageTransferDstBit),
			vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit),
		)
	}

	device.CopyWithStagingBufferCompute(dev, massObjects, func(cb vulkan.CommandBuffer, staging vulkan.Buffer) {
		for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
			vulkan.CmdCopyBuffer(cb, staging, g.buffers[i].massBuffer, 1, []vulkan.BufferCopy{
				{
					SrcOffset: 0,
					DstOffset: 0,
					Size:      vulkan.DeviceSize(bufferSize),
				},
			})
		}
	})

	for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
		vulkan.UpdateDescriptorSets(dev.LogicalDevice, 2, []vulkan.WriteDescriptorSet{
			{
				SType:           vulkan.StructureTypeWriteDescriptorSet,
				DstSet:          g.DescriptorsSets[i],
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				DescriptorCount: 1,
				PBufferInfo: []vulkan.DescriptorBufferInfo{
					{
						Buffer: g.buffers[(i+swapchain.MAX_FRAMES_IN_FLIGHT-1)%swapchain.MAX_FRAMES_IN_FLIGHT].massBuffer,
						Offset: 0,
						Range:  vulkan.DeviceSize(bufferSize),
					},
				},
			},
			{
				SType:           vulkan.StructureTypeWriteDescriptorSet,
				DstSet:          g.DescriptorsSets[i],
				DstBinding:      1,
				DstArrayElement: 0,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				DescriptorCount: 1,
				PBufferInfo: []vulkan.DescriptorBufferInfo{
					{
						Buffer: g.buffers[i].massBuffer,
						Offset: 0,
						Range:  vulkan.DeviceSize(bufferSize),
					},
				},
			},
		}, 0, nil)
	}
}

func (g *Gravity) UploadFieldObjects(
	dev *device.Device,
	objects []*object.GameObject,
) {
	var fieldObjects []VectorField
	for _, object := range objects {
		if object.Field != nil {
			fieldObjects = append(fieldObjects, VectorField{
				position: object.GetPosition(),
			})
		}
	}
	g.fieldElementsCount = len(fieldObjects)
	bufferSize := int(unsafe.Sizeof(VectorField{})) * g.fieldElementsCount

	for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
		g.buffers[i].forceBuffer, g.buffers[i].forceMemory = dev.CreateBuffer(
			vulkan.DeviceSize(g.fieldElementsCount*int(unsafe.Sizeof(ForceField{}))),
			vulkan.BufferUsageFlags(vulkan.BufferUsageVertexBufferBit|vulkan.BufferUsageStorageBufferBit),
			vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit),
		)
	}

	g.vecBuffer, g.vecMemory = dev.CreateBuffer(
		vulkan.DeviceSize(g.fieldElementsCount*int(unsafe.Sizeof(VectorField{}))),
		vulkan.BufferUsageFlags(vulkan.BufferUsageStorageBufferBit|vulkan.BufferUsageTransferDstBit),
		vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit),
	)

	device.CopyWithStagingBufferCompute(dev, fieldObjects, func(cb vulkan.CommandBuffer, staging vulkan.Buffer) {
		vulkan.CmdCopyBuffer(cb, staging, g.vecBuffer, 1, []vulkan.BufferCopy{
			{
				SrcOffset: 0,
				DstOffset: 0,
				Size:      vulkan.DeviceSize(bufferSize),
			},
		})
	})

	for i := range swapchain.MAX_FRAMES_IN_FLIGHT {
		vulkan.UpdateDescriptorSets(dev.LogicalDevice, 2, []vulkan.WriteDescriptorSet{
			{
				SType:           vulkan.StructureTypeWriteDescriptorSet,
				DstSet:          g.DescriptorsSets[i],
				DstBinding:      2,
				DstArrayElement: 0,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				DescriptorCount: 1,
				PBufferInfo: []vulkan.DescriptorBufferInfo{
					{
						Buffer: g.vecBuffer,
						Offset: 0,
						Range:  vulkan.DeviceSize(g.fieldElementsCount * int(unsafe.Sizeof(VectorField{}))),
					},
				},
			},
			{
				SType:           vulkan.StructureTypeWriteDescriptorSet,
				DstSet:          g.DescriptorsSets[i],
				DstBinding:      3,
				DstArrayElement: 0,
				DescriptorType:  vulkan.DescriptorTypeStorageBuffer,
				DescriptorCount: 1,
				PBufferInfo: []vulkan.DescriptorBufferInfo{
					{
						Buffer: g.buffers[i].forceBuffer,
						Offset: 0,
						Range:  vulkan.DeviceSize(g.fieldElementsCount * int(unsafe.Sizeof(ForceField{}))),
					},
				},
			},
		}, 0, nil)
	}
}

func (g *Gravity) ComputeGravity(
	commandBuffer vulkan.CommandBuffer,
	frameIdx uint32,
) {
	vulkan.CmdBindPipeline(commandBuffer, vulkan.PipelineBindPointCompute, g.pipelines[0])
	lastRenderedTime := float32(time.Since(g.lastRenderedTime).Seconds())
	g.lastRenderedTime = time.Now()

	steps := int(lastRenderedTime / 0.001)
	steps = (steps/swapchain.MAX_FRAMES_IN_FLIGHT + 1) * swapchain.MAX_FRAMES_IN_FLIGHT
	currentIdx := frameIdx
	for range steps {
		vulkan.CmdBindDescriptorSets(commandBuffer, vulkan.PipelineBindPointCompute, g.pipelinesLayout, 0, 1, []vulkan.DescriptorSet{
			g.DescriptorsSets[currentIdx],
		}, 0, nil)
		vulkan.CmdPushConstants(commandBuffer, g.pipelinesLayout, vulkan.ShaderStageFlags(vulkan.ShaderStageComputeBit), 0, uint32(unsafe.Sizeof(pushMassData{})), unsafe.Pointer(&pushMassData{
			timeSince:   lastRenderedTime / float32(steps),
			numElements: uint32(g.massElementsCount),
		}))
		vulkan.CmdDispatch(commandBuffer, 1, 1, 1)
		currentIdx = (currentIdx + 1) % swapchain.MAX_FRAMES_IN_FLIGHT
	}
}

func (g *Gravity) ComputeGravityField(
	commandBuffer vulkan.CommandBuffer,
	frameIdx uint32,
) (vulkan.Semaphore, vulkan.DescriptorSet) {
	vulkan.CmdBindPipeline(commandBuffer, vulkan.PipelineBindPointCompute, g.pipelines[1])
	vulkan.CmdPushConstants(commandBuffer, g.pipelinesLayout, vulkan.ShaderStageFlags(vulkan.ShaderStageComputeBit), 0, uint32(unsafe.Sizeof(pushFieldData{})), unsafe.Pointer(&pushFieldData{
		totalElements: uint32(g.fieldElementsCount),
		numElements:   uint32(g.massElementsCount),
	}))
	vulkan.CmdBindDescriptorSets(commandBuffer, vulkan.PipelineBindPointCompute, g.pipelinesLayout, 0, 1, []vulkan.DescriptorSet{
		g.DescriptorsSets[frameIdx],
	}, 0, nil)

	vulkan.CmdDispatch(commandBuffer, 8, 1, 1)

	if err := vulkan.Error(vulkan.EndCommandBuffer(commandBuffer)); err != nil {
		panic("failed to end command buffer: " + err.Error())
	}

	if err := vulkan.Error(vulkan.QueueSubmit(g.device.ComputeQueue, 1, []vulkan.SubmitInfo{
		{
			SType:              vulkan.StructureTypeSubmitInfo,
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vulkan.Semaphore{
				g.computeFinished[(frameIdx+swapchain.MAX_FRAMES_IN_FLIGHT-1)%swapchain.MAX_FRAMES_IN_FLIGHT],
			},
			PWaitDstStageMask: []vulkan.PipelineStageFlags{
				vulkan.PipelineStageFlags(vulkan.PipelineStageComputeShaderBit),
			},
			CommandBufferCount:   1,
			PCommandBuffers:      []vulkan.CommandBuffer{commandBuffer},
			SignalSemaphoreCount: 2,
			PSignalSemaphores: []vulkan.Semaphore{
				g.computeFinished[frameIdx],
				g.computeFinishedForGraphics[frameIdx],
			},
		},
	}, nil)); err != nil {
		panic("failed to submit compute command buffer: " + err.Error())
	}

	return g.computeFinishedForGraphics[frameIdx], g.DescriptorsSets[frameIdx]
}

func (g *Gravity) Close() {
	for _, semaphore := range g.computeFinished {
		vulkan.DestroySemaphore(g.device.LogicalDevice, semaphore, nil)
	}

	for _, semaphore := range g.computeFinishedForGraphics {
		vulkan.DestroySemaphore(g.device.LogicalDevice, semaphore, nil)
	}

	for _, buffer := range g.buffers {
		vulkan.DestroyBuffer(g.device.LogicalDevice, buffer.massBuffer, nil)
		vulkan.FreeMemory(g.device.LogicalDevice, buffer.massMemory, nil)
		vulkan.DestroyBuffer(g.device.LogicalDevice, buffer.forceBuffer, nil)
		vulkan.FreeMemory(g.device.LogicalDevice, buffer.forceMemory, nil)
	}
	vulkan.DestroyBuffer(g.device.LogicalDevice, g.vecBuffer, nil)
	vulkan.FreeMemory(g.device.LogicalDevice, g.vecMemory, nil)
	for _, pipeline := range g.pipelines {
		vulkan.DestroyPipeline(g.device.LogicalDevice, pipeline, nil)
	}
	vulkan.DestroyPipelineLayout(g.device.LogicalDevice, g.pipelinesLayout, nil)
	vulkan.DestroyDescriptorSetLayout(g.device.LogicalDevice, g.DescriptorsLayout, nil)
	vulkan.DestroyDescriptorPool(g.device.LogicalDevice, g.descriptorsPool, nil)
	vulkan.DestroyShaderModule(g.device.LogicalDevice, g.massModule, nil)
	vulkan.DestroyShaderModule(g.device.LogicalDevice, g.fieldModule, nil)
}
