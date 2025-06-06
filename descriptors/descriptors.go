package descriptors

import (
	"game/device"

	"github.com/goki/vulkan"
)

type DescriptorsSet struct {
	device *device.Device
	Layout vulkan.DescriptorSetLayout
	pool   vulkan.DescriptorPool
	Set    vulkan.DescriptorSet
}

func NewDescriptors(device *device.Device) *DescriptorsSet {
	layout := newDescriptorSetLayout(device)
	pool := newDescriptorsPool(device)
	set := newDescriptorsSet(device, layout, pool)

	return &DescriptorsSet{
		device: device,
		Layout: layout,
		pool:   pool,
		Set:    set,
	}
}

func newDescriptorSetLayout(device *device.Device) vulkan.DescriptorSetLayout {
	var descriptorsLayout vulkan.DescriptorSetLayout
	if err := vulkan.Error(vulkan.CreateDescriptorSetLayout(device.LogicalDevice, &vulkan.DescriptorSetLayoutCreateInfo{
		SType:        vulkan.StructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 1,
		PBindings: []vulkan.DescriptorSetLayoutBinding{
			{
				Binding:         0,
				DescriptorCount: 1,
				DescriptorType:  vulkan.DescriptorTypeUniformBuffer,
				StageFlags:      vulkan.ShaderStageFlags(vulkan.ShaderStageVertexBit),
			},
		},
	}, nil, &descriptorsLayout)); err != nil {
		panic("failed to create descriptor set layout: " + err.Error())
	}

	return descriptorsLayout
}

func newDescriptorsPool(device *device.Device) vulkan.DescriptorPool {
	var descriptorsPool vulkan.DescriptorPool
	if err := vulkan.Error(vulkan.CreateDescriptorPool(device.LogicalDevice, &vulkan.DescriptorPoolCreateInfo{
		SType:         vulkan.StructureTypeDescriptorPoolCreateInfo,
		PoolSizeCount: 1,
		PPoolSizes: []vulkan.DescriptorPoolSize{
			{
				Type:            vulkan.DescriptorTypeUniformBuffer,
				DescriptorCount: 1,
			},
		},
		MaxSets: 1,
	}, nil, &descriptorsPool)); err != nil {
		panic("failed to create descriptor pool: " + err.Error())
	}

	return descriptorsPool
}

func newDescriptorsSet(
	device *device.Device,
	descriptorsLayout vulkan.DescriptorSetLayout,
	descriptorsPool vulkan.DescriptorPool,
) vulkan.DescriptorSet {
	descriptorsSets := make([]vulkan.DescriptorSet, 1)
	if err := vulkan.Error(vulkan.AllocateDescriptorSets(device.LogicalDevice, &vulkan.DescriptorSetAllocateInfo{
		SType:              vulkan.StructureTypeDescriptorSetAllocateInfo,
		DescriptorPool:     descriptorsPool,
		DescriptorSetCount: 1,
		PSetLayouts: []vulkan.DescriptorSetLayout{
			descriptorsLayout,
		},
	}, &descriptorsSets[0])); err != nil {
		panic("failed to allocate descriptor sets: " + err.Error())
	}

	return descriptorsSets[0]
}

func (d *DescriptorsSet) UpdateDescriptorSet(
	buffer vulkan.Buffer,
) {
	vulkan.UpdateDescriptorSets(d.device.LogicalDevice, 1, []vulkan.WriteDescriptorSet{
		{
			SType:           vulkan.StructureTypeWriteDescriptorSet,
			DstSet:          d.Set,
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  vulkan.DescriptorTypeUniformBuffer,
			DescriptorCount: 1,
			PBufferInfo: []vulkan.DescriptorBufferInfo{
				{
					Buffer: buffer,
					Offset: 0,
					Range:  vulkan.DeviceSize(vulkan.WholeSize),
				},
			},
		},
	}, 0, nil)
}

func (d *DescriptorsSet) Close() {
	vulkan.DestroyDescriptorPool(d.device.LogicalDevice, d.pool, nil)
	vulkan.DestroyDescriptorSetLayout(d.device.LogicalDevice, d.Layout, nil)
}
