package sms

import "context"

type Service interface {
	// Send tplId：模板id
	Send(ctx context.Context, tplId string, args []string, numbers ...string) error
}
