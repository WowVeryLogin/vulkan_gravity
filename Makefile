run:
	glslc shaders/simple.vert -o shaders/vert.spv
	glslc shaders/simple.frag -o shaders/frag.spv
	glslc shaders/field.comp -o shaders/field.comp.spv
	glslc shaders/gravity.comp -o shaders/gravity.comp.spv
	VK_LOADER_DEBUG=all go run main.go