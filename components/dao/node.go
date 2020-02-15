package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	. "go.7yes.com/j7f/components/dao/errors"
	"go.7yes.com/j7f/components/dao/shard"
	"strconv"
	"sync"
	"time"
	"net/url"
)

const TRANSACTION_MAX_RUNTIME = time.Second * 1

type checkHandler func(err error)

type DBConfig struct {
	Name         string
	MaxConnNum   int
	MaxIdleConns int
	MaxLifetime	int

	Master *NodeConfig
	Slave  []*NodeConfig

	Shard []*shard.ShardConfig
}

type NodeConfig struct {
	Addr     string
	User     string
	Password string
	Timezone string
	Weight   int
}

type Node struct {
	l *sync.RWMutex

	Cfg *DBConfig

	Master *sql.DB

	Slave          []*sql.DB
	LastSlaveIndex int
	RoundRobinQ    []int
	SlaveWeights   []int

	Shard   []shard.Shard
	shardDb []string
}

type transactionKey struct{}
type transactionCancelKey struct{}

func NewNode(cfg *DBConfig, c checkHandler) (*Node, error) {
	if len(cfg.Master.Addr) == 0 {
		return nil, ErrNoMasterDB
	}
	l := new(sync.RWMutex)
	shardDb := make([]string, 0)
	for _, v := range cfg.Shard {
		shardDb = append(shardDb, v.DB)
	}
	shards, err := shard.ParseShard(cfg.Shard)
	if err != nil {
		return nil, err
	}
	n := &Node{
		l:       l,
		Cfg:     cfg,
		shardDb: shardDb,
		Shard:   shards,
	}

	err = n.parseMaster()
	if err != nil {
		return nil, err
	}

	err = n.parseSlave()
	if err != nil {
		return nil, err
	}

	n.InitBalancer()

	go n.CheckNode(c)

	return n, nil
}

func (n *Node) parseMaster() (err error) {
	n.Master, err = n.openDB(n.Cfg.Master)

	return err
}

func (n *Node) parseSlave() (err error) {
	var db *sql.DB

	count := len(n.Cfg.Slave)
	if count == 0 {
		return nil
	}

	n.Slave = make([]*sql.DB, 0, count)
	n.SlaveWeights = make([]int, 0, count)
	//parse addr and weight
	for _, slave := range n.Cfg.Slave {
		n.SlaveWeights = append(n.SlaveWeights, slave.Weight)
		if db, err = n.openDB(slave); err != nil {
			return err
		}
		n.Slave = append(n.Slave, db)
	}

	return
}

// check the node alive
func (n *Node) CheckNode(checkHandler checkHandler) {
	t := time.NewTicker(30 * time.Second)

	for {
		<-t.C
		n.checkMaster(checkHandler)
		n.checkSlave(checkHandler)
	}
}

func (n *Node) checkMaster(checkHandler checkHandler) {
	db := n.Master
	if db == nil {
		checkHandler(errors.New("Node checkMaster  Master is not online"))
		return
	}

	if err := db.Ping(); err != nil {
		checkHandler(errors.New("Node checkMaster  ping error " + err.Error()))
		return
	}
	return
}

func (n *Node) checkSlave(checkHandler checkHandler) {
	n.l.RLock()
	if n.Slave == nil {
		n.l.RUnlock()
		return
	}
	slaves := make([]*sql.DB, len(n.Slave))
	copy(slaves, n.Slave)
	n.l.RUnlock()

	for i := 0; i < len(slaves); i++ {
		if err := slaves[i].Ping(); err != nil {
			checkHandler(errors.New("Node checkSlave[" + strconv.Itoa(i) + "]  ping error " + err.Error()))
		}
	}

}

func (n *Node) openDB(dsn *NodeConfig) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8&loc=%s&parseTime=true", dsn.User, dsn.Password, dsn.Addr,url.QueryEscape(dsn.Timezone)))
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(n.Cfg.MaxIdleConns)
	db.SetMaxOpenConns(n.Cfg.MaxConnNum)
	db.SetConnMaxLifetime( time.Duration(n.Cfg.MaxLifetime) * time.Second )
	err = db.Ping()

	return db, err
}

func (n *Node) GetMasterConn() (*sql.DB, error) {
	db := n.Master
	if db == nil {
		return nil, ErrNoMasterConn
	}

	return db, nil
}

func (n *Node) GetSlaveConn() (*sql.DB, error) {
	n.l.Lock()
	db, err := n.getNextSlave()
	n.l.Unlock()
	if err != nil {
		return nil, err
	}

	if db == nil {
		return nil, ErrNoSlaveDB
	}

	return db, nil
}

func (n *Node) GetTable(db, table string, key ...interface{}) (string, error) {
	if shardModel, ok := n.checkShard(db, table); ok {
		k, err := shardModel.FindForKey(key...)
		if err != nil {
			return "", err
		}
		return db + "." + table + "_" + strconv.Itoa(k), nil
	}
	return db + "." + table, nil
}

func (n *Node) checkShard(db, table string) (shard.Shard, bool) {
	for _, shardDb := range n.shardDb {
		if shardDb == db {
			for k, v := range n.Cfg.Shard {
				if v.DB == db && v.Table == table {
					return n.Shard[k], true
				}
			}
		}
	}
	return nil, false
}

func (n *Node) BeginTransaction(ctx context.Context, maxRuntime time.Duration) (context.Context, error) {
	var cancel context.CancelFunc
	c, err := n.GetMasterConn()
	if err != nil {
		return nil, err
	}

	if maxRuntime == 0 || maxRuntime > TRANSACTION_MAX_RUNTIME {
		maxRuntime = TRANSACTION_MAX_RUNTIME
	}

	ctx, cancel = context.WithTimeout(ctx, maxRuntime)

	tx, err := c.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, transactionKey{}, tx)
	ctx = context.WithValue(ctx, transactionCancelKey{}, cancel)
	return ctx, nil
}

func (n *Node) Commit(ctx context.Context) error {
	tx, err := n.GetConnFromCtx(ctx)
	if err != nil {
		return err
	}

	c := ctx.Value(transactionCancelKey{})
	cancel, ok := c.(context.CancelFunc)
	if !ok {
		defer cancel()
	}

	return tx.Commit()
}

func (n *Node) Rollback(ctx context.Context) error {
	tx, err := n.GetConnFromCtx(ctx)
	if err != nil {
		return err
	}

	c := ctx.Value(transactionCancelKey{})
	cancel, ok := c.(context.CancelFunc)
	if !ok {
		defer cancel()
	}

	return tx.Rollback()
}

func (n *Node) GetConnFromCtx(ctx context.Context) (*sql.Tx, error) {
	t := ctx.Value(transactionKey{})
	tx, ok := t.(*sql.Tx)
	if !ok {
		return nil, errors.New("assert tx err")
	}
	return tx, nil
}
