package device

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/WowVeryLogin/vulkan_engine/src/window"
	"github.com/goki/vulkan"
)

func checkValidationLayers() {
	var layerCount uint32

	if err := vulkan.Error(vulkan.EnumerateInstanceLayerProperties(&layerCount, nil)); err != nil {
		panic("failed to enumerate instance layers: " + err.Error())
	}

	availableLayers := make([]vulkan.LayerProperties, layerCount)
	if err := vulkan.Error(vulkan.EnumerateInstanceLayerProperties(&layerCount, availableLayers)); err != nil {
		panic("failed to enumerate instance layers: " + err.Error())
	}

	for _, layer := range availableLayers {
		layer.Deref()
		if vulkan.ToString(layer.LayerName[:]) == "VK_LAYER_KHRONOS_validation" {
			return
		}
	}

	panic(fmt.Sprintf("validation layer %s not found", "VK_LAYER_KHRONOS_validation"))
}

func newInstance(withValidation bool, w *window.Window) vulkan.Instance {
	if withValidation {
		checkValidationLayers()
	}

	extensions := w.GetRequiredInstanceExtensions()
	if withValidation {
		extensions = append(extensions, vulkan.ExtDebugUtilsExtensionName+"\x00")
	}
	extensions = append(extensions, vulkan.KhrPortabilityEnumerationExtensionName+"\x00")

	createInfo := vulkan.InstanceCreateInfo{
		SType: vulkan.StructureTypeInstanceCreateInfo,
		PApplicationInfo: &vulkan.ApplicationInfo{
			SType:              vulkan.StructureTypeApplicationInfo,
			PApplicationName:   "Game App\x00",
			ApplicationVersion: vulkan.MakeVersion(1, 0, 0),
			PEngineName:        "Game Engine\x00",
			EngineVersion:      vulkan.MakeVersion(1, 0, 0),
			ApiVersion:         vulkan.MakeVersion(1, 3, 0),
		},
		EnabledExtensionCount:   uint32(len(extensions)),
		PpEnabledExtensionNames: extensions,
		EnabledLayerCount:       0,
		Flags:                   vulkan.InstanceCreateFlags(vulkan.InstanceCreateEnumeratePortabilityBit),
	}
	if withValidation {
		createInfo.EnabledLayerCount = 1
		createInfo.PpEnabledLayerNames = []string{"VK_LAYER_KHRONOS_validation\x00"}
	}

	var instance vulkan.Instance
	if err := vulkan.Error(vulkan.CreateInstance(&createInfo, nil, &instance)); err != nil {
		panic("failed to create instance: " + err.Error())
	}

	if err := vulkan.InitInstance(instance); err != nil {
		panic("failed to init instance: " + err.Error())
	}

	return instance
}

func deviceIsSuitable(device vulkan.PhysicalDevice) bool {
	var extensionCount uint32
	if err := vulkan.Error(vulkan.EnumerateDeviceExtensionProperties(device, "", &extensionCount, nil)); err != nil {
		panic("failed to enumerate device extensions: " + err.Error())
	}

	extensions := make([]vulkan.ExtensionProperties, extensionCount)
	if err := vulkan.Error(vulkan.EnumerateDeviceExtensionProperties(device, "", &extensionCount, extensions)); err != nil {
		panic("failed to enumerate device extensions: " + err.Error())
	}

	for _, extension := range extensions {
		extension.Deref()
		if vulkan.ToString(extension.ExtensionName[:]) == vulkan.KhrSwapchainExtensionName {
			return true
		}
	}

	return false
}

func pickPhysicalDevice(instance vulkan.Instance) (vulkan.PhysicalDevice, error) {
	var devicesCount uint32
	if err := vulkan.Error(vulkan.EnumeratePhysicalDevices(instance, &devicesCount, nil)); err != nil {
		panic("failed to enumerate physical devices: " + err.Error())
	}
	devices := make([]vulkan.PhysicalDevice, devicesCount)
	if err := vulkan.Error(vulkan.EnumeratePhysicalDevices(instance, &devicesCount, devices)); err != nil {
		panic("failed to enumerate physical devices: " + err.Error())
	}

	for _, device := range devices {
		if deviceIsSuitable(device) {
			return device, nil
		}
	}

	return nil, errors.New("no suitable device found")
}

func findGraphicQueueFamily(
	device vulkan.PhysicalDevice,
	surface vulkan.Surface,
	queuesFamilies []vulkan.QueueFamilyProperties,
) int {
	for i, family := range queuesFamilies {
		family.Deref()
		if family.QueueCount > 0 && family.QueueFlags&vulkan.QueueFlags(vulkan.QueueGraphicsBit) > 0 {
			var supported vulkan.Bool32
			vulkan.GetPhysicalDeviceSurfaceSupport(device, uint32(i), surface, &supported)
			if supported.B() {
				return i
			}
		}
	}

	panic("no suitable queue family found")
}

func findComputeQueueFamily(
	queuesFamilies []vulkan.QueueFamilyProperties,
) int {
	for i, family := range queuesFamilies {
		family.Deref()
		if family.QueueCount > 0 && family.QueueFlags&vulkan.QueueFlags(vulkan.QueueComputeBit) > 0 {
			return i
		}
	}

	panic("no suitable queue family found")
}

func getQueueFamilies(device vulkan.PhysicalDevice) []vulkan.QueueFamilyProperties {
	var queueFamilyCount uint32
	vulkan.GetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, nil)
	queuesFamilies := make([]vulkan.QueueFamilyProperties, queueFamilyCount)
	vulkan.GetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, queuesFamilies)

	return queuesFamilies
}

func createLogicalDevice(device vulkan.PhysicalDevice, queueIdx int) vulkan.Device {
	var logicalDevice vulkan.Device
	if err := vulkan.Error(vulkan.CreateDevice(device, &vulkan.DeviceCreateInfo{
		SType:                vulkan.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount: 1,
		PQueueCreateInfos: []vulkan.DeviceQueueCreateInfo{
			{
				SType:            vulkan.StructureTypeDeviceQueueCreateInfo,
				QueueFamilyIndex: uint32(queueIdx),
				QueueCount:       1,
				PQueuePriorities: []float32{1.0},
			},
		},
		PEnabledFeatures: []vulkan.PhysicalDeviceFeatures{
			{
				SamplerAnisotropy: vulkan.True,
			},
		},

		EnabledExtensionCount: 2,
		PpEnabledExtensionNames: []string{
			vulkan.KhrSwapchainExtensionName + "\x00",
			vulkan.KhrPortabilitySubsetExtensionName + "\x00",
		},
	}, nil, &logicalDevice)); err != nil {
		panic("failed to create logical device: " + err.Error())
	}

	return logicalDevice
}

func createCommandPool(device vulkan.Device, queueIdx int) vulkan.CommandPool {
	var pool vulkan.CommandPool
	if err := vulkan.Error(vulkan.CreateCommandPool(device, &vulkan.CommandPoolCreateInfo{
		SType:            vulkan.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: uint32(queueIdx),
		Flags:            vulkan.CommandPoolCreateFlags(vulkan.CommandPoolCreateTransientBit | vulkan.CommandPoolCreateResetCommandBufferBit),
	}, nil, &pool)); err != nil {
		panic("failed to create command pool: " + err.Error())
	}
	return pool
}

type Device struct {
	instance        vulkan.Instance
	Surface         vulkan.Surface
	queueIdx        int
	computeQueueIdx int
	Queue           vulkan.Queue
	ComputeQueue    vulkan.Queue
	physicalDevice  vulkan.PhysicalDevice
	LogicalDevice   vulkan.Device
	Pool            vulkan.CommandPool
	ComputePool     vulkan.CommandPool
}

func New(w *window.Window) *Device {
	instance := newInstance(true, w)
	surface := w.CreateSurface(instance)
	device, err := pickPhysicalDevice(instance)
	if err != nil {
		panic("failed to pick physical device: " + err.Error())
	}
	queueFamilies := getQueueFamilies(device)
	queueIdx := findGraphicQueueFamily(device, surface, queueFamilies)
	computeQueueIdx := findComputeQueueFamily(queueFamilies)

	logicalDevice := createLogicalDevice(device, queueIdx)

	var queue vulkan.Queue
	vulkan.GetDeviceQueue(logicalDevice, uint32(queueIdx), 0, &queue)

	var computeQueue vulkan.Queue
	vulkan.GetDeviceQueue(logicalDevice, uint32(computeQueueIdx), 0, &computeQueue)

	pool := createCommandPool(logicalDevice, queueIdx)
	computePool := createCommandPool(logicalDevice, computeQueueIdx)

	return &Device{
		instance:        instance,
		Surface:         surface,
		physicalDevice:  device,
		queueIdx:        queueIdx,
		computeQueueIdx: computeQueueIdx,
		LogicalDevice:   logicalDevice,
		Queue:           queue,
		ComputeQueue:    computeQueue,
		Pool:            pool,
		ComputePool:     computePool,
	}
}

func (v *Device) findMemoryType(typeFilter uint32, properties vulkan.MemoryPropertyFlags) uint32 {
	var memProperties vulkan.PhysicalDeviceMemoryProperties
	vulkan.GetPhysicalDeviceMemoryProperties(v.physicalDevice, &memProperties)
	memProperties.Deref()

	for i := uint32(0); i < memProperties.MemoryTypeCount; i++ {
		memProperties.MemoryTypes[i].Deref()
		if typeFilter&(1<<i) != 0 && (memProperties.MemoryTypes[i].PropertyFlags&properties) == properties {
			return i
		}
	}

	panic("failed to find suitable memory type")
}

func (v *Device) CreateBuffer(
	size vulkan.DeviceSize,
	usage vulkan.BufferUsageFlags,
	memProperties vulkan.MemoryPropertyFlags,
) (vulkan.Buffer, vulkan.DeviceMemory) {
	var buffer vulkan.Buffer
	if err := vulkan.Error(vulkan.CreateBuffer(v.LogicalDevice, &vulkan.BufferCreateInfo{
		SType:       vulkan.StructureTypeBufferCreateInfo,
		Size:        size,
		Usage:       usage,
		SharingMode: vulkan.SharingModeExclusive,
	}, nil, &buffer)); err != nil {
		panic("failed to create buffer: " + err.Error())
	}

	var memRequirements vulkan.MemoryRequirements
	vulkan.GetBufferMemoryRequirements(v.LogicalDevice, buffer, &memRequirements)
	memRequirements.Deref()

	var memory vulkan.DeviceMemory
	if err := vulkan.Error(vulkan.AllocateMemory(v.LogicalDevice, &vulkan.MemoryAllocateInfo{
		SType:           vulkan.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: v.findMemoryType(memRequirements.MemoryTypeBits, memProperties),
	}, nil, &memory)); err != nil {
		panic("failed to allocate buffer: " + err.Error())
	}

	if err := vulkan.Error(vulkan.BindBufferMemory(v.LogicalDevice, buffer, memory, 0)); err != nil {
		panic("failed to bind buffer memory: " + err.Error())
	}

	return buffer, memory
}

func (v *Device) CreateImageWithInfo(
	createInfo vulkan.ImageCreateInfo,
	properties vulkan.MemoryPropertyFlags,
) (vulkan.Image, vulkan.DeviceMemory) {
	var image vulkan.Image
	if err := vulkan.Error(vulkan.CreateImage(v.LogicalDevice, &createInfo, nil, &image)); err != nil {
		panic("failed to create image: " + err.Error())
	}

	var memReq vulkan.MemoryRequirements
	vulkan.GetImageMemoryRequirements(v.LogicalDevice, image, &memReq)
	memReq.Deref()

	var imageMemory vulkan.DeviceMemory
	if err := vulkan.Error(vulkan.AllocateMemory(v.LogicalDevice, &vulkan.MemoryAllocateInfo{
		SType:           vulkan.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: v.findMemoryType(memReq.MemoryTypeBits, properties),
	}, nil, &imageMemory)); err != nil {
		panic("failed to allocate memory for image: " + err.Error())
	}

	if err := vulkan.Error(vulkan.BindImageMemory(v.LogicalDevice, image, imageMemory, 0)); err != nil {
		panic("failed to bind image to memory: " + err.Error())
	}

	return image, imageMemory
}

func (v *Device) FindSupportedFormat(
	candidates []vulkan.Format,
	tiling vulkan.ImageTiling,
	features vulkan.FormatFeatureFlags,
) vulkan.Format {
	for _, format := range candidates {
		var properties vulkan.FormatProperties
		vulkan.GetPhysicalDeviceFormatProperties(v.physicalDevice, format, &properties)
		properties.Deref()

		if tiling == vulkan.ImageTilingLinear && (properties.LinearTilingFeatures&features) == features {
			return format
		}
		if tiling == vulkan.ImageTilingOptimal && (properties.OptimalTilingFeatures&features) == features {
			return format
		}
	}
	panic("failed to find supported format")
}

type SwapchainProperties struct {
	Caps     vulkan.SurfaceCapabilities
	Formats  []vulkan.SurfaceFormat
	Presents []vulkan.PresentMode
}

func CopyWithStagingBufferGraphic[T any](
	device *Device,
	initialBuffer []T,
	copyFn func(commandBuffer vulkan.CommandBuffer, staging vulkan.Buffer),
) {
	copyWithStagingBuffer(device, device.Queue, device.Pool, initialBuffer, copyFn)
}

func CopyWithStagingBufferCompute[T any](
	device *Device,
	initialBuffer []T,
	copyFn func(commandBuffer vulkan.CommandBuffer, staging vulkan.Buffer),
) {
	copyWithStagingBuffer(device, device.ComputeQueue, device.ComputePool, initialBuffer, copyFn)
}

func copyWithStagingBuffer[T any](
	device *Device,
	queue vulkan.Queue,
	pool vulkan.CommandPool,
	initialBuffer []T,
	copyFn func(commandBuffer vulkan.CommandBuffer, staging vulkan.Buffer),
) {
	bufferSize := len(initialBuffer) * int(unsafe.Sizeof(initialBuffer[0]))

	buffer, memory := device.CreateBuffer(
		vulkan.DeviceSize(bufferSize),
		vulkan.BufferUsageFlags(vulkan.BufferUsageTransferSrcBit),
		vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyHostVisibleBit|vulkan.MemoryPropertyHostCoherentBit),
	)

	var data unsafe.Pointer
	if err := vulkan.Error(vulkan.MapMemory(device.LogicalDevice, memory, 0, vulkan.DeviceSize(bufferSize), 0, &data)); err != nil {
		panic("failed to map buffer memory: " + err.Error())
	}
	slice := unsafe.Slice((*T)(data), len(initialBuffer))
	copy(slice, initialBuffer)
	vulkan.UnmapMemory(device.LogicalDevice, memory)

	commandBuffer := make([]vulkan.CommandBuffer, 1)
	if err := vulkan.Error(vulkan.AllocateCommandBuffers(device.LogicalDevice, &vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandPool:        pool,
		CommandBufferCount: 1,
	}, commandBuffer)); err != nil {
		panic("failed to allocate command buffers: " + err.Error())
	}

	if err := vulkan.Error(vulkan.BeginCommandBuffer(commandBuffer[0], &vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
		Flags: vulkan.CommandBufferUsageFlags(vulkan.CommandBufferUsageOneTimeSubmitBit),
	})); err != nil {
		panic("failed to begin recording command buffer: " + err.Error())
	}

	copyFn(commandBuffer[0], buffer)

	if err := vulkan.Error(vulkan.EndCommandBuffer(commandBuffer[0])); err != nil {
		panic("failed to end command buffer: " + err.Error())
	}

	vulkan.QueueSubmit(queue, 1, []vulkan.SubmitInfo{
		{
			SType:              vulkan.StructureTypeSubmitInfo,
			CommandBufferCount: 1,
			PCommandBuffers:    commandBuffer,
		},
	}, nil)
	vulkan.QueueWaitIdle(queue)
	vulkan.FreeCommandBuffers(device.LogicalDevice, pool, 1, commandBuffer)
	vulkan.DestroyBuffer(device.LogicalDevice, buffer, nil)
	vulkan.FreeMemory(device.LogicalDevice, memory, nil)
}

func (v *Device) SwapchainSupport() SwapchainProperties {
	sp := SwapchainProperties{}

	if err := vulkan.Error(vulkan.GetPhysicalDeviceSurfaceCapabilities(v.physicalDevice, v.Surface, &sp.Caps)); err != nil {
		panic("failed quering surface caps: " + err.Error())
	}
	sp.Caps.Deref()
	sp.Caps.CurrentExtent.Deref()
	sp.Caps.MaxImageExtent.Deref()
	sp.Caps.MinImageExtent.Deref()

	var formatCount uint32
	if err := vulkan.Error(vulkan.GetPhysicalDeviceSurfaceFormats(v.physicalDevice, v.Surface, &formatCount, nil)); err != nil {
		panic("failed quering surface formats: " + err.Error())
	}

	if formatCount != 0 {
		sp.Formats = make([]vulkan.SurfaceFormat, formatCount)
		if err := vulkan.Error(vulkan.GetPhysicalDeviceSurfaceFormats(v.physicalDevice, v.Surface, &formatCount, sp.Formats)); err != nil {
			panic("failed quering surface formats: " + err.Error())
		}
	}

	var presentCount uint32
	if err := vulkan.Error(vulkan.GetPhysicalDeviceSurfacePresentModes(v.physicalDevice, v.Surface, &presentCount, nil)); err != nil {
		panic("failed quering surface present modes: " + err.Error())
	}

	if presentCount != 0 {
		sp.Presents = make([]vulkan.PresentMode, presentCount)
		if err := vulkan.Error(vulkan.GetPhysicalDeviceSurfacePresentModes(v.physicalDevice, v.Surface, &presentCount, sp.Presents)); err != nil {
			panic("failed quering surface present modes: " + err.Error())
		}
	}

	return sp
}

func (v *Device) Close() {
	vulkan.DestroyCommandPool(v.LogicalDevice, v.ComputePool, nil)
	vulkan.DestroyCommandPool(v.LogicalDevice, v.Pool, nil)
	vulkan.DestroyDevice(v.LogicalDevice, nil)
	vulkan.DestroySurface(v.instance, v.Surface, nil)
	vulkan.DestroyInstance(v.instance, nil)
}
