package sproto

import "github.com/determined-ai/determined/proto/pkg/resourcepoolv1"

// StringFromResourcePoolTypeProto returns a string from the protobuf resource pool type.
func StringFromResourcePoolTypeProto(t resourcepoolv1.ResourcePoolType) string {
	switch t {
	case resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC:
		return "static"
	case resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_AWS:
		return "aws"
	case resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_GCP:
		return "gcp"
	case resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S:
		return "k8s"
	default:
		return "unspecified"
	}
}
