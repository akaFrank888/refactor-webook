package async

import (
	"context"
	"refactor-webook/webook/internal/sms_project/domain"
	"refactor-webook/webook/internal/sms_project/repository"
	"refactor-webook/webook/internal/sms_project/service"
	"refactor-webook/webook/pkg/logger"
	"time"
)

type AsyncService struct {
	svc  service.Service
	repo repository.AsyncSmsRepository
	l    logger.LoggerV1
}

func NewAsyncService(svc service.Service, repo repository.AsyncSmsRepository) *AsyncService {
	res := &AsyncService{svc: svc, repo: repo}
	// 创建service后就开启异步发送消息的线程
	go func() {
		res.StartAsyncCycle()
	}()
	return res
}

func (s *AsyncService) StartAsyncCycle() {
	for {
		s.AsyncSend()
	}
}

func (s *AsyncService) AsyncSend() {
	// 不断询问数据库有无waiting的sms的过程中，避免无限制的阻塞 ==> 创建一个限时的ctx
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// note 基于 数据库行级锁 的机制，实现了抢占式调度
	// note 确保在 K8s环境中部署的多个 Pod 中，针对同一个异步短信发送请求，只有一个 Pod 实例能够成功获取并处理
	asyncSms, err := s.repo.PreemptWaitingSms(ctx)
	cancel() // note 执行完数据库操作就立即 cancel 掉 ctx （在哪里创建ctx就在哪里销毁）

	switch err {
	case nil:
		// 需要执行 发送 的操作，执行之前创建一个 ctx（和 ReportScheduleResult() 共用一个 ctx）
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = s.svc.Send(ctx, asyncSms.TplId, asyncSms.Args, asyncSms.Numbers...)
		if err != nil {
			s.l.Error("抢占成功，但发送短信失败", logger.Error(err), logger.Int64("id", asyncSms.Id))
		}
		// 若 err != nil，则 success = false
		success := err == nil
		// note 通知 repository 我这一次的执行结果
		err = s.repo.ReportScheduleResult(ctx, asyncSms.Id, success)
		if err != nil {
			s.l.Error("抢占和发送成功，但标记数据库失败",
				logger.Error(err),
				logger.Int64("id", asyncSms.Id),
				logger.Bool("res", success))
		}
	case repository.ErrWaitingSmsNotFound:
		// 没有需要异步发送的消息，可以睡眠一会等一等
		time.Sleep(time.Second)
	default:
		// 一般是数据库出了问题，但是为了尽量运行，还是要继续的
		// 睡眠可以规避掉 短时间的网络抖动问题
		s.l.Error("抢占式异步发送短信失败", logger.Error(err))
		time.Sleep(time.Second)
	}
}

func (s *AsyncService) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	if s.needAsync() {
		// 根据业务指标，该sms需要异步发送 ==> 先存入数据库，再等待异步发送
		err := s.repo.Add(ctx, domain.AsyncSms{
			TplId:   tplId,
			Args:    args,
			Numbers: numbers,

			RetryMax: 3, // note 设置可以重试 3 次
		})
		return err
	}
	// 不需要异步发送
	return s.svc.Send(ctx, tplId, args, numbers...)

}

func (s *AsyncService) needAsync() bool {
	// note 需要根据业务指标设计的，各种判定要不要触发异步的方案（判定服务商崩溃 和 触发了限流）
	// 1. 基于响应时间
	// 		1.1 绝对阈值，比如连续N个请求响应时间超过了 500ms，后续请求就转异步
	// 		1.2 变化趋势，比如当前一秒钟内的所有请求的响应时间比上一秒钟增长了 X%，后续请求就转异步
	// 2. 基于错误率：
	// 		2.1 一段时间内，收到 err 的请求比率大于 X%，后续请求就转异步
	// 		2.2 只要出现 EOF或者其他网络的error，后续请求就转异步
	// 3， 被服务商限流
	// 		3.1 基于服务商返回的特定的错误码

	// note 什么时候退出异步？
	// 1. 固定时间：进入异步 N 分钟后
	// 2. 流量比例：保留 10% 的流量（或者更少）进行同步发送，根据这部分请求的响应时间/错误率，进一步增大同步请求的比例
	// 		怎么控制 10%进行同步？ ===》 用随机数：设置0-100的随机数，当 <10时就同步，当 >10时就异步

	return true
}
