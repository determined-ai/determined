package grpcutil

import (
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// OrderBySQL maps our protobuf OrderBy enum to SQL. Unspecified maps to ASC.
var OrderBySQL = map[apiv1.OrderBy]string{
	apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
	apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
	apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
}
