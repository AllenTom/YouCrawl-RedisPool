package redispool

import (
	"context"
	"fmt"
	"github.com/allentom/youcrawl"
	"github.com/go-redis/redis/v8"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

var ctx = context.Background()
var LogField = logrus.WithField("scope", "Redis-Pool")

type RedisPool struct {
	db             *redis.Client
	CloseFlag      int64
	Store          youcrawl.GlobalStore
	ItemSerializer RedisItemSerializer
	GetTaskChan    chan *youcrawl.Task
	PreventStop    bool
	DoneChan       chan struct{}
}

const (
	redisUnRequestedKey = "unrequested"
	redisUnCompletedKey = "uncompleted"
)

func NewRedisTaskPool(e *youcrawl.Engine, serializer RedisItemSerializer) *RedisPool {
	if serializer == nil {
		serializer = &DefaultItemSerializer{}
	}
	return &RedisPool{
		Store:          e.GlobalStore,
		ItemSerializer: serializer,
		DoneChan:       make(chan struct{}),
	}
}
func (p *RedisPool) InitRedis(redisOption *redis.Options) error {
	p.db = redis.NewClient(redisOption)
	pong, err := p.db.Ping(ctx).Result()
	if err != nil {
		return err
	}
	LogField.Info(fmt.Sprintf("Ping for redis database, result = %s", pong))
	return nil
}
func (p *RedisPool) AddURLs(urls ...string) {
	LogField.Info(fmt.Sprintf("append new url with len = %d", len(urls)))
	for _, url := range urls {
		id := xid.New().String()
		entity := TaskEntity{
			ID:   id,
			Url:  url,
			Item: youcrawl.DefaultItem{Store: map[string]interface{}{}},
		}
		serializedData, err := entity.Serialize()
		if err != nil {
			panic(err)
		}
		err = p.db.HSet(ctx, id, serializedData).Err()
		if err != nil {
			panic(err)
		}

		err = p.db.LPush(ctx, redisUnRequestedKey, id).Err()
		if err != nil {
			panic(err)
		}

		err = p.db.LPush(ctx, redisUnCompletedKey, id).Err()
		if err != nil {
			panic(err)
		}

	}

	// suspend task requirement exist,resume
	// see also `RequestPool.GetOneTask` method
	if p.GetTaskChan != nil && p.CloseFlag == 0 {
		resumeTask := p.GetUnRequestedTask()
		if resumeTask != nil {
			resumeTask.Requested = true
			p.GetTaskChan <- resumeTask
			p.GetTaskChan = nil
		}
	}
}

func (p *RedisPool) AddTasks(tasks ...*youcrawl.Task) {
	LogField.Info(fmt.Sprintf("append new url with len = %d", len(tasks)))
	for _, addTask := range tasks {

		id := xid.New().String()
		entity := TaskEntity{
			ID:  id,
			Url: addTask.Url,
		}

		item := addTask.Context.Item
		if item == nil {
			item = youcrawl.DefaultItem{
				Store: map[string]interface{}{},
			}
			entity.Item = map[string]interface{}{}
		} else {
			entity.Item = addTask.Context.Item
		}

		serializedData, err := entity.Serialize()
		if err != nil {
			panic(err)
		}
		err = p.db.HSet(ctx, id, serializedData).Err()
		if err != nil {
			panic(err)
		}

		err = p.db.LPush(ctx, redisUnRequestedKey, id).Err()
		if err != nil {
			panic(err)
		}

		err = p.db.LPush(ctx, redisUnCompletedKey, id).Err()
		if err != nil {
			panic(err)
		}
	}

	if p.GetTaskChan != nil && p.CloseFlag == 0 {
		resumeTask := p.GetUnRequestedTask()
		if resumeTask != nil {
			resumeTask.Requested = true
			p.GetTaskChan <- resumeTask
			p.GetTaskChan = nil
		}
	}
}

func (p *RedisPool) GetOneTask(e *youcrawl.Engine) <-chan *youcrawl.Task {
	taskChan := make(chan *youcrawl.Task)
	go func(callbackChan chan *youcrawl.Task) {
		if p.CloseFlag == 1 {
			return
		}
		unRequestedTask := p.GetUnRequestedTask()
		if unRequestedTask != nil {
			unRequestedTask.Requested = true
			callbackChan <- unRequestedTask
			return
		}

		// no more request,suspend task
		LogField.Info("suspend get task ")
		p.GetTaskChan = callbackChan

	}(taskChan)
	return taskChan
}

func (p *RedisPool) GetUnRequestedTask() (target *youcrawl.Task) {
	taskId := p.db.LPop(ctx, redisUnRequestedKey).Val()
	if len(taskId) == 0 {
		return nil
	}
	storeTask := TaskEntity{ID: taskId}
	err := storeTask.Load(p.db, p.ItemSerializer)
	if err != nil {
		LogField.Error(err)
	}

	return &youcrawl.Task{
		ID:  taskId,
		Url: storeTask.Url,
		Context: youcrawl.Context{
			Item:        storeTask.Item,
			GlobalStore: p.Store,
			Pool:        p,
		},
	}
}

func (p *RedisPool) OnTaskDone(task *youcrawl.Task) {
	// set true
	p.db.HSet(ctx, task.ID, "completed", true)
	p.db.LRem(ctx, redisUnCompletedKey, 1, task.ID)
	// check weather all task are complete
	if p.CloseFlag == 0 {
		LogField.Info(fmt.Sprintf("uncompleted len  = %d", p.db.LLen(ctx, redisUnCompletedKey).Val()))
		if p.db.LLen(ctx, redisUnCompletedKey).Val() != 0 || p.db.LLen(ctx, redisUnRequestedKey).Val() != 0 {
			return
		}
	}

	// no new request
	if !p.PreventStop {
		LogField.Info("no more task to resume , go to done!")
		p.DoneChan <- struct{}{}
	}
}

func (p *RedisPool) GetDoneChan() chan struct{} {
	return p.DoneChan
}

func (p *RedisPool) Close() {
	p.CloseFlag = 1
}

func (p *RedisPool) SetPrevent(isPrevent bool) {
	p.PreventStop = isPrevent

	if !isPrevent {
		if p.db.LLen(ctx, redisUnCompletedKey).Val() != 0 || p.db.LLen(ctx, redisUnRequestedKey).Val() != 0 {
			return
		}
		LogField.Info("no more task to resume , go to done!")
		p.DoneChan <- struct{}{}
	}
}
