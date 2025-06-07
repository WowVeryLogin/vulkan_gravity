package renderpass

import (
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/WowVeryLogin/vulkan_engine/src/runtime/swapchain"
	"github.com/goki/vulkan"
)

type MultistageCommandBuffer struct {
	ComputeCommandBuffer  vulkan.CommandBuffer
	GraphicsCommandBuffer vulkan.CommandBuffer
}

type RenderPass struct {
	RenderPass       vulkan.RenderPass
	CommandBuffers   []MultistageCommandBuffer
	swapchainFactory *swapchain.SwapchainFactory
	imageIdx         int
	FrameIdx         int
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

func New(device *device.Device, extent vulkan.Extent2D) *RenderPass {
	swapchainFactory := swapchain.New(device, extent)

	commandBuffers := createCommandBuffers(device)
	return &RenderPass{
		RenderPass:       swapchainFactory.RenderPass,
		swapchainFactory: swapchainFactory,
		CommandBuffers:   commandBuffers,
	}
}

func (r *RenderPass) GetAspectRatio() float32 {
	extent := r.swapchainFactory.Swapchain.Extent
	if extent.Height == 0 {
		return 1.0
	}
	return float32(extent.Width) / float32(extent.Height)
}

func (r *RenderPass) UpdateSwapchain(extent vulkan.Extent2D) {
	r.swapchainFactory.UpdateSwapchain(extent)
}

func (r *RenderPass) BeginFrame() (*MultistageCommandBuffer, error) {
	var err error
	r.imageIdx, err = r.swapchainFactory.Swapchain.NextImage(r.FrameIdx)
	if err == swapchain.ErrOutOfDate {
		return nil, err
	}

	cb := r.CommandBuffers[r.FrameIdx]

	if err := vulkan.Error(vulkan.BeginCommandBuffer(cb.GraphicsCommandBuffer, &vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
	})); err != nil {
		panic("failed to begin recording command buffer:" + err.Error())
	}

	return &cb, nil
}

func (r *RenderPass) BeginSwapChainRenderPass() {
	cb := r.CommandBuffers[r.FrameIdx].GraphicsCommandBuffer
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

func (r *RenderPass) EndSwapChainRenderPass() {
	vulkan.CmdEndRenderPass(r.CommandBuffers[r.FrameIdx].GraphicsCommandBuffer)
}

func (r *RenderPass) EndFrame() error {
	if err := vulkan.Error(vulkan.EndCommandBuffer(r.CommandBuffers[r.FrameIdx].GraphicsCommandBuffer)); err != nil {
		panic("failed to end command buffer: " + err.Error())
	}
	if err := r.swapchainFactory.Swapchain.SubmitCommandBuffer(
		r.CommandBuffers[r.FrameIdx].GraphicsCommandBuffer,
		uint32(r.FrameIdx),
		uint32(r.imageIdx),
	); err != nil {
		return err
	}
	r.FrameIdx = (r.FrameIdx + 1) % swapchain.MAX_FRAMES_IN_FLIGHT
	return nil
}

func (r *RenderPass) Close() {
	r.swapchainFactory.Close()
}
