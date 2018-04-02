package core

//package main
import (
	"sync"
	"utility"

	"github.com/garyburd/redigo/redis"
)

type redisCmdHandler func()

const (
	redisConnStateNotConneted = 1
	redisConnStateConnected   = 2
)

//DBRedis redis struct
type DBRedis struct {
	conn        redis.Conn
	dbID        int
	lock        *sync.Mutex
	log         *utility.LogContext
	handlerChan chan redisCmdHandler
	waitGroup   *sync.WaitGroup
	exitChan    chan int
	cmdChan     chan *DBCmd
	state       int
}

type DBMsg struct {
}

type DBResult struct {
	reply interface{}
	err   error
}

type DBCmd struct {
	cmd        string
	value      []interface{}
	resultChan chan *DBResult
}

const constUIDCounter = "UIDCounter"

//NewRedisClient return a new redis instance
func NewRedisClient(dbID int, addr string, log *utility.LogModule) *DBRedis {
	l := utility.NewLogContext(0, log)
	conn, err := redis.Dial("tcp", addr)
	if err != nil {
		l.LogFatal("Failed to connect redis in this node:%s", addr)
		return nil
	}
	if _, err := conn.Do("select", dbID); err != nil {
		l.LogFatal("Failed to select db on redis. %v", err)
		return nil
	}
	db := &DBRedis{
		conn:        conn,
		dbID:        dbID,
		lock:        new(sync.Mutex),
		handlerChan: make(chan redisCmdHandler, 1024),
		log:         l,
		exitChan:    make(chan int),
		cmdChan:     make(chan *DBCmd, 512),
	}

	//go db.handleCmd()
	l.LogInfo("init redis connection successfully. addr:%s", addr)
	return db
}

//Init initialize redis instance
func (db *DBRedis) Init(waitGroup *sync.WaitGroup) {
	db.waitGroup = waitGroup
	waitGroup.Add(1)
	db.conn.Do("INCRBY", constUIDCounter, 20)
	go db.handleCmd()
}

// Close close the redis connection
func (db *DBRedis) Close() {
	db.conn.Close()
	db.exitChan <- 1
}

//GetCounter get one counter which increases only
func (db *DBRedis) GetCounter() (uint64, error) {
	return db.Incr(constUIDCounter)
}

//GetResult yeh, get result
func (db *DBRedis) GetResult(cmd string, args ...interface{}) *DBResult {
	dbCmd := new(DBCmd)
	dbCmd.resultChan = make(chan *DBResult, 1)
	dbCmd.cmd = cmd
	dbCmd.value = args
	db.cmdChan <- dbCmd
	return <-dbCmd.resultChan
}

//Incr increment the value of key by 1
func (db *DBRedis) Incr(key string) (uint64, error) {
	inChan := make(chan *DBResult, 1)
	aFunc := func() {
		r := new(DBResult)
		r.reply, r.err = db.conn.Do("INCR", key)
		inChan <- r
	}
	db.handlerChan <- aFunc
	r := <-inChan
	if r.err != nil {
		return 0, r.err
	}
	return redis.Uint64(r.reply, r.err)
}

//Set(key, value string)(error){
//Whether this is efficient
func (db *DBRedis) Set(key, value interface{}) error {
	inChan := make(chan error)
	aFunc := func() {
		if _, err := db.conn.Do("SET", key, value); err != nil {
			db.log.LogWarn("Failed to save key:%s, value:%v err:%s", key, value, err.Error())
			inChan <- err
		}
		inChan <- nil
	}
	db.handlerChan <- aFunc
	return <-inChan
}

//SetWithTimeout the value bind to key 'key' will timeout after 'timeout' seconds
func (db *DBRedis) SetWithTimeout(key, value interface{}, timeout int) error {
	/*
		db.lock.Lock()
		defer db.lock.Unlock()
		if _, err := db.conn.Do("SETEX", key, timeout, value); err != nil {
			db.log.LogWarn("Failed to save key:%s, value:%v timeout:%d, err:%s", key, value, timeout, err.Error())
			return err
		}*/
	return nil
}

//Get Return silice of byte may be better, avoid convertion from string to byte
func (db *DBRedis) Get(key string) ([]byte, error) {
	inChan := make(chan *DBResult, 1)
	aFunc := func() {
		r := new(DBResult)
		r.reply, r.err = db.conn.Do("Get", key)
		inChan <- r
	}
	db.handlerChan <- aFunc
	r := <-inChan

	if r.err != nil {
		return nil, r.err
	}
	//avoid Redis.ErrNIL
	if r.reply == nil {
		return nil, nil
	}
	return redis.Bytes(r.reply, r.err)

}

func (db *DBRedis) ListAdd(k string, v interface{}) (int, error) {
	r := db.GetResult("ladd", k, v)
	return redis.Int(r.reply, r.err)
}

func (db *DBRedis) ListGetALL(k string) ([][]byte, error) {
	r := db.GetResult("lrange", k, 0, -1)
	return redis.ByteSlices(r.reply, r.err)
}

//HashAdd add one element into hash table
func (db *DBRedis) HashAdd(key string, field interface{}, v interface{}) (int, error) {
	r := db.GetResult("hset", key, field, v)
	return redis.Int(r.reply, r.err)
}

//HashRemove remove one element from hash table
func (db *DBRedis) HashRemove(key string, field interface{}) (int, error) {
	r := db.GetResult("hdel", key, field)
	return redis.Int(r.reply, r.err)
}

func (db *DBRedis) hashGetAll(key string) (map[string]string, error) {
	r := db.GetResult("hgetall", key)
	return redis.StringMap(r.reply, r.err)
}

//Del remove the given key
func (db *DBRedis) Del(key string) (int, error) {
	r := db.GetResult("del", key)
	return redis.Int(r.reply, r.err)
}

func (db *DBRedis) handleCmd() {
	utility.CatchPanic(db.log, nil)
	for {
		select {
		case handler := <-db.handlerChan:
			handler()
		case cmd := <-db.cmdChan:
			r := new(DBResult)
			r.reply, r.err = db.conn.Do(cmd.cmd, cmd.value...)
			cmd.resultChan <- r
		case <-db.exitChan:
			db.log.LogInfo("dbredis prepare to exit")
			db.waitGroup.Done()
			return
		}

	}
}

/*
 *return 1 and error is nil when sadd sucess
 *return 0 and error is nil when the member already exist in set
 */
/*
func (db *DBRedis) db_set_add(key string, v interface{}) (int, error) {
	return db.exec_return_int("sadd", key, v)
}

func (db *DBRedis) db_set_remove(key string, v interface{}) (int, error) {
	return db.exec_return_int("srem", key, v)
}

func (db *DBRedis) db_set_search_member(key string, member string, cursor int) ([]string, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	members := make([]string, 0)
REDO:
	rt, err := redis.Values(db.conn.Do("sscan", key, cursor, "match", member))
	if err != nil {
		g_logger.Log(LOG_WARN, "search member on key:%s member:%s cursor:%d err:%s", key, member, cursor, err.Error())
		return nil, err
	}
	for _, e := range rt {
		switch v := e.(type) {
		case int:
		case string:
		case [][]string:
		case []byte:
			g_logger.Log(LOG_DBG, "bytes:%s\n", string(v))
			newCursor, err := atoi(string(v), 32)
			cursor = int(newCursor)
			if err != nil {
				g_logger.Log(LOG_WARN, "Failed to parse cursor:%s err:%v", string(v), err)
				return nil, err
			}
		case [][]byte:
		case []interface{}:
			for _, sub_ele := range v {
				switch v1 := sub_ele.(type) {
				case []string:
				case interface{}:
					tmp, _ := redis.String(v1, nil)
					members = append(members, tmp)
				case []byte:
				default:
				}
			}
		default:
		}
	}

	if cursor != 0 {
		goto REDO
	}

	return members, nil

}


func (db *DBRedis) db_set_get(key string) ([]string, error) {
	return db.exec_return_strings("smembers", key)
}

func (db *DBRedis) db_del_set(key string) (int, error) {
	return db.db_del_key(key)
}

func (db *DBRedis) db_del_list(key string) (int, error) {
	return db.db_del_key(key)
}

func (db *DBRedis) db_del_hash(key string) (int, error) {
	return db.db_del_key(key)
}
func (db *DBRedis) db_del_key(key string) (int, error) {
	return db.exec_return_int("del", key)
}

func (db *DBRedis) db_list_add(k string, v interface{}) (int, error) {
	return db.exec_return_int("rpush", k, v)
}

func (db *DBRedis) db_list_remove(k string) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, err := redis.String(db.conn.Do("rpop", k))
	if err != nil {
		g_logger.Log(LOG_WARN, "failed remove element from list:%s err:%s", k, err.Error())
	}
	return err
}

func (db *DBRedis) db_list_trim(k string, limit, num int) {
	start_idx := num - limit
	if start_idx <= 0 {
		return
	}
	db.conn.Do("ltrim", k, start_idx, -1)
}

func (db *DBRedis) db_list_get_all(k string) ([]string, error) {
	return db.exec_return_strings("lrange", k, 0, -1)
}


func (db *DBRedis) get_friend_data(v ...interface{}) []string {
	rt, _ := db.exec_return_strings("mget", v...)
	return rt
}
func (db *DBRedis) key_search(key string) []string {
	rt, _ := db.exec_return_strings("keys", key)
	return rt
}

func (db *DBRedis) del_friend(from, to string, from_id, to_id interface{}) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.conn.Send("MULTI")
	db.conn.Send("srem", from, to_id)
	db.conn.Send("srem", to, from_id)
	_, err := db.conn.Do("exec")
	return err
}

func (db *DBRedis) add_friend(from, to string, limit int, from_id interface{}, to_id interface{}) (int, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	script := redis.NewScript(2, "if(redis.call('scard', KEYS[1]) >= tonumber(ARGV[1])) then return 1 end if(redis.call('scard', KEYS[2]) >= tonumber(ARGV[1]))then return 2 end redis.call('sadd', KEYS[1], ARGV[2]) redis.call('sadd', KEYS[2], ARGV[3]) return 0")
	r, err := redis.Int(script.Do(db.conn, from, to, limit, to_id, from_id))
	return r, err
}

func (db *DBRedis) zset_add(key string, score int, member interface{}) (int, error) {
	return db.exec_return_int("zadd", key, score, member)
}

func (db *DBRedis) zset_remove(key string, member interface{}) (int, error) {
	return db.exec_return_int("zrem", key, member)
}

func (db *DBRedis) zset_search(key string, min, max, offset, count int) ([]string, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	rt, err := redis.Strings(db.conn.Do("zrangebyscore", key, min, max, "withscores", "limit", offset, count))
	if err != nil {
		g_logger.Log(LOG_WARN, "Failed to got members on key:%s in range:{%d, %d} offset:%d count:%d err:%v", key, min, max, offset, count, err)
		return nil, err
	}

	return rt, nil
}
*/
/*

func main(){

	db, _ := DB_init(0)

	//rt, _ := db.db_set_add("myset", 100)
	//rt, _ := db.db_set_remove("myset", 1)
	rt, _ := db.db_list_add("mylist", "hello")
	g_logger.Log(LOG_WARN,"return value:%d\n", rt)
	rt, _ = db.db_list_add("mylist", "hz")
	g_logger.Log(LOG_WARN,"return value:%d\n", rt)
	vs, _ := db.db_list_get_all("mylist")
	g_logger.Log(LOG_WARN,"return value:%v\n", vs)

	db.Close()
}
*/
