#version 450

layout(location = 0) in vec3 inVertexPos;
layout(location = 1) in vec3 color;
layout(location = 2) in vec3 inNormal;
layout(location = 3) in vec2 uv;

layout(location = 0) out vec3 fragColor;

layout(set = 0, binding = 0) uniform WorldUbo {
	mat4 projection;
} worldUbo;

layout(set = 1, binding = 0) uniform CameraUbo {
	mat4 view;
	vec3 lightPosition;
	vec4 lightColour;
} cameraUbo;

layout(push_constant) uniform Push {
	mat4 model;
	vec3 color;
} push;

const vec3 LIGHT_DIRECTION = normalize(vec3(1.0, -3.0, -1.0));
const float AMBIENT_LIGHT_INTENSITY = 0.02;

void main() {
	vec4 positionWorld = push.model * vec4(inVertexPos, 1.0);
	vec3 normalWorld = normalize(mat3(push.model) * inNormal);

	vec3 directionToLight = cameraUbo.lightPosition - positionWorld.xyz;
	float attenuation = 1.0 / dot(directionToLight, directionToLight);
	vec3 lightColour = cameraUbo.lightColour.xyz * cameraUbo.lightColour.w * attenuation;

	vec3 pointLightIntensity = lightColour * max(dot(normalWorld, normalize(directionToLight)), 0.0);

	float lightIntensity = AMBIENT_LIGHT_INTENSITY + max(dot(normalWorld, LIGHT_DIRECTION), 0.0);

	gl_Position = worldUbo.projection * cameraUbo.view * positionWorld;
	fragColor = (lightIntensity + pointLightIntensity) * color;
}