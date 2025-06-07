#version 450

layout(location = 0) in vec3 worldPosition;
layout(location = 1) in vec3 fragNormalWorld;
layout(location = 2) in vec2 texCoord;

layout (location = 0) out vec4 outColour;

layout(set = 0, binding = 0) uniform CameraUbo {
	mat4 projection;
	mat4 view;
	vec3 lightPosition;
	vec4 lightColour;
} cameraUbo;

layout(set = 0, binding = 1) uniform sampler2D avocadoSampler;
layout(set = 0, binding = 2) uniform sampler2D boxSampler;

layout(push_constant) uniform Push {
	mat4 model;
	int textureType;
} push;

const vec3 LIGHT_DIRECTION = normalize(vec3(1.0, -3.0, -1.0));
const float AMBIENT_LIGHT_INTENSITY = 0.02;

void main() {
	vec3 directionToLight = cameraUbo.lightPosition - worldPosition;
	float attenuation = 1.0 / (0.5 * dot(directionToLight, directionToLight));
	
	vec3 lightColour = cameraUbo.lightColour.xyz * cameraUbo.lightColour.w * attenuation;
	vec3 pointLightIntensity = lightColour * max(dot(normalize(fragNormalWorld), normalize(directionToLight)), 0.0);

	float gobalLightIntensity = AMBIENT_LIGHT_INTENSITY + 0.5 * max(dot(normalize(fragNormalWorld), LIGHT_DIRECTION), 0.0);


	vec4 colour;
	if (push.textureType == 0) {
		colour = texture(avocadoSampler, texCoord);
	} else if (push.textureType == 1) {
		colour = texture(boxSampler, texCoord);
	} else {
		colour = vec4(.72, .72, .72, 1.0);
	}
	outColour = vec4((gobalLightIntensity + pointLightIntensity) * colour.xyz, 1.0);
}