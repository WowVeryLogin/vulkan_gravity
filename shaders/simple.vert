#version 450

layout(location = 0) in vec3 inVertexPos;
layout(location = 1) in vec3 color;
layout(location = 2) in vec3 inNormal;
layout(location = 3) in vec2 uv;

layout(location = 0) out vec3 fragColor;
layout(location = 1) out vec3 worldPosition;
layout(location = 2) out vec3 fragNormalWorld;
layout(location = 3) out vec2 texCoord;

layout(set = 0, binding = 0) uniform CameraUbo {
	mat4 projection;
	mat4 view;
	vec3 lightPosition;
	vec4 lightColour;
} cameraUbo;

layout(push_constant) uniform Push {
	mat4 model;
	int textureType;
} push;

void main() {
	vec4 positionWorld = push.model * vec4(inVertexPos, 1.0);
	fragNormalWorld = normalize(mat3(push.model) * inNormal);
	worldPosition = positionWorld.xyz;

	gl_Position = cameraUbo.projection * cameraUbo.view * positionWorld;
	fragColor = color;
	texCoord = uv;
}