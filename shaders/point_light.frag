#version 450

layout(set = 0, binding = 0) uniform CameraUbo {
	mat4 projection;
	mat4 view;
	vec3 lightPosition;
	vec4 lightColour;
} cameraUbo;

layout(location = 0) in vec2 fragOffset;
layout (location = 0) out vec4 outColour;

void main() {
	float dis = sqrt(dot(fragOffset, fragOffset));
	if (dis >= 1.0) {
		discard;
	}
	outColour = vec4(cameraUbo.lightColour.xyz, 1.0);
}