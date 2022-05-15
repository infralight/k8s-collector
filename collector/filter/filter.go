package filter

import "context"

type DataFilter func(ctx context.Context, data map[string][]interface{}) error

var All = []DataFilter{ArgoFilter}
