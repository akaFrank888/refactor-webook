package failover

import (
	"context"
	"refactor-webook/webook/internal/service/sms"
	"sync/atomic"
)

type TimeoutFailoverSmsService struct {
	svcs []sms.Service
	// 当前使用的服务商
	idx int32
	// 记录已经超时的个数
	cnt int32
	// 切换的阈值，只读（所以没有并发安全的问题）
	threshold int32
}

func (t *TimeoutFailoverSmsService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	length := len(t.svcs)
	// 原子操作保证拿到的是最新的 idx 和 cnt
	idx := atomic.LoadInt32(&t.idx)
	cnt := atomic.LoadInt32(&t.cnt)
	if cnt >= t.threshold {
		// 先计算下一个idx
		newIdx := (idx + 1) % (int32)(length)
		// note 注意此处的并发问题：可能有两个请求同时获得 newIdx。我们期望 idx 只因一个请求超过 threshold 而被赋值成 newIdx 即可，其他请求共享这个 idx
		// note 利用原子操作的 CAS 若返回 false，则说明 idx 已经因为其他请求而被修改成 newIdx；若返回 true，则说明是因为本请求修改的，要进一步将 cnt 置为 0
		if atomic.CompareAndSwapInt32(&t.idx, idx, newIdx) {
			atomic.StoreInt32(&t.cnt, 0)
		}
	}
	svc := t.svcs[t.idx]
	err := svc.Send(ctx, tplId, args, numbers...)
	switch err {
	case nil:
		// 请求没超时，重置 cnt
		atomic.StoreInt32(&t.cnt, 0)
	case context.DeadlineExceeded:
		// 请求超时，cnt++
		atomic.AddInt32(&t.cnt, 1)
	default:
		// 不是超时的错误
		// note 可以考虑若是 EOF 错误，可直接切换
	}
	return err
}
