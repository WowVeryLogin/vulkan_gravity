package descriptors

import (
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"

	"github.com/goki/vulkan"
)

type SetsManager struct {
	device *device.Device
	pool   vulkan.DescriptorPool
}

type DescriptorSet struct {
	device      *device.Device
	Descriptors []Descriptor
	Layout      vulkan.DescriptorSetLayout
	Set         vulkan.DescriptorSet
}

type Descriptor struct {
	Type   vulkan.DescriptorType
	Flags  vulkan.ShaderStageFlags
	Buffer vulkan.Buffer
	Image  vulkan.Image
}

func NewSets(
	device *device.Device,
	sets []*DescriptorSet,
) SetsManager {
	uniqueDescriptors := map[vulkan.DescriptorType]int{}
	for _, set := range sets {
		for _, descriptor := range set.Descriptors {
			uniqueDescriptors[descriptor.Type]++
		}
	}

	for _, set := range sets {
		layoutBindings := []vulkan.DescriptorSetLayoutBinding{}
		for _, descriptor := range set.Descriptors {
			layoutBindings = append(layoutBindings, vulkan.DescriptorSetLayoutBinding{
				Binding:         0,
				DescriptorCount: 1,
				DescriptorType:  descriptor.Type,
				StageFlags:      descriptor.Flags,
			})
		}
		var descriptorsLayout vulkan.DescriptorSetLayout
		if err := vulkan.Error(vulkan.CreateDescriptorSetLayout(device.LogicalDevice, &vulkan.DescriptorSetLayoutCreateInfo{
			SType:        vulkan.StructureTypeDescriptorSetLayoutCreateInfo,
			BindingCount: uint32(len(layoutBindings)),
			PBindings:    layoutBindings,
		}, nil, &descriptorsLayout)); err != nil {
			panic("failed to create descriptor set layout: " + err.Error())
		}
		set.Layout = descriptorsLayout
		set.device = device
	}

	sizes := []vulkan.DescriptorPoolSize{}
	for descType, count := range uniqueDescriptors {
		sizes = append(sizes, vulkan.DescriptorPoolSize{
			Type:            descType,
			DescriptorCount: uint32(count),
		})
	}

	var descriptorsPool vulkan.DescriptorPool
	if err := vulkan.Error(vulkan.CreateDescriptorPool(device.LogicalDevice, &vulkan.DescriptorPoolCreateInfo{
		SType:         vulkan.StructureTypeDescriptorPoolCreateInfo,
		PoolSizeCount: uint32(len(sizes)),
		PPoolSizes:    sizes,
		MaxSets:       uint32(len(sets)),
	}, nil, &descriptorsPool)); err != nil {
		panic("failed to create descriptor pool: " + err.Error())
	}

	for _, set := range sets {
		var s vulkan.DescriptorSet
		if err := vulkan.Error(vulkan.AllocateDescriptorSets(device.LogicalDevice, &vulkan.DescriptorSetAllocateInfo{
			SType:              vulkan.StructureTypeDescriptorSetAllocateInfo,
			DescriptorPool:     descriptorsPool,
			DescriptorSetCount: 1,
			PSetLayouts: []vulkan.DescriptorSetLayout{
				set.Layout,
			},
		}, &s)); err != nil {
			panic("failed to allocate descriptor sets: " + err.Error())
		}
		set.Set = s
	}

	writeDescSets := []vulkan.WriteDescriptorSet{}
	for _, set := range sets {
		bufInfo := []vulkan.DescriptorBufferInfo{}
		for _, descriptor := range set.Descriptors {
			bufInfo = append(bufInfo, vulkan.DescriptorBufferInfo{
				Buffer: descriptor.Buffer,
				Offset: 0,
				Range:  vulkan.DeviceSize(vulkan.WholeSize),
			})
		}
		writeDescSets = append(writeDescSets, vulkan.WriteDescriptorSet{
			SType:           vulkan.StructureTypeWriteDescriptorSet,
			DstSet:          set.Set,
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  vulkan.DescriptorTypeUniformBuffer,
			DescriptorCount: uint32(len(bufInfo)),
			PBufferInfo:     bufInfo,
		})
	}
	vulkan.UpdateDescriptorSets(device.LogicalDevice, uint32(len(writeDescSets)), writeDescSets, 0, nil)

	return SetsManager{
		device: device,
		pool:   descriptorsPool,
	}
}

func (d *DescriptorSet) Close() {
	vulkan.DestroyDescriptorSetLayout(d.device.LogicalDevice, d.Layout, nil)
}

func (m *SetsManager) Close() {
	vulkan.DestroyDescriptorPool(m.device.LogicalDevice, m.pool, nil)
}
