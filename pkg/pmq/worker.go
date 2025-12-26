package pmq

import (
	"context"
	"runtime"

	// "github.com/streadway/amqp"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	amqp "github.com/rabbitmq/amqp091-go"
)

// --------------------------------------------------
// Consume协程 -[amqp.Delivery]-> Delivery协程 -[mqMsg]-> Worker协程
type mqMsg struct {
	mqDelivery amqp.Delivery
	cb         func(ctx context.Context) error
	cbName     string
}

func (cli *AmqpClient) initPublicWorker() {
	cli.mqConsumerChan = make(chan mqMsg)
	workerCnt := runtime.NumCPU() / 2
	if workerCnt < 4 {
		workerCnt = 4
	}
	for i := int32(0); i != int32(workerCnt); i++ {
		go cli.worker(cli.mqConsumerChan)
	}
}

func (cli *AmqpClient) closeWorker() {
	plogger.Debug("closing all msg chan")

	close(cli.mqConsumerChan)
	cli.mqConsumerChan = nil

	for _, c := range cli.mqConsumerPriChanList {
		close(c)
	}
	cli.mqConsumerPriChanList = cli.mqConsumerPriChanList[:0]

	plogger.Debug("closing all msg chan, done")
}

// 从msgChan获取数据，构造mqMsg然后提交给worker处理
// newWorker 启用一个专用协程，“单线程”地处理对应队列中的请求
func (cli *AmqpClient) handleDelivery(msgChan chan amqp.Delivery,
	cb func(ctx context.Context) error, queue string, newWorker bool) {

	priChan := make(chan mqMsg)
	if newWorker {
		cli.mqConsumerPriChanList = append(cli.mqConsumerPriChanList, priChan)
		go cli.worker(priChan)
	}
	// plogger.Debugf("delivery set [%v]", queue)
	go func() {
		for d := range msgChan {
			rpcMsg := mqMsg{mqDelivery: d, cb: cb, cbName: queue}
			// plogger.Debugf("delivery get msg [%v]", queue)
			if newWorker {
				priChan <- rpcMsg
			} else {
				cli.mqConsumerChan <- rpcMsg
			}
			// 原本是接受并塞到内部队列就应答，改为业务处理完毕根据业务是否成功来应答
			// d.Ack(false)
		}
		// 主动关闭，[优雅退出程序]
		plogger.Debugf("delivery exit [%v]", queue)
	}()
}

// --------------------------------------------------

func (cli *AmqpClient) worker(msgChan chan mqMsg) {
	workerId := putil.DefaultIDMgr.GetNewSmallestAndUniqueID()
	defer putil.DefaultIDMgr.ReleaseID(workerId)

	cli.workerWaitGroup.Add(1)
	defer cli.workerWaitGroup.Done()

	for {
		//从队列中读取一条信息
		msg, ok := <-msgChan
		if !ok {
			plogger.Debugf("worker exit [%v]", workerId)
			break
		}
		// plogger.Debugf("worker get msg [%v]", msg.cbName)
		//调用不同的消息处理
		err := cli.handleMQMsg(&msg, workerId)

		if msg.mqDelivery.Type != msgType_rpc && // rpc不用重试，立刻返回调用方错误
			err != nil {
			// msg.mqDelivery.Headers // TODO 需要精准控制重试N次，则header自定义变量记录重试次数
			if msg.mqDelivery.Redelivered {
				plogger.Errorf("w[%v] call[%v] err[%v] nack", workerId, msg.cbName, err)
				msg.mqDelivery.Nack(false, false) // 重试失败了，就放弃了

			} else {
				plogger.Errorf("w[%v] call[%v] err[%v] requeue", workerId, msg.cbName, err)
				msg.mqDelivery.Nack(false, true)
			}
		} else {
			// plogger.Debugf("w[%v] call[%v] done", workerId, msg.cbName)
			msg.mqDelivery.Ack(false)
		}
	}
}

func (cli *AmqpClient) logReqMsg(workerId int32, keyStr string, cbName string,
	correlationId string, detail string) {
	if cli.isLogMessage == 0 {
		plogger.Debugf("w[%v] %v[%v] ID[%v] l[%v]", workerId, keyStr, cbName,
			correlationId, len(detail))
	} else if cli.isLogMessage == 1 && len(detail) > 4096 {
		plogger.Debugf("w[%v] %v[%v] ID[%v] l[%v] msg : %v", workerId, keyStr, cbName,
			correlationId, len(detail), detail[0:4096])
	} else {
		plogger.Debugf("w[%v] %v[%v] ID[%v] l[%v] msg : %v", workerId, keyStr, cbName,
			correlationId, len(detail), detail)
	}
}

func (cli *AmqpClient) logRespMsg(workerId int32, keyStr string, cbName string,
	correlationId string, detail string) {
	if cli.isLogMessage == 0 {
		plogger.Debugf("w[%v] %v[%v] ID[%v] e[%v] l[%v]", workerId, keyStr, cbName,
			correlationId, len(detail))
	} else if cli.isLogMessage == 1 && len(detail) > 4096 {
		plogger.Debugf("w[%v] %v[%v] ID[%v] e[%v] l[%v] msg : %v", workerId, keyStr, cbName,
			correlationId, len(detail), detail[0:4096])
	} else {
		plogger.Debugf("w[%v] %v[%v] ID[%v] e[%v] l[%v] msg : %v", workerId, keyStr, cbName,
			correlationId, len(detail), detail)
	}
}

// --------------------------------------------------
type PMQContext struct {
	workerNum int32
	cbName    string
	Req       string
	Resp      string
}
type _PMQContextKey struct{}

var PMQContextKey = _PMQContextKey{}

func GetPMQContext(ctx context.Context) *PMQContext {
	return ctx.Value(PMQContextKey).(*PMQContext)
}

// 流程上，一个消息被调用处理函数之前，注册到这里的函数将被调用，用于：
// 1：编写框架性代码，如框架性数据结构的封包和解包；
// 2：公共的业务代码，如接口权限控制。
// 该方法如果返回错误，则消息处理函数不会被调用，直接返回resp给调用方。
func (cli *AmqpClient) SetBeforeCall(f func(ctx context.Context) error) {
	cli.beforeCall = f
}

func (cli *AmqpClient) handleMQMsg(rpcMsg *mqMsg, i int32) error {
	var pmqCtx PMQContext
	pmqCtx.Req = string(rpcMsg.mqDelivery.Body)

	ctx := context.WithValue(context.Background(), PMQContextKey, &pmqCtx)
	ctx = context.WithValue(ctx, putil.PgoTraceIDKey, putil.UUID_S())

	plogger.Debug("-------------------------------------------------------------------")
	cli.logReqMsg(i, "svr mq req ", rpcMsg.cbName, rpcMsg.mqDelivery.CorrelationId, pmqCtx.Req)

	if cli.beforeCall != nil {
		err := cli.beforeCall(ctx)
		if err != nil {
			plogger.LogErr(err)
			cli.publishResponse(&pmqCtx, &rpcMsg.mqDelivery)
			return err
		}
	}
	err := rpcMsg.cb(ctx)
	plogger.LogErr(err)
	cli.publishResponse(&pmqCtx, &rpcMsg.mqDelivery)
	return err
}

func (cli *AmqpClient) publishResponse(pmqCtx *PMQContext, delivery *amqp.Delivery) {
	if delivery.ReplyTo == "" {
		plogger.Debug("delivery.ReplyTo is empty, wont send response")
		return
	}

	cli.logRespMsg(pmqCtx.workerNum, "svr mq resp", pmqCtx.cbName,
		delivery.CorrelationId, pmqCtx.Resp)

	// 返回resp的时候，能否用消费者的channel，而不是用公共的channel呢？怎么封装才能实现？
	err := cli.ch.mqChannel.Publish(
		"",               // exchange
		delivery.ReplyTo, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: delivery.CorrelationId,
			Body:          []byte(pmqCtx.Resp),
		})
	if err != nil {
		plogger.Error("Failed to publish a response message")
		return // rpc返回失败了，能怎么处理呢，只能让调用方等到超时了
	}
}
