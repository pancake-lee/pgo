package pmq

import (
	"context"
	"errors"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// 声明一个消费者处理函数，这是多个封装中，功能最齐全的注册方法：
// 1：exchange为空则定义队列direct绑定到默认交换机
// 2：newWorker为true则启用专用单线程处理队列中的请求，否则使用公共的worker
// 3：callBack为处理函数，支持的形式见ParseMsgAndExecRpcCallBack的备注
func (cli *AmqpClient) DeclareConsumeFunc(
	exchange string, exchangeType string,
	route string, queue string,
	newWorker bool,
	cb func(ctx context.Context) error,
) error {

	if cli.queuePrefix != "" { //可以通过配置追加前缀，提供一定的环境隔离，先用于云端mq接口分派
		route = cli.queuePrefix + "_" + route
		queue = cli.queuePrefix + "_" + queue
	}

	autoDelete := false
	if cli.queueSuffix != "" { //可以通过配置追加后缀，提供一定的环境隔离，debug时可以用
		route += "_" + cli.queueSuffix
		queue += "_" + cli.queueSuffix
		autoDelete = true
	}

	err := cli.declareAndConsumeQueue(queue, autoDelete, newWorker, cb)
	if err != nil {
		return plogger.LogErr(err)
	}

	plogger.Debug("rpc queue register success [", queue, "]")

	if exchange != "" {
		err = cli.bindQueueToExchange(exchange, exchangeType, route, queue)
		if err != nil {
			return plogger.LogErr(err)
		}
	}

	return nil
}

// --------------------------------------------------
// 以下算是语法糖，简化/统一/规范了几种常见的注册方式
// --------------------------------------------------

// RPC：用函数名作为队列名，direct绑定到默认交换机
func (cli *AmqpClient) DeclareSimpleRpc(cb func(ctx context.Context) error) error {
	return cli.DeclareConsumeFunc("", "", "",
		putil.GetFuncName(cb),
		false, cb)
}

// 简单事件，函数名和事件结构体名相同，topic绑定到“默认事件交换机”DefaultNotifyExchange
func (cli *AmqpClient) DeclareSimpleEvent(cb func(ctx context.Context) error) error {
	return cli.DeclareServerEvent(cli.defaultEventExchange,
		putil.GetFuncName(cb), false, cb)
}
func (cli *AmqpClient) DeclareSimpleEventPrivateWorker(cb func(ctx context.Context) error) error {
	return cli.DeclareServerEvent(cli.defaultEventExchange,
		putil.GetFuncName(cb), true, cb)
}

// 内部事件：队列名为执行文件名+函数名，保证多个服务可以同时监听同一个事件
func (cli *AmqpClient) DeclareServerEvent(
	exchange, route string, newWorker bool, cb func(ctx context.Context) error,
) error {
	queue := putil.GetExecName() + "_" + putil.GetFuncName(cb)
	return cli.DeclareConsumeFunc(exchange, "topic", route, queue, newWorker, cb)
}

// --------------------------------------------------
func (cli *AmqpClient) GetQueueLen(rpcName string) (int, error) {
	ch, err := cli.conn.Channel()
	if err != nil {
		return 0, plogger.LogErr(err)
	}
	defer ch.Close()

	q, err := ch.QueueInspect(rpcName)
	if err != nil {
		return 0, plogger.LogErr(err)
	}
	return q.Messages, nil
}

func (cli *AmqpClient) PauseConsumption(rpcName string) error {
	controlChan, ok := cli.controlChanMap[rpcName]
	if !ok {
		return errors.New("queue not found: " + rpcName)
	}
	controlChan <- true
	return nil
}

func (cli *AmqpClient) ResumeConsumption(rpcName string) error {
	controlChan, ok := cli.controlChanMap[rpcName]
	if !ok {
		return errors.New("queue not found: " + rpcName)
	}
	controlChan <- false
	return nil
}
