#version 450

struct ForceObject {
	vec2 force;
};

struct MassObject {
	vec2 position;
	vec2 velocity;
	float mass;
};

layout(location = 0) in vec2 inVertexPos;
layout(location = 1) in vec3 color;

layout(std140, binding = 1) readonly buffer InMass{
	MassObject massObjectsIn[];
};

layout(std140, binding = 3) readonly buffer InForce{
	ForceObject forceObjectIn[];
};

layout(push_constant) uniform Push {
	mat2 transform;
	vec2 offset;
	uint isField;
	uint index;
	vec3 color;
} push;

void main() {	
	if (push.isField == 0) {
		MassObject obj = massObjectsIn[push.index];

		// Transform vertex by object position (no rotation or scaling here yet)
		vec2 worldPos = push.transform * inVertexPos + obj.position;
		gl_Position = vec4(worldPos, 0.0, 1.0);
	} else {
		ForceObject force = forceObjectIn[push.index];
		float forceMag = length(force.force);
		float angle = atan(force.force[0], force.force[1]);

		// Rotation matrix
		mat2 rotation = mat2(
			cos(angle), -sin(angle),
			sin(angle), cos(angle)
		);

		float scale = 0.005 + 0.045 * clamp(log(forceMag + 1) / 3.0, 0.0, 1.0);

		vec2 scaledVertex = rotation * mat2(
			1.0, 0.0,
			0.0, scale * 200.0
		) * push.transform * (vec2(0.0, 0.5) + inVertexPos) + push.offset;
		gl_Position = vec4(scaledVertex, 0.0, 1.0);
	}
}