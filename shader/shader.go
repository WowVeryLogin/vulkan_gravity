package shader

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/goki/vulkan"
)

func sliceUint32(raw []byte) []uint32 {
	const SIZEOF_UINT32 = 4

	result := make([]uint32, len(raw)/SIZEOF_UINT32)
	for i := range result {
		// assuming little endian
		result[i] = uint32(binary.LittleEndian.Uint32(raw[i*SIZEOF_UINT32 : (i+1)*SIZEOF_UINT32]))
	}

	return result
}

func CreateShaderModule(path string, logicalDevice vulkan.Device) vulkan.ShaderModule {
	code := readFile(path)

	var shaderModule vulkan.ShaderModule
	if err := vulkan.Error(vulkan.CreateShaderModule(logicalDevice, &vulkan.ShaderModuleCreateInfo{
		SType:    vulkan.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint64(len(code)),
		PCode:    sliceUint32(code),
	}, nil, &shaderModule)); err != nil {
		panic("failed to create shader module: " + err.Error())
	}

	return shaderModule
}

func readFile(path string) []byte {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		panic("failed to open shader file: " + err.Error())
	}

	defer file.Close()
	buf, err := io.ReadAll(file)
	if err != nil {
		panic("failed to read shader file: " + err.Error())
	}

	return buf
}
