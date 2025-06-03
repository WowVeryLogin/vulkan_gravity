#version 450

layout(location = 0) in vec3 inVertexPos;
layout(location = 1) in vec3 color;

layout(location = 0) out vec3 fragColor;

layout(push_constant) uniform Push {
	mat4 camera;
	mat4 transform;
	vec3 color;
} push;

void main() {
	gl_Position = push.camera * push.transform * vec4(inVertexPos, 1.0);
	fragColor = color;
}