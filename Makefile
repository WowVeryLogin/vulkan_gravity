run:
	glslc shaders/simple.vert -o shaders/vert.spv
	glslc shaders/simple.frag -o shaders/frag.spv
	glslc shaders/field.comp -o shaders/field.comp.spv
	glslc shaders/gravity.comp -o shaders/gravity.comp.spv
	glslc shaders/point_light.vert -o shaders/point_light.vert.spv
	glslc shaders/point_light.frag -o shaders/point_light.frag.spv
	VK_LOADER_DEBUG=all go run main.go