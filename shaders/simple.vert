#version 450

layout(location = 0) in vec3 inVertexPos;
layout(location = 1) in vec3 color;
layout(location = 2) in vec3 inNormal;
layout(location = 3) in vec2 uv;

layout(location = 0) out vec3 fragColor;

layout(push_constant) uniform Push {
	mat4 projection;
	mat4 view;
	mat4 model;
	vec3 color;
} push;

const vec3 LIGHT_DIRECTION = normalize(vec3(1.0, -3.0, -1.0));
const float AMBIENT_LIGHT_INTENSITY = 0.02;

void main() {
	vec4 positionWorld = push.model * vec4(inVertexPos, 1.0);
	vec3 normalWorld = normalize(mat3(push.model) * inNormal);

	float lightIntensity = AMBIENT_LIGHT_INTENSITY + max(dot(normalWorld, LIGHT_DIRECTION), 0.0);

	gl_Position = push.projection * push.view * positionWorld;
	fragColor = lightIntensity * color;
}