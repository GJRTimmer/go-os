package auth

import (
	"github.com/micro/go-micro/client"
	co "github.com/micro/go-micro/context"
	"github.com/micro/go-micro/server"

	"golang.org/x/net/context"
)

type clientWrapper struct {
	client.Client
	a Auth
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// retrieve token if one exists
	t, err := c.a.Introspect(ctx)
	if err != nil {
		// no? ok let's try make the call as ourself
		t, err = c.a.Token()
		if err != nil {
			return err
		}
	}

	// create new context with token
	newCtx := c.a.NewContext(ctx, t)

	// set metadata
	newCtx = co.WithMetadata(newCtx, c.a.NewHeader(map[string]string{}, t))

	// circuit break, check authorization here
	t, err = c.a.Authorized(newCtx, req)
	if err != nil {
		return ErrInvalidToken
	}

	// now just make a regular call down the stack
	err = c.Client.Call(newCtx, req, rsp, opts...)
	return err
}

func handlerWrapper(fn server.HandlerFunc, a Auth) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		// retrieve token
		t, err := a.Introspect(ctx)
		if err != nil {
			return ErrInvalidToken
		}

		// check if authorized
		t, err = a.Authorized(ctx, req)
		if err != nil {
			return ErrInvalidToken
		}

		// create new context with token
		newCtx := a.NewContext(ctx, t)

		err = fn(newCtx, req, rsp)
		return err
	}
}
