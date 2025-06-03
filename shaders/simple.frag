#version 450

layout(location = 0) in vec3 fragColor;

layout (location = 0) out vec4 outColour;

layout(push_constant) uniform Push {
	mat4 camera;
	mat4 transform;
	vec3 color;
} push;

void main() {
	outColour = vec4(fragColor, 1.0);
}