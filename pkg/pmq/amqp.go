package pmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	// "github.com/streadway/amqp"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	amqp "github.com/rabbitmq/amqp091-go"
)

var DefaultClient *AmqpClient

func MustInitMQByConfig() {
	err := InitMQByConfig()
	if err != nil {
		panic(err)
	}
}
func InitMQByConfig() error {
	addr := pconfig.GetStringM("RabbitMQ.Addr")

	strList := putil.StrToStrList(addr, ":")
	if len(strList) < 2 {
		return fmt.Errorf("invalid mq addr: %v", addr)
	}
	host, port := strList[0], strList[1]

	userName := pconfig.GetStringM("RabbitMQ.User")
	password := pconfig.GetStringM("RabbitMQ.Password")
	vHost := pconfig.GetStringM("RabbitMQ.VHost")

	var err error
	DefaultClient, err = NewMQ(host, port, userName, password, vHost)
	if err != nil {
		plogger.Error("Failed to connect to RabbitMQ")
		return err
	}

	DefaultClient.isLogMessage = pconfig.GetInt32D("RabbitMQ.IsLogMessage", 0)
	DefaultClient.rpcTimeOut = pconfig.GetInt32D("RabbitMQ.RpcTimeOut", 10)
	DefaultClient.queuePrefix = pconfig.GetStringD("RabbitMQ.QueuePrefix", "")
	DefaultClient.queueSuffix = pconfig.GetStringD("RabbitMQ.QueueSuffix", "")
	DefaultClient.defaultEventExchange = pconfig.GetStringD("RabbitMQ.DefaultEventExchange", "")

	return nil
}

// --------------------------------------------------
// 以下是可复用的mq对象封装

type AmqpClient struct {
	host string //保留host作为日志输出，其他参数都拼到url了
	url  string

	isLogMessage         int32
	rpcTimeOut           int32
	queuePrefix          string
	queueSuffix          string
	defaultEventExchange string // 默认事件交换机

	conn *amqp.Connection
	ch   *consumeChannel

	workerWaitGroup       sync.WaitGroup // 公共worker的waitGroup
	mqConsumerChan        chan mqMsg     // 公共worker的消息队列
	mqConsumerPriChanList []chan mqMsg   // 独立worker的消息队列

	beforeCall func(ctx context.Context) error

	controlChanMap map[string]chan bool
}

func NewMQ(host, port, userName, password, vhost string) (*AmqpClient, error) {
	var cli AmqpClient

	cli.host = host
	cli.url = fmt.Sprintf("amqp://%s:%s@%s:%s/%s", userName, password, host, port, vhost)
	cli.controlChanMap = make(map[string]chan bool)

	cli.initPublicWorker()

	err := cli.connect()
	if err != nil {
		plogger.Error("Failed to set QoS")
		return nil, err
	}

	go cli.waitReconnect()

	return &cli, nil
}

func (cli *AmqpClient) GetDefaultEventExchange() string {
	return cli.defaultEventExchange
}

func (cli *AmqpClient) Wait() {
	cli.workerWaitGroup.Wait()
}

func (cli *AmqpClient) Close() {
	// conn的关闭会让所有channel关闭
	cli.conn.Close()
	// 关闭mq和worker之间的chan，最后导致worker关闭，进而workerWaitGroupWait()结束
	cli.closeWorker()
}

func (cli *AmqpClient) connect() (err error) {
	cli.conn, err = amqp.Dial(cli.url)
	if err != nil {
		plogger.Errorf("mq[%v] Failed to connect to RabbitMQ", cli.host)
		return err
	}

	// 主要的操作channel
	cli.ch, err = cli.channel()
	if err != nil {
		plogger.Errorf("mq[%v] err : %v", cli.host, err)
		return err
	}

	//每次mq获取多少条消息，一般摸块多进程运行消费同一条队列时设置为1，则每个消费者轮询消费
	//global为true表示这个设置是针对整个conn的，后续创建的channel都会继承这个设置
	err = cli.ch.mqChannel.Qos(1, 0, true)
	if err != nil {
		plogger.Error("Failed to set QoS")
		return err
	}
	return nil
}

func (cli *AmqpClient) waitReconnect() {
	// NotifyClose的实现是，由我们传递了一个chan进去
	// 如果是意外断连，则通过该chan通知我们
	// 只有主动conn.Close时，由内部帮我们close这个chan
	for {
		reason, ok := <-cli.conn.NotifyClose(make(chan *amqp.Error))
		if !ok { // 主动关闭，[优雅退出程序]
			plogger.Debug("conn closed, notify chan closed, exit reconnect goroutine")
			break
		}
		plogger.Debugf("conn closed, reason: %v", reason)

		for {
			time.Sleep(1 * time.Second)

			err := cli.connect()
			if err == nil { // 重新订阅新连接的NotifyClose
				plogger.Debug("reconnect success")
				break
			}

			plogger.Errorf("reconnect failed, err: %v", err)
		}
	}
}

// 把队列绑定到指定交换机
func (cli *AmqpClient) bindQueueToExchange(
	exchangeName, exchangeType, routeKey, queue string) error {

	plogger.Debugf("mq[%v] exchange[%v][%v] routeKey[%v] queue[%v]", cli.host,
		exchangeName, exchangeType, routeKey, queue)

	err := cli.ch.mqChannel.ExchangeDeclare(exchangeName, exchangeType,
		true, false, false, false, nil)
	if err != nil {
		return plogger.LogErr(err)
	}
	err = cli.ch.mqChannel.QueueBind(queue, routeKey, exchangeName, false, nil)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

// 声明并消费队列，将自动绑定到默认交换机，则exchange="",type="direct",routeKey=queueName
func (cli *AmqpClient) declareAndConsumeQueue(
	queue string, autoDelete bool, newWorker bool,
	cb func(ctx context.Context) error) error {

	// 每条队列独立一个channel做消费者
	ch, err := cli.channel()
	if err != nil {
		return plogger.LogErr(err)
	}

	_, err = ch.mqChannel.QueueDeclare(
		queue,      // name
		true,       // durable //持久的，mq重启，队列恢复
		autoDelete, // delete when unused //自动删除的，即使没有消费者，也需要维持这个队列存活
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return plogger.LogErr(err)
	}

	// 下面Consume和Delivery分开两个协程了，
	// 是因为当mq重连时，Consume会重新构建协程，
	// 但不想重建Delivery以及后面的worker协程。

	dChan, ctlChan := ch.asyncConsumeQueue(queue)
	cli.controlChanMap[queue] = ctlChan

	cli.handleDelivery(dChan, cb, queue, newWorker)

	return nil
}

func (cli *AmqpClient) channel() (*consumeChannel, error) {

	mqChannel, err := cli.conn.Channel()
	if err != nil {
		return nil, err
	}

	var ch consumeChannel
	ch.mqChannel = mqChannel

	go ch.waitReconnect(cli)

	return &ch, nil
}

// --------------------------------------------------
type consumeChannel struct {
	mqChannel *amqp.Channel
}

func (ch *consumeChannel) waitReconnect(cli *AmqpClient) {
	for {
		reason, ok := <-ch.mqChannel.NotifyClose(make(chan *amqp.Error))
		if !ok { // 主动关闭，[优雅退出程序]
			plogger.Debug("channel closed, notify chan closed, exit reconnect goroutine")
			break
		}
		plogger.Debug("channel closed, reason: %v", reason)

		for {
			time.Sleep(1 * time.Second)

			mqChannel, err := cli.conn.Channel()
			if err == nil {
				plogger.Debug("channel recreate success")
				ch.mqChannel = mqChannel
				break
			}

			plogger.Debug("channel recreate failed, err: %v", err)
		}
	}
}

func (ch *consumeChannel) asyncConsumeQueue(queue string,
) (dChan chan amqp.Delivery, ctlChan chan bool) {
	// 因为重连会导致重新消费时拿到新的chan，所以要定义一个固定的chan给外部使用
	// 这个方法最终隔离了mq连接和worker队列之间的关系
	// conn和channel都有各自的重连机制
	// 但是这里定义了独立的deliveries队列，而不是Consume方法返回的chan
	// 我们只是在mq重连时，重新把消息从mq中消费到这个deliveries队列中，再给worker处理
	// 所以deliveries队列是[有且只有]我们自己主动关闭，则[优雅退出程序]时

	dChan = make(chan amqp.Delivery)

	ctlChan = make(chan bool) //用于控制暂停消费消息

	go func() {
		retryCnt := 0
		for {
			if ch.mqChannel.IsClosed() {
				time.Sleep(1 * time.Second)
				retryCnt++
				plogger.Debugf("channel closed, retry[%v] to consume[%v]", retryCnt, queue)
				continue
			}

			msgChan, err := ch.mqChannel.Consume(
				queue, // queue
				"",    // consumer
				false, // auto-ack, true自动应答，mq不再能存储下消息，所有消息会立刻被消费，获取到了进程中缓冲
				false, // exclusive
				false, // no-local
				false, // no-wait
				nil,   // args
			)
			if err != nil {
				time.Sleep(1 * time.Second)
				retryCnt++
				plogger.Debugf("consume failed, retry[%v] to consume[%v]", retryCnt, queue)
				continue
			}

			// for d := range msgChan {
			// 	deliveries <- d
			// }

			for {
				exitConsumer := false
				select {
				case b := <-ctlChan:
					if b { //一次true进入暂停
						for {
							b, ok := <-ctlChan
							if !ok {
								break
							}
							if !b {
								break //直到false才退出暂停
							}
						}
					}

				case d, ok := <-msgChan:
					if !ok {
						exitConsumer = true
						plogger.Debugf("msgChan exit [%v]", queue)
						break //mqChannel关闭的时候，msgs会关闭，这里就会break，break select
					}
					// plogger.Debugf("msgChan get msg [%v]", queue)
					dChan <- d
				}

				if exitConsumer {
					break // break for
				}
			}
		}
	}()

	return dChan, ctlChan
}
