package tel

import (
	"context"

	"connectrpc.com/connect"
)

func LogErrorsInterceptor(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			Log.Error("rpc", "response error", "err", err)
		}
		return resp, err
	}
}
