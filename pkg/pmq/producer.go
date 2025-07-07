package pmq

import (
	"context"
	"time"

	"github.com/kataras/iris/v12/x/errors"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// --------------------------------------------------
const (
	msgType_rpc = "pgo-rpc" //内部rpc调用，rpc处理失败不会重新入队
	msgType_msg = "pgo-msg" //内部通知，普通消息，如果消费失败返回错误码，则将重新入队
)

// --------------------------------------------------
func (cli *AmqpClient) RpcCall(ctx context.Context, queue string,
	reqPB protoreflect.ProtoMessage, respPB protoreflect.ProtoMessage,
) error {
	return cli.rpcCall(ctx, "", queue, queue, reqPB, respPB)
}

func (cli *AmqpClient) RpcCallStr(ctx context.Context, queue string,
	req, resp *string,
) error {
	return cli.rpcCallStr(ctx, "", queue, queue, req, resp)
}

func (cli *AmqpClient) SendSimpleEvent(ctx context.Context,
	msg protoreflect.ProtoMessage) error {
	//直接使用proto定义的名字作为路由，发送到内部通知默认交换机
	return cli.SendServerEvent(ctx, cli.defaultEventExchange,
		string(msg.ProtoReflect().Descriptor().Name()), msg)
}

func (cli *AmqpClient) SendServerEvent(ctx context.Context,
	exchange, route string, request protoreflect.ProtoMessage) error {
	return cli.rpcCall(ctx, exchange, route, route, request, nil)
}

func (cli *AmqpClient) SendServerEventStr(ctx context.Context,
	exchange, route string, request *string) error {
	return cli.rpcCallStr(ctx, exchange, route, route, request, nil)
}

// --------------------------------------------------

func (cli *AmqpClient) rpcCall(ctx context.Context,
	exchange, route, queue string,
	reqPB protoreflect.ProtoMessage, respPB protoreflect.ProtoMessage,
) error {

	var req string
	err := putil.MessageToString(reqPB, &req)
	if err != nil {
		return plogger.LogErr(err)
	}

	if respPB == nil {
		return cli.rpcCallStr(ctx, exchange, route, queue, &req, nil)
	}

	var resp string
	err = cli.rpcCallStr(ctx, exchange, route, queue, &req, &resp)
	if err != nil {
		return plogger.LogErr(err)
	}

	err = putil.StringToMessage(&resp, respPB)
	if err != nil {
		return plogger.LogErr(err)
	}

	return nil
}

func (cli *AmqpClient) rpcCallStr(ctx context.Context,
	exchange, route, queue string,
	req *string, resp *string,
) error {
	if req == nil {
		req = new(string)
	}

	if cli.queuePrefix != "" {
		route = cli.queuePrefix + "_" + route
		queue = cli.queuePrefix + "_" + queue
	}
	if cli.queueSuffix != "" {
		route += "_" + cli.queueSuffix
		queue += "_" + cli.queueSuffix
	}

	// 想要打印类似wokerid，reqid之类的外部信息，应该由ctx提供固定参数
	// 而比起固定参数，更好的是提供闭包，则ctx提供logger函数
	cli.logReqMsg(0, "cli mq send", queue, "", *req)

	// 这里使用临时的channel，这样保证完成后随着channel的回收，把返回队列以及消费者也清掉
	mqChannel, err := cli.conn.Channel()
	if err != nil {
		return plogger.LogErr(err)
	}
	defer mqChannel.Close()

	corrId := putil.UUID_S()

	if resp == nil {
		//发送消息
		err = mqChannel.Publish(
			exchange, // exchange
			route,    // routing key
			false,    // mandatory
			false,    // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: corrId,
				ReplyTo:       "",
				Body:          []byte(*req),
				Expiration:    amqp.NeverExpire,
				//标记我们自己的消息，相应的用pb.InternalRequest解析最外层
				Type: msgType_msg,
			})
		if err != nil {
			return plogger.LogErr(err)
		}
		return nil
	}

	//构建响应队列
	q, err := mqChannel.QueueDeclare(
		"",    // name
		false, // durable
		true,  // delete when unused
		true,  // exclusive 排他队列，只能由声明的连接使用
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return plogger.LogErr(err)
	}

	defer mqChannel.QueueDelete(q.Name, false, false, false)

	//消费响应队列
	respDelivery, err := mqChannel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return plogger.LogErr(err)
	}

	var ttl time.Duration
	ddl, ok := ctx.Deadline()
	if !ok {
		ttl = time.Until(ddl)
	}

	//发送消息
	err = mqChannel.Publish(
		exchange, // exchange
		route,    // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       q.Name,
			Body:          []byte(*req),
			//超时了都没被消费，就不用了
			Expiration: putil.Int64ToStr(ttl.Milliseconds()),
			Type:       msgType_rpc,
		})
	if err != nil {
		return plogger.LogErr(err)
	}

	select {
	case delivery := <-respDelivery:
		*resp = string(delivery.Body)
		cli.logRespMsg(0, "cli mq recv", queue, delivery.CorrelationId, *resp)
		return nil

	case <-time.After(ttl):
		//mqChannel为临时变量，回收时将自动取消消费“返回队列”，返回队列将自动关闭

		plogger.Errorf("Failed to call [%v], timeout [%v] + [%v]",
			queue, 0, 0)
		return errors.New("rpc call timeout")
	}
}
