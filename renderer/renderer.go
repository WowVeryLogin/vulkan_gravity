package renderer

import (
	"game/device"
	"game/swapchain"

	"github.com/goki/vulkan"
)

type MultistageCommandBuffer struct {
	ComputeCommandBuffer  vulkan.CommandBuffer
	GraphicsCommandBuffer vulkan.CommandBuffer
}

type Renderer struct {
	RenderPass       vulkan.RenderPass
	CommandBuffers   []MultistageCommandBuffer
	swapchainFactory *swapchain.SwapchainFactory
	imageIdx         int
	frameIdx         int
}

func createCommandBuffers(device *device.Device) []MultistageCommandBuffer {
	commandBuffers := make([]vulkan.CommandBuffer, swapchain.MAX_FRAMES_IN_FLIGHT)
	if err := vulkan.Error(vulkan.AllocateCommandBuffers(device.LogicalDevice, &vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandPool:        device.Pool,
		CommandBufferCount: uint32(len(commandBuffers)),
	}, commandBuffers)); err != nil {
		panic("failed to allocate command buffers: " + err.Error())
	}

	computeCommandBuffers := make([]vulkan.CommandBuffer, swapchain.MAX_FRAMES_IN_FLIGHT)
	if err := vulkan.Error(vulkan.AllocateCommandBuffers(device.LogicalDevice, &vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandPool:        device.ComputePool,
		CommandBufferCount: uint32(len(computeCommandBuffers)),
	}, computeCommandBuffers)); err != nil {
		panic("failed to allocate command buffers: " + err.Error())
	}

	multistageCommandBuffers := make([]MultistageCommandBuffer, swapchain.MAX_FRAMES_IN_FLIGHT)
	for i := range multistageCommandBuffers {
		multistageCommandBuffers[i].GraphicsCommandBuffer = commandBuffers[i]
		multistageCommandBuffers[i].ComputeCommandBuffer = computeCommandBuffers[i]
	}

	return multistageCommandBuffers
}

func New(device *device.Device, extent vulkan.Extent2D) *Renderer {
	swapchainFactory := swapchain.New(device, extent)

	commandBuffers := createCommandBuffers(device)
	return &Renderer{
		RenderPass:       swapchainFactory.RenderPass,
		swapchainFactory: swapchainFactory,
		CommandBuffers:   commandBuffers,
	}
}

func (r *Renderer) UpdateSwapchain(extent vulkan.Extent2D) {
	r.swapchainFactory.UpdateSwapchain(extent)
}

func (r *Renderer) BeginFrame() (*MultistageCommandBuffer, uint32, error) {
	var err error
	r.imageIdx, err = r.swapchainFactory.Swapchain.NextImage(r.frameIdx)
	if err == swapchain.ErrOutOfDate {
		return nil, 0, err
	}

	cb := r.CommandBuffers[r.frameIdx]

	if err := vulkan.Error(vulkan.BeginCommandBuffer(cb.ComputeCommandBuffer, &vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
	})); err != nil {
		panic("failed to begin recording command buffer:" + err.Error())
	}

	return &cb, uint32(r.frameIdx), nil
}

func (r *Renderer) BeginSwapChainRenderPass() {
	cb := r.CommandBuffers[r.frameIdx].GraphicsCommandBuffer

	if err := vulkan.Error(vulkan.BeginCommandBuffer(cb, &vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
	})); err != nil {
		panic("failed to begin recording command buffer:" + err.Error())
	}

	vulkan.CmdBeginRenderPass(cb, &vulkan.RenderPassBeginInfo{
		SType:       vulkan.StructureTypeRenderPassBeginInfo,
		RenderPass:  r.swapchainFactory.RenderPass,
		Framebuffer: r.swapchainFactory.Swapchain.FrameBuffers[r.imageIdx],
		RenderArea: vulkan.Rect2D{
			Offset: vulkan.Offset2D{
				X: 0,
				Y: 0,
			},
			Extent: r.swapchainFactory.Swapchain.Extent,
		},
		ClearValueCount: 2,
		PClearValues: []vulkan.ClearValue{
			vulkan.NewClearValue([]float32{0.1, 0.1, 0.1, 1.0}),
			vulkan.NewClearDepthStencil(1.0, 0),
		},
	}, vulkan.SubpassContentsInline)

	vulkan.CmdSetViewport(cb, 0, 1, []vulkan.Viewport{
		{
			Width:    float32(r.swapchainFactory.Swapchain.Extent.Width),
			Height:   float32(r.swapchainFactory.Swapchain.Extent.Height),
			MinDepth: 0,
			MaxDepth: 1,
		},
	})
	vulkan.CmdSetScissor(cb, 0, 1, []vulkan.Rect2D{
		{
			Offset: vulkan.Offset2D{},
			Extent: r.swapchainFactory.Swapchain.Extent,
		},
	})

}

func (r *Renderer) EndSwapChainRenderPass() {
	vulkan.CmdEndRenderPass(r.CommandBuffers[r.frameIdx].GraphicsCommandBuffer)
}

func (r *Renderer) EndFrame(computeSemaphore vulkan.Semaphore) {
	if err := vulkan.Error(vulkan.EndCommandBuffer(r.CommandBuffers[r.frameIdx].GraphicsCommandBuffer)); err != nil {
		panic("failed to end command buffer: " + err.Error())
	}
	r.swapchainFactory.Swapchain.SubmitCommandBuffer(
		r.CommandBuffers[r.frameIdx].GraphicsCommandBuffer,
		uint32(r.frameIdx),
		uint32(r.imageIdx),
		computeSemaphore,
	)
	r.frameIdx = (r.frameIdx + 1) % swapchain.MAX_FRAMES_IN_FLIGHT
}

func (r *Renderer) Close() {
	r.swapchainFactory.Close()
}
