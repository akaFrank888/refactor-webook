package failover

import (
	"context"
	"errors"
	"log"
	"refactor-webook/webook/internal/sms_project/service"
	"sync/atomic"
)

type FailOverSmsService struct {

	// 候选的服务商
	svcs []service.Service

	// failover第二种实现（指定svc[idx]开始轮询）的字段
	idx uint64
}

func NewFailOverSmsService(svcs []service.Service) *FailOverSmsService {
	return &FailOverSmsService{svcs: svcs}
}

// Send 从头开始轮询服务商，即 svc[0]开始
func (f *FailOverSmsService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	for _, svc := range f.svcs {
		err := svc.Send(ctx, tplId, args, numbers...)
		// note 若返回 err，则表明该短信服务商不可用，不必处理，遍历下一个
		if err == nil {
			return nil
		}

		log.Println(err)
	}
	return errors.New("轮询了所有服务商，Sms发送失败") // note 到这一步，应该是自己的问题（因为服务商都是高可用的）
}

// SendV1 可动态指定起始svc，从svc[idx]开始轮询服务商
func (f *FailOverSmsService) SendV1(ctx context.Context, tplId string, args []string, numbers ...string) error {
	// note 保证了即使在多线程或多协程环境下，对该变量的读取和写入操作也是原子的
	// note 确保每次调用 SendV1 时，都会从不同的服务商开始轮询，这样可以公平地分配请求给所有注册的服务商
	idx := atomic.AddUint64(&f.idx, 1)
	length := uint64(len(f.svcs))

	for i := idx; i < idx+length; i++ {
		// note 取余防溢出
		err := f.svcs[i%length].Send(ctx, tplId, args, numbers...)
		// note 根据是否是调用者主动造成的 err 进行分类
		switch err {
		case nil:
			return nil
		case context.Canceled, context.DeadlineExceeded:
			// note 调用者主动取消 和 ctx超时，结束轮询
			return err
		}
		log.Println(err)
	}
	return errors.New("轮询了所有服务商，Sms发送失败")
}
