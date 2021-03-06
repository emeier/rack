package router

import (
	"fmt"
	"sync"
	"time"
)

type StorageMemory struct {
	hostActivity   activityTracker
	targetActivity activityTracker
	idle           sync.Map
	routes         sync.Map

	targetLock sync.Mutex
}

func NewStorageMemory() *StorageMemory {
	fmt.Printf("ns=storage.memory at=new\n")

	return &StorageMemory{
		idle:   sync.Map{},
		routes: sync.Map{},
	}
}

func (b *StorageMemory) IdleGet(host string) (bool, error) {
	fmt.Printf("ns=storage.memory at=idle.get host=%q\n", host)

	v, ok := b.idle.Load(host)
	if !ok {
		return false, nil
	}

	i, ok := v.(bool)
	if !ok {
		return false, nil
	}

	return i, nil
}

func (b *StorageMemory) IdleSet(host string, idle bool) error {
	fmt.Printf("ns=storage.memory at=idle.get host=%q idle=%t\n", host, idle)

	b.idle.Store(host, idle)

	return nil
}

func (b *StorageMemory) RequestBegin(host string) error {
	fmt.Printf("ns=storage.memory at=request.begin host=%q\n", host)

	if err := b.hostActivity.Begin(host); err != nil {
		return err
	}

	ts, err := b.TargetList(host)
	if err != nil {
		return err
	}

	for _, t := range ts {
		if err := b.targetActivity.Begin(t); err != nil {
			return err
		}
	}

	return nil
}

func (b *StorageMemory) RequestEnd(host string) error {
	fmt.Printf("ns=storage.memory at=request.end host=%q\n", host)

	if err := b.hostActivity.End(host); err != nil {
		return err
	}

	ts, err := b.TargetList(host)
	if err != nil {
		return err
	}

	for _, t := range ts {
		if err := b.targetActivity.End(t); err != nil {
			return err
		}
	}

	return nil
}

func (b *StorageMemory) Stale(cutoff time.Time) ([]string, error) {
	fmt.Printf("ns=storage.memory at=stale cutoff=%s\n", cutoff)

	stale := []string{}

	b.routes.Range(func(k, v interface{}) bool {
		host, ok := k.(string)
		if !ok {
			return true
		}

		if a, err := b.hostActivity.ActiveSince(host, cutoff); err != nil || a {
			return true
		}

		ts, err := b.TargetList(host)
		if err != nil {
			return true
		}

		for _, t := range ts {
			if a, err := b.targetActivity.ActiveSince(t, cutoff); err != nil || a {
				return true
			}
		}

		stale = append(stale, host)

		return true
	})

	return stale, nil
}

func (b *StorageMemory) TargetAdd(host, target string) error {
	fmt.Printf("ns=storage.memory at=target.add host=%q target=%q\n", host, target)

	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	ts[target] = true

	b.routes.Store(host, ts)

	return nil
}

func (b *StorageMemory) TargetList(host string) ([]string, error) {
	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	targets := []string{}

	for t := range ts {
		targets = append(targets, t)
	}

	return targets, nil
}

func (b *StorageMemory) TargetRemove(host, target string) error {
	fmt.Printf("ns=storage.memory at=target.remove host=%q target=%q\n", host, target)

	b.targetLock.Lock()
	defer b.targetLock.Unlock()

	ts := b.targets(host)

	delete(ts, target)

	b.routes.Store(host, ts)

	return nil
}

func (b *StorageMemory) targets(host string) map[string]bool {
	v, ok := b.routes.Load(host)
	if !ok {
		return map[string]bool{}
	}

	h, ok := v.(map[string]bool)
	if !ok {
		return map[string]bool{}
	}

	return h
}

type activityTracker struct {
	activity  sync.Map
	counts    sync.Map
	countLock sync.Mutex
}

func (t *activityTracker) ActiveSince(key string, cutoff time.Time) (bool, error) {
	a, err := t.Activity(key)
	if err != nil {
		return false, err
	}

	c, err := t.Count(key)
	if err != nil {
		return false, err
	}

	return a.After(cutoff) || c > 0, nil
}

func (t *activityTracker) Activity(key string) (time.Time, error) {
	av, _ := t.activity.LoadOrStore(key, time.Time{})

	if a, ok := av.(time.Time); ok {
		return a, nil
	}

	return time.Time{}, fmt.Errorf("invalid activity type: %T", av)
}

func (t *activityTracker) Begin(key string) error {
	t.activity.Store(key, time.Now().UTC())

	if err := t.addCount(key, 1); err != nil {
		return err
	}

	return nil
}

func (t *activityTracker) Count(key string) (int64, error) {
	t.countLock.Lock()
	defer t.countLock.Unlock()

	cv, _ := t.counts.LoadOrStore(key, int64(0))

	if c, ok := cv.(int64); ok {
		return c, nil
	}

	return 0, fmt.Errorf("invalid count type: %T", cv)
}

func (t *activityTracker) End(key string) error {
	return t.addCount(key, -1)
}

func (t *activityTracker) addCount(key string, n int64) error {
	t.countLock.Lock()
	defer t.countLock.Unlock()

	cv, _ := t.counts.LoadOrStore(key, int64(0))

	c, ok := cv.(int64)
	if !ok {
		return fmt.Errorf("invalid count type: %T", cv)
	}

	t.counts.Store(key, c+n)

	return nil
}
