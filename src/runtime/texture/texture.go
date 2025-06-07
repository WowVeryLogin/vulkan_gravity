package texture

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"log"

	"github.com/WowVeryLogin/vulkan_engine/src/runtime/device"
	"github.com/goki/vulkan"
)

type Texture struct {
	device    *device.Device
	image     vulkan.Image
	memory    vulkan.DeviceMemory
	ImageView vulkan.ImageView
	Sampler   vulkan.Sampler
}

type TextureConfig struct {
	Width  int
	Height int
	Data   []uint8
}

func TextureConfigFromPNG(reader io.Reader) *TextureConfig {
	decodedImage, err := png.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}

	bounds := decodedImage.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Convert image to RGBA
	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, decodedImage.At(x, y))
		}
	}
	pixelData := rgbaImg.Pix

	return &TextureConfig{
		Width:  width,
		Height: height,
		Data:   pixelData,
	}
}

func New(dev *device.Device, textureData *TextureConfig) *Texture {
	image, memory := dev.CreateImageWithInfo(vulkan.ImageCreateInfo{
		SType:     vulkan.StructureTypeImageCreateInfo,
		ImageType: vulkan.ImageType2d,
		Format:    vulkan.FormatR8g8b8a8Srgb,
		Extent: vulkan.Extent3D{
			Width:  uint32(textureData.Width),
			Height: uint32(textureData.Height),
			Depth:  1,
		},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     vulkan.SampleCount1Bit,
		Tiling:      vulkan.ImageTilingOptimal,
		Usage:       vulkan.ImageUsageFlags(vulkan.ImageUsageSampledBit | vulkan.ImageUsageTransferDstBit),
		SharingMode: vulkan.SharingModeExclusive,
	}, vulkan.MemoryPropertyFlags(vulkan.MemoryPropertyDeviceLocalBit))

	device.CopyWithStagingBufferGraphic(dev, textureData.Data, func(cb vulkan.CommandBuffer, staging vulkan.Buffer) {
		barrier := vulkan.ImageMemoryBarrier{
			SType:               vulkan.StructureTypeImageMemoryBarrier,
			OldLayout:           vulkan.ImageLayoutUndefined,
			NewLayout:           vulkan.ImageLayoutTransferDstOptimal,
			SrcQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			DstQueueFamilyIndex: vulkan.QueueFamilyIgnored,
			Image:               image,
			SubresourceRange: vulkan.ImageSubresourceRange{
				AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			SrcAccessMask: 0,
			DstAccessMask: vulkan.AccessFlags(vulkan.AccessTransferWriteBit),
		}
		vulkan.CmdPipelineBarrier(
			cb,
			vulkan.PipelineStageFlags(vulkan.PipelineStageTopOfPipeBit), // srcStage
			vulkan.PipelineStageFlags(vulkan.PipelineStageTransferBit),  // dstStage
			0,
			0, nil,
			0, nil,
			1, []vulkan.ImageMemoryBarrier{barrier},
		)

		vulkan.CmdCopyBufferToImage(cb, staging, image, vulkan.ImageLayoutTransferDstOptimal, 1, []vulkan.BufferImageCopy{
			{
				BufferOffset:      0,
				BufferRowLength:   0,
				BufferImageHeight: 0,
				ImageSubresource: vulkan.ImageSubresourceLayers{
					AspectMask: vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
					LayerCount: 1,
				},
				ImageExtent: vulkan.Extent3D{
					Width:  uint32(textureData.Width),
					Height: uint32(textureData.Height),
					Depth:  1,
				},
			},
		})

		barrier.OldLayout = vulkan.ImageLayoutTransferDstOptimal
		barrier.NewLayout = vulkan.ImageLayoutShaderReadOnlyOptimal
		barrier.SrcAccessMask = vulkan.AccessFlags(vulkan.AccessTransferWriteBit)
		barrier.DstAccessMask = vulkan.AccessFlags(vulkan.AccessShaderReadBit)

		vulkan.CmdPipelineBarrier(
			cb,
			vulkan.PipelineStageFlags(vulkan.PipelineStageTransferBit),
			vulkan.PipelineStageFlags(vulkan.PipelineStageFragmentShaderBit),
			0,
			0, nil,
			0, nil,
			1, []vulkan.ImageMemoryBarrier{barrier},
		)
	})

	var view vulkan.ImageView
	if err := vulkan.Error(vulkan.CreateImageView(dev.LogicalDevice, &vulkan.ImageViewCreateInfo{
		SType:    vulkan.StructureTypeImageViewCreateInfo,
		Image:    image,
		ViewType: vulkan.ImageViewType2d,
		Format:   vulkan.FormatR8g8b8a8Srgb,
		SubresourceRange: vulkan.ImageSubresourceRange{
			AspectMask:     vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}, nil, &view)); err != nil {
		panic(fmt.Sprintf("failed to create texture image view: %s", err))
	}

	var sampler vulkan.Sampler
	if err := vulkan.Error(vulkan.CreateSampler(dev.LogicalDevice, &vulkan.SamplerCreateInfo{
		SType:       vulkan.StructureTypeSamplerCreateInfo,
		MipmapMode:  vulkan.SamplerMipmapModeLinear,
		CompareOp:   vulkan.CompareOpAlways,
		BorderColor: vulkan.BorderColorFloatOpaqueBlack,
	}, nil, &sampler)); err != nil {
		panic("failed to create texture sampler: " + err.Error())
	}

	return &Texture{
		device:    dev,
		image:     image,
		ImageView: view,
		Sampler:   sampler,
		memory:    memory,
	}
}

func (t *Texture) Close() {
	vulkan.DestroySampler(t.device.LogicalDevice, t.Sampler, nil)
	vulkan.DestroyImageView(t.device.LogicalDevice, t.ImageView, nil)
	vulkan.DestroyImage(t.device.LogicalDevice, t.image, nil)
	vulkan.FreeMemory(t.device.LogicalDevice, t.memory, nil)
}
