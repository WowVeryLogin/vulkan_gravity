package swapchain

import (
	"errors"
	"fmt"

	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/goki/vulkan"
)

type SwapchainFactory struct {
	Swapchain  *Swapchain
	RenderPass vulkan.RenderPass

	syncObjects []syncObject

	surfaceFormat vulkan.SurfaceFormat
	presentMode   vulkan.PresentMode
	depthFormat   vulkan.Format
}

func New(device *device.Device, windowExtent vulkan.Extent2D) *SwapchainFactory {
	sp := device.SwapchainSupport()
	surfaceFormat := chooseSwapSurfaceFormat(sp.Formats)
	presentMode := chooseSwapPresentMode(sp.Presents)

	depthFormat := findDepthFormat(device)
	renderPass := createRenderPass(device, surfaceFormat.Format, depthFormat)

	syncObjects := createSyncObjects(device)

	sf := SwapchainFactory{
		RenderPass:    renderPass,
		surfaceFormat: surfaceFormat,
		presentMode:   presentMode,
		depthFormat:   depthFormat,
		syncObjects:   syncObjects,
	}

	sf.Swapchain = sf.newSwapchain(device, windowExtent, sf.syncObjects)

	return &sf
}

func (sf *SwapchainFactory) Close() {
	sf.Swapchain.Close()
	vulkan.DestroyRenderPass(sf.Swapchain.device.LogicalDevice, sf.RenderPass, nil)
	for _, syncObjects := range sf.syncObjects {
		vulkan.DestroySemaphore(sf.Swapchain.device.LogicalDevice, syncObjects.imageAvailable, nil)
		vulkan.DestroySemaphore(sf.Swapchain.device.LogicalDevice, syncObjects.renderFinished, nil)
		vulkan.DestroyFence(sf.Swapchain.device.LogicalDevice, syncObjects.inFlightFence, nil)
	}
}

func (sf *SwapchainFactory) UpdateSwapchain(extent vulkan.Extent2D) {
	oldSwapchain := sf.Swapchain
	sf.Swapchain = sf.newSwapchain(sf.Swapchain.device, extent, sf.syncObjects)
	oldSwapchain.Close()
}

type Swapchain struct {
	syncObjects    []syncObject
	device         *device.Device
	views          []vulkan.ImageView
	swapchain      vulkan.Swapchain
	depthResources []depthImage
	imagesInFlight []vulkan.Fence

	Extent       vulkan.Extent2D
	FrameBuffers []vulkan.Framebuffer

	ImageCount int
}

func chooseSwapSurfaceFormat(formats []vulkan.SurfaceFormat) vulkan.SurfaceFormat {
	for i := range formats {
		formats[i].Deref()
		if formats[i].Format == vulkan.FormatB8g8r8Srgb && formats[i].ColorSpace == vulkan.ColorSpaceSrgbNonlinear {
			return formats[i]
		}
	}

	return formats[0]
}

func chooseSwapPresentMode(modes []vulkan.PresentMode) vulkan.PresentMode {
	for _, mode := range modes {
		if mode == vulkan.PresentModeMailbox {
			return mode
		}
	}

	return vulkan.PresentModeFifo
}

const MaxUint32 = ^uint32(0)
const MaxUint64 = ^uint64(0)

func chooseSwapExtent(windowExtent vulkan.Extent2D, caps vulkan.SurfaceCapabilities) vulkan.Extent2D {
	if caps.CurrentExtent.Width != MaxUint32 {
		return caps.CurrentExtent
	}

	windowExtent.Width = max(caps.MinImageExtent.Width, min(caps.MaxImageExtent.Width, windowExtent.Width))
	windowExtent.Height = max(caps.MinImageExtent.Height, min(caps.MaxImageExtent.Height, windowExtent.Height))

	return windowExtent
}

func (swapchain *Swapchain) createSwapchain(
	device *device.Device,
	sp device.SwapchainProperties,
	surfaceFormat vulkan.SurfaceFormat,
	extent vulkan.Extent2D,
	presentMode vulkan.PresentMode,
	oldSwapchain vulkan.Swapchain,
) []vulkan.Image {
	imageCount := sp.Caps.MinImageCount + 1
	if sp.Caps.MaxImageCount > 0 && imageCount > sp.Caps.MaxImageCount {
		imageCount = sp.Caps.MaxImageCount
	}

	var newSwapchain vulkan.Swapchain
	if err := vulkan.Error(vulkan.CreateSwapchain(device.LogicalDevice, &vulkan.SwapchainCreateInfo{
		SType:            vulkan.StructureTypeSwapchainCreateInfo,
		Surface:          device.Surface,
		OldSwapchain:     oldSwapchain,
		MinImageCount:    imageCount,
		ImageFormat:      surfaceFormat.Format,
		ImageColorSpace:  surfaceFormat.ColorSpace,
		ImageExtent:      extent,
		ImageArrayLayers: 1,
		ImageUsage:       vulkan.ImageUsageFlags(vulkan.ImageUsageColorAttachmentBit),
		ImageSharingMode: vulkan.SharingModeExclusive,
		PreTransform:     sp.Caps.CurrentTransform,
		CompositeAlpha:   vulkan.CompositeAlphaOpaqueBit,
		PresentMode:      presentMode,
		Clipped:          vulkan.True,
	}, nil, &newSwapchain)); err != nil {
		panic("failed creating swapchain: " + err.Error())
	}
	swapchain.swapchain = newSwapchain

	var imagesCount uint32
	if err := vulkan.Error(vulkan.GetSwapchainImages(device.LogicalDevice, swapchain.swapchain, &imagesCount, nil)); err != nil {
		panic("failed to get images count: " + err.Error())
	}
	swapchain.ImageCount = int(imageCount)

	images := make([]vulkan.Image, imagesCount)
	if err := vulkan.Error(vulkan.GetSwapchainImages(device.LogicalDevice, swapchain.swapchain, &imagesCount, images)); err != nil {
		panic("failed to get swapchain images: " + err.Error())
	}

	return images
}

func createImageViews(device *device.Device, images []vulkan.Image, format vulkan.Format) []vulkan.ImageView {
	views := make([]vulkan.ImageView, len(images))
	for i, image := range images {
		if err := vulkan.Error(vulkan.CreateImageView(device.LogicalDevice, &vulkan.ImageViewCreateInfo{
			SType:    vulkan.StructureTypeImageViewCreateInfo,
			Image:    image,
			ViewType: vulkan.ImageViewType2d,
			Format:   format,
			SubresourceRange: vulkan.ImageSubresourceRange{
				AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		}, nil, &views[i])); err != nil {
			panic(fmt.Sprintf("failed to create image view (%d): %s", i, err))
		}
	}

	return views
}

func findDepthFormat(device *device.Device) vulkan.Format {
	return device.FindSupportedFormat([]vulkan.Format{
		vulkan.FormatD32Sfloat,
		vulkan.FormatD32SfloatS8Uint,
		vulkan.FormatD24UnormS8Uint,
	}, vulkan.ImageTilingOptimal, vulkan.FormatFeatureFlags(vulkan.FormatFeatureDepthStencilAttachmentBit))
}

func createRenderPass(device *device.Device, surfaceFormat vulkan.Format, depthFormat vulkan.Format) vulkan.RenderPass {
	var renderPass vulkan.RenderPass
	if err := vulkan.Error(vulkan.CreateRenderPass(device.LogicalDevice, &vulkan.RenderPassCreateInfo{
		SType:           vulkan.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 2,
		PAttachments: []vulkan.AttachmentDescription{
			{
				Format:         surfaceFormat,
				Samples:        vulkan.SampleCount1Bit,
				LoadOp:         vulkan.AttachmentLoadOpClear,
				StoreOp:        vulkan.AttachmentStoreOpStore,
				StencilLoadOp:  vulkan.AttachmentLoadOpDontCare,
				StencilStoreOp: vulkan.AttachmentStoreOpDontCare,
				InitialLayout:  vulkan.ImageLayoutUndefined,
				FinalLayout:    vulkan.ImageLayoutPresentSrc,
			},
			{
				Format:         depthFormat,
				Samples:        vulkan.SampleCount1Bit,
				LoadOp:         vulkan.AttachmentLoadOpClear,
				StoreOp:        vulkan.AttachmentStoreOpDontCare,
				StencilLoadOp:  vulkan.AttachmentLoadOpDontCare,
				StencilStoreOp: vulkan.AttachmentStoreOpDontCare,
				InitialLayout:  vulkan.ImageLayoutUndefined,
				FinalLayout:    vulkan.ImageLayoutDepthStencilAttachmentOptimal,
			},
		},
		SubpassCount: 1,
		PSubpasses: []vulkan.SubpassDescription{
			{
				PipelineBindPoint:    vulkan.PipelineBindPointGraphics,
				ColorAttachmentCount: 1,
				PColorAttachments: []vulkan.AttachmentReference{
					{
						Attachment: 0,
						Layout:     vulkan.ImageLayoutColorAttachmentOptimal,
					},
				},
				PDepthStencilAttachment: &vulkan.AttachmentReference{
					Attachment: 1,
					Layout:     vulkan.ImageLayoutDepthStencilAttachmentOptimal,
				},
			},
		},
		DependencyCount: 1,
		PDependencies: []vulkan.SubpassDependency{
			{
				SrcSubpass:    vulkan.SubpassExternal,
				SrcAccessMask: 0,
				SrcStageMask:  vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit | vulkan.PipelineStageEarlyFragmentTestsBit),
				DstSubpass:    0,
				DstStageMask:  vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit | vulkan.PipelineStageEarlyFragmentTestsBit),
				DstAccessMask: vulkan.AccessFlags(vulkan.AccessColorAttachmentWriteBit | vulkan.AccessDepthStencilAttachmentWriteBit),
			},
		},
	}, nil, &renderPass)); err != nil {
		panic("failed to create render pass: " + err.Error())
	}

	return renderPass
}

type depthImage struct {
	image  vulkan.Image
	memory vulkan.DeviceMemory
	view   vulkan.ImageView
}

func createDepthResources(
	device *device.Device,
	depthFormat vulkan.Format,
	extent vulkan.Extent2D,
	images []vulkan.Image,
) []depthImage {
	depthImages := make([]depthImage, len(images))
	for i := range depthImages {
		depthImages[i].image, depthImages[i].memory = device.CreateImageWithInfo(vulkan.ImageCreateInfo{
			SType:     vulkan.StructureTypeImageCreateInfo,
			ImageType: vulkan.ImageType2d,
			Extent: vulkan.Extent3D{
				Width:  extent.Width,
				Height: extent.Height,
				Depth:  1,
			},
			MipLevels:     1,
			ArrayLayers:   1,
			Format:        depthFormat,
			Tiling:        vulkan.ImageTilingOptimal,
			InitialLayout: vulkan.ImageLayoutUndefined,
			Usage:         vulkan.ImageUsageFlags(vulkan.ImageUsageDepthStencilAttachmentBit),
			Samples:       vulkan.SampleCount1Bit,
			SharingMode:   vulkan.SharingModeExclusive,
		}, vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit))

		if err := vulkan.Error(vulkan.CreateImageView(device.LogicalDevice, &vulkan.ImageViewCreateInfo{
			SType:    vulkan.StructureTypeImageViewCreateInfo,
			Image:    depthImages[i].image,
			ViewType: vulkan.ImageViewType2d,
			Format:   depthFormat,
			SubresourceRange: vulkan.ImageSubresourceRange{
				AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectDepthBit),
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		}, nil, &depthImages[i].view)); err != nil {
			panic("failed to create depth image view: " + err.Error())
		}
	}

	return depthImages
}

func createFrameBuffers(
	device *device.Device,
	images []vulkan.Image,
	imageViews []vulkan.ImageView,
	depthImages []depthImage,
	extent vulkan.Extent2D,
	renderPass vulkan.RenderPass,
) []vulkan.Framebuffer {
	framebuffers := make([]vulkan.Framebuffer, len(images))
	for i := range framebuffers {
		if err := vulkan.Error(vulkan.CreateFramebuffer(device.LogicalDevice, &vulkan.FramebufferCreateInfo{
			SType:           vulkan.StructureTypeFramebufferCreateInfo,
			RenderPass:      renderPass,
			AttachmentCount: 2,
			PAttachments: []vulkan.ImageView{
				imageViews[i],
				depthImages[i].view,
			},
			Width:  extent.Width,
			Height: extent.Height,
			Layers: 1,
		}, nil, &framebuffers[i])); err != nil {
			panic("failed to create framebuffer: " + err.Error())
		}
	}

	return framebuffers
}

const MAX_FRAMES_IN_FLIGHT = 2

type syncObject struct {
	imageAvailable vulkan.Semaphore
	renderFinished vulkan.Semaphore
	inFlightFence  vulkan.Fence
}

func createSyncObjects(
	device *device.Device,
) []syncObject {
	syncObjects := make([]syncObject, MAX_FRAMES_IN_FLIGHT)
	for i := range syncObjects {
		if err := vulkan.Error(vulkan.CreateSemaphore(device.LogicalDevice, &vulkan.SemaphoreCreateInfo{
			SType: vulkan.StructureTypeSemaphoreCreateInfo,
		}, nil, &syncObjects[i].imageAvailable)); err != nil {
			panic("failed to create image semaphore: " + err.Error())
		}

		if err := vulkan.Error(vulkan.CreateSemaphore(device.LogicalDevice, &vulkan.SemaphoreCreateInfo{
			SType: vulkan.StructureTypeSemaphoreCreateInfo,
		}, nil, &syncObjects[i].renderFinished)); err != nil {
			panic("failed to create render semaphore: " + err.Error())
		}

		if err := vulkan.Error(vulkan.CreateFence(device.LogicalDevice, &vulkan.FenceCreateInfo{
			SType: vulkan.StructureTypeFenceCreateInfo,
			Flags: vulkan.FenceCreateFlags(vulkan.FenceCreateSignaledBit),
		}, nil, &syncObjects[i].inFlightFence)); err != nil {
			panic("failed to create inflight fence: " + err.Error())
		}
	}

	return syncObjects
}

func (sf *SwapchainFactory) newSwapchain(
	device *device.Device,
	windowExtent vulkan.Extent2D,
	syncObjects []syncObject,
) *Swapchain {
	swapchainProps := device.SwapchainSupport()
	extent := chooseSwapExtent(windowExtent, swapchainProps.Caps)
	swapchain := &Swapchain{
		device:      device,
		Extent:      extent,
		syncObjects: syncObjects,
	}

	var oldSwapchain vulkan.Swapchain
	if sf.Swapchain != nil {
		oldSwapchain = sf.Swapchain.swapchain
	}

	images := swapchain.createSwapchain(device, swapchainProps, sf.surfaceFormat, extent, sf.presentMode, oldSwapchain)
	views := createImageViews(device, images, sf.surfaceFormat.Format)
	depthResources := createDepthResources(device, sf.depthFormat, extent, images)
	frameBuffers := createFrameBuffers(device, images, views, depthResources, extent, sf.RenderPass)

	swapchain.views = views
	swapchain.depthResources = depthResources
	swapchain.FrameBuffers = frameBuffers
	swapchain.imagesInFlight = make([]vulkan.Fence, len(images))

	return swapchain
}

var ErrOutOfDate = errors.New("error out of date")

func (s *Swapchain) NextImage(currentFrame int) (int, error) {
	vulkan.WaitForFences(s.device.LogicalDevice, 1, []vulkan.Fence{
		s.syncObjects[currentFrame].inFlightFence,
	}, vulkan.True, MaxUint64)

	var imageIndex uint32

	result := vulkan.AcquireNextImage(s.device.LogicalDevice, s.swapchain, MaxUint64, s.syncObjects[currentFrame].imageAvailable, vulkan.Fence(vulkan.NullHandle), &imageIndex)
	if result == vulkan.ErrorOutOfDate {
		return 0, ErrOutOfDate
	}
	if err := vulkan.Error(result); err != nil && result != vulkan.Suboptimal {
		panic("failed to get new image: " + err.Error())
	}

	return int(imageIndex), nil
}

func (s *Swapchain) SubmitCommandBuffer(
	buffers vulkan.CommandBuffer,
	currentFrame uint32,
	imageIndex uint32,
	// computeSemaphore vulkan.Semaphore,
) error {
	if s.imagesInFlight[imageIndex] != vulkan.Fence(vulkan.NullHandle) {
		vulkan.WaitForFences(s.device.LogicalDevice, 1, []vulkan.Fence{
			s.imagesInFlight[imageIndex],
		}, vulkan.True, MaxUint64)
	}
	s.imagesInFlight[imageIndex] = s.syncObjects[currentFrame].inFlightFence

	if err := vulkan.Error(vulkan.ResetFences(s.device.LogicalDevice, 1, []vulkan.Fence{s.syncObjects[currentFrame].inFlightFence})); err != nil {
		panic("failed to reset render fence: " + err.Error())
	}

	if err := vulkan.Error(vulkan.QueueSubmit(s.device.Queue, 1, []vulkan.SubmitInfo{
		{
			SType:              vulkan.StructureTypeSubmitInfo,
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vulkan.Semaphore{
				s.syncObjects[currentFrame].imageAvailable,
				// computeSemaphore,
			},
			PWaitDstStageMask: []vulkan.PipelineStageFlags{
				vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit),
				vulkan.PipelineStageFlags(vulkan.PipelineStageComputeShaderBit),
			},
			CommandBufferCount:   1,
			PCommandBuffers:      []vulkan.CommandBuffer{buffers},
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vulkan.Semaphore{
				s.syncObjects[currentFrame].renderFinished,
			},
		},
	}, s.syncObjects[currentFrame].inFlightFence)); err != nil {
		panic("failed to submit command buffer to queue: " + err.Error())
	}

	result := vulkan.QueuePresent(s.device.Queue, &vulkan.PresentInfo{
		SType:              vulkan.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vulkan.Semaphore{
			s.syncObjects[currentFrame].renderFinished,
		},
		SwapchainCount: 1,
		PSwapchains: []vulkan.Swapchain{
			s.swapchain,
		},
		PImageIndices: []uint32{imageIndex},
	})

	if result == vulkan.ErrorOutOfDate || result == vulkan.Suboptimal {
		return ErrOutOfDate
	}

	if err := vulkan.Error(result); err != nil {
		panic("failed to submit command buffer to queue: " + err.Error())
	}

	return nil
	// s.currentFrame = (s.currentFrame + 1) % MAX_FRAMES_IN_FLIGHT
}

func (s *Swapchain) Close() {
	for _, image := range s.views {
		vulkan.DestroyImageView(s.device.LogicalDevice, image, nil)
	}
	vulkan.DestroySwapchain(s.device.LogicalDevice, s.swapchain, nil)
	for _, depthResource := range s.depthResources {
		vulkan.DestroyImageView(s.device.LogicalDevice, depthResource.view, nil)
		vulkan.DestroyImage(s.device.LogicalDevice, depthResource.image, nil)
		vulkan.FreeMemory(s.device.LogicalDevice, depthResource.memory, nil)
	}

	for _, framebuffer := range s.FrameBuffers {
		vulkan.DestroyFramebuffer(s.device.LogicalDevice, framebuffer, nil)
	}
}
