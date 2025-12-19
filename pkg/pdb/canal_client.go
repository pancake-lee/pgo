package pdb

import (
	"context"
	"strings"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/predis"
	"github.com/pancake-lee/pgo/pkg/putil"
)

const REDIS_KEY_CANAL_BINLOG_POS = "canal:binlog:pos"

type CanalClient struct {
	c      *canal.Canal
	dbName string
}

type MyEventHandler struct {
	canal.DummyEventHandler
	dbName string
}

func (h *MyEventHandler) SetDBName(db string) {
	h.dbName = db
}

func (h *MyEventHandler) OnPosSynced(header *replication.EventHeader,
	pos mysql.Position, set mysql.GTIDSet, force bool) error {

	plogger.Debugf("update binlog pos[%v] to redis", pos)

	err := predis.DefaultClient.Set(
		REDIS_KEY_CANAL_BINLOG_POS,
		pos.Name+":"+putil.Uint32ToStr(pos.Pos),
		0).
		Err()
	if err != nil {
		plogger.Error(err)
	}

	return nil
}

func (h *MyEventHandler) OnRow(e *canal.RowsEvent) error {
	if e.Table.Schema != h.dbName {
		return nil
	}

	cb, ok := GetCallback(e.Table.Name)
	if !ok {
		return nil
	}

	ctx := context.Background()
	cb(ctx, e)

	return nil
}

func (h *MyEventHandler) String() string {
	return "MyEventHandler"
}

func NewCanal(dbConf MysqlConfig) (*CanalClient, error) {
	cfg := canal.NewDefaultConfig()
	cfg.Addr = dbConf.Mysql.Addr
	cfg.User = dbConf.Mysql.User
	cfg.Password = dbConf.Mysql.Password

	cfg.Dump.TableDB = dbConf.Mysql.DbName
	cfg.Dump.Tables = GetAllTables()

	cfg.Logger = newCanalLogger()
	c, err := canal.NewCanal(cfg)
	if err != nil {
		return nil, err
	}

	var myHandler MyEventHandler
	myHandler.SetDBName(dbConf.Mysql.DbName)
	c.SetEventHandler(&myHandler)

	return &CanalClient{
		c:      c,
		dbName: dbConf.Mysql.DbName,
	}, nil
}

func (client *CanalClient) Run() error {
	redisPos, err := predis.DefaultClient.Get(REDIS_KEY_CANAL_BINLOG_POS).Result()
	if err != nil || redisPos == "" {
		plogger.Warn("redis binlog pos not found, and no DB fallback implemented")
	}

	plogger.Debug("redisPos get current pos : ", redisPos)

	if redisPos == "" {
		panic("redis pos is nil")
	}

	var pos mysql.Position

	if redisPos == "0" {
		return client.c.Run()
	}

	if redisPos == "cur" {
		curPos, err := client.c.GetMasterPos()
		if err != nil {
			return err
		}
		pos = curPos
		plogger.Debug("curPos : ", pos)

	} else {
		posStrList := strings.Split(redisPos, ":")
		if len(posStrList) != 2 {
			plogger.Error("invalid binlog pos : ", redisPos)
			return nil // Or return error
		}

		pos.Name = posStrList[0]
		pos.Pos, err = (putil.StrToUint32(posStrList[1]))
		if err != nil {
			return err
		}
		plogger.Debug("curPos : ", pos)
	}

	return client.c.RunFrom(pos)
}
