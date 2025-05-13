package instance

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/consumer/command"
	"lightiot/pkg/consumer/common"
	"lightiot/pkg/consumer/data"
	"lightiot/pkg/consumer/event"
	"lightiot/pkg/consumer/job"
	"lightiot/pkg/consumer/thing"
	"sync/atomic"
	"time"
)

type consumerInfo struct {
	IDs        []uint16 `json:"is"`
	Name       string   `json:"n"`
	RecentTime int64    `json:"rt"`
}

type consumers []*consumerInfo

func (cs consumers) findByName(name string) int {
	for i := range cs {
		if cs[i].Name == name {
			return i
		}
	}
	return -1
}

type Consumer struct {
	*Config
	stopCh  <-chan struct{}
	jobCnts []int32

	registry *registry
	jm       *job.Manager
	chs      map[job.Group]chan *job.Job

	dataWorker     *data.Worker
	deleteHandlers map[job.OperationType]common.DeleteHandler
}

func New(stopCh <-chan struct{}, config *Config, jobGroups []job.Group) *Consumer {
	ret := &Consumer{
		Config:         config,
		stopCh:         stopCh,
		jobCnts:        make([]int32, config.MaxInstances),
		chs:            make(map[job.Group]chan *job.Job, len(jobGroups)),
		deleteHandlers: make(map[job.OperationType]common.DeleteHandler),
	}

	for _, jg := range jobGroups {
		switch jg {
		case job.GroupRollup:
			ret.chs[jg] = make(chan *job.Job, config.RollupBufferSize)
		case job.GroupDelete:
			ret.chs[jg] = make(chan *job.Job, config.DeleteBufferSize)
		}
	}
	return ret
}

func (c *Consumer) init() {
	c.registry = newRegistry(c.stopCh, c.HeartbeatInterval, c.MaxInstances, c.LogStorage)
	c.jm = job.New(c.stopCh, c.MaxInstances, 0, false, 0, c.LogStorage)

	c.dataWorker = data.NewWorker(c.stopCh, &c.DataWorkerConfig, c.LogStorage)
	c.dataWorker.Init()
	c.deleteHandlers[job.OpEventTypeDelete] = event.NewHandler(c.jm, c.LogStorage)
	c.deleteHandlers[job.OpCommandTypeDelete] = command.NewHandler(c.jm, c.LogStorage)
	c.deleteHandlers[job.OpTimeSeriesDelete] = thing.NewHandler(c.jm, c.LogStorage)
	c.deleteHandlers[job.OpThingTimeSeriesDelete] = data.NewHandler(c.jm, c.LogStorage)
	thingRelationHandler := thing.NewRelationHandler(c.jm, c.LogStorage)
	c.deleteHandlers[job.OpThingCommandDelete] = thingRelationHandler
	c.deleteHandlers[job.OpThingActionDelete] = thingRelationHandler
	c.deleteHandlers[job.OpThingEventDelete] = thingRelationHandler
}

func (c *Consumer) Start() {
	c.init()
	go func() {
		for {
			select {
			case <-c.stopCh:
				c.LogStorage.Close()
				klog.V(2).InfoS("Stopped consumer")
				return
			default:
			}

			ids := c.registry.allocateIDs(c.getLockedIDs())

			var cnt int
			for k, v := range c.chs {
				cnt += c.jm.RetrieveJobs(ids, v, k)
			}

			if cnt == 0 {
				// it is difficult to choose sleep or timer. For our case, sleep is better, although code is ugly (not golang style).
				// https://go101.org/article/unofficial-faq.html#time-sleep-after
				// https://groups.google.com/g/golang-nuts/c/9BL6v7Nqj_I
				// https://github.com/golang/go/issues/27707
				time.Sleep(time.Duration(c.RetrieveJobInterval) * time.Second)
			}
		}
	}()

	for i := uint16(0); i < c.RollupWorkers; i++ {
		go c.rollup()
	}

	for i := uint16(0); i < c.DeleteWorkers; i++ {
		go c.delete()
	}
}

func (c *Consumer) rollup() {
	ch := c.chs[job.GroupRollup]
	for {
		select {
		case j, ok := <-ch:
			if !ok {
				klog.InfoS("There is no event")
				// TODO recover
				return
			}
			atomic.AddInt32(&c.jobCnts[j.ConsumerID], 1)

			klog.V(5).InfoS("Received rollup", "job", j)

			c.dataWorker.Rollup(j)

			c.jm.DeleteJob(j)

			atomic.AddInt32(&c.jobCnts[j.ConsumerID], -1)
		case <-c.stopCh:
			klog.V(2).InfoS("Stopped consume rollup job")
		}
	}
}

func (c *Consumer) delete() {
	ch := c.chs[job.GroupDelete]
	for {
		select {
		case j, ok := <-ch:
			if !ok {
				klog.InfoS("There is no event")
				// TODO recover
				return
			}
			atomic.AddInt32(&c.jobCnts[j.ConsumerID], 1)

			klog.V(5).InfoS("Received delete", "job", j)
			if err := c.executeDeletionJob(j); err == nil {
				c.jm.DeleteJob(j)
			}

			atomic.AddInt32(&c.jobCnts[j.ConsumerID], -1)
		case <-c.stopCh:
			klog.V(2).InfoS("Stopped consume delete job")
		}
	}
}

func (c *Consumer) getLockedIDs() sets.Int32 {
	lockedIDs := sets.Int32{}
	for i, cnt := range c.jobCnts {
		if cnt > 0 {
			lockedIDs.Insert(int32(i))
		}
	}
	return lockedIDs
}

func (c *Consumer) executeDeletionJob(j *job.Job) error {
	if handler, exist := c.deleteHandlers[j.Operation]; exist {
		return handler.Delete(j)
	}
	klog.V(2).InfoS("Unsupported job operationType", "operation", j.Operation)
	return nil
}
