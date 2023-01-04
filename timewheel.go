package timewheel

import (
	"container/list"
	"math/rand"
	"time"
)

type job func()

type TimeWheel struct {
	step            time.Duration // 间隔步伐
	slot            []*list.List  // 槽位
	ticker          *time.Ticker  // 定时器
	slotNum         int           // 槽位的数量
	currentPos      int           // 当前走到那个槽位
	addTask         chan task     // 添加任务
	stopTimeWheel   chan bool     // 关闭定时器
	workPoolSize    int           // 工作groutine大小 默认10
	workerQueueSize int           // 队列大小 默认1024
	workQueue       []chan job    // groutine对应的工作列队
	closeWorkQueue  []chan bool   // 关闭工作队列
}

// 添加的任务
type task struct {
	delay     time.Duration // 延时的时间
	circleNum int           // 需要转动几圈的轮次
	job       job           // 执行函数
	cycle     int           // 0 一次性 1周期性
}

// 步长 槽位大小
func New(step time.Duration, slot int) *TimeWheel {

	if step <= 0 || slot <= 0 {
		panic("error:  步长,槽位 不能小于0")
	}

	timeWheel := &TimeWheel{
		step:            step,
		slotNum:         slot,
		slot:            make([]*list.List, slot),
		currentPos:      0,
		addTask:         make(chan task, 1),
		stopTimeWheel:   make(chan bool, 1),
		workPoolSize:    10,
		workerQueueSize: 1024,
		workQueue:       make([]chan job, 10),
		closeWorkQueue:  make([]chan bool, 10),
	}

	timeWheel.initSlot()
	return timeWheel
}

// 调整工作池容量
func (t *TimeWheel) SetInitWorkPoolSize(workPoolSize int) {
	if workPoolSize <= 0 {
		return
	}
	t.workPoolSize = workPoolSize
}

// 调整工作队列容量
func (t *TimeWheel) SetInitWorkerQueueSize(workerQueueSize int) {
	if workerQueueSize <= 0 {
		return
	}
	t.workerQueueSize = workerQueueSize
}

// 启动时间轮
func (t *TimeWheel) Start() {

	t.ticker = time.NewTicker(t.step)

	go t.start()
	go t.startWorkPool()
}

// 停止时间轮
func (t *TimeWheel) Stop() {
	t.stopTimeWheel <- true
}

// 延时任务
func (t *TimeWheel) AddFunc(delay time.Duration, job func()) {
	t.addFunc(delay, 0, job)
}

// 定时任务
func (t *TimeWheel) AddCycleFunc(delay time.Duration, job func()) {
	t.addFunc(delay, 1, job)
}

func (t *TimeWheel) addFunc(delay time.Duration, cycle int, job func()) {

	task := task{
		delay: delay,
		job:   job,
		cycle: cycle,
	}
	t.addTask <- task

}

func (t *TimeWheel) start() {

	for {
		select {
		case <-t.ticker.C:
			t.jobHandler()
		case task := <-t.addTask:
			t.addTaskToSlot(&task)
		case <-t.stopTimeWheel:
			t.stop()
			return
		}
	}

}

func (t *TimeWheel) stop() {

	t.ticker.Stop()
	for _, v := range t.closeWorkQueue {
		v <- true
	}
}

func (t *TimeWheel) jobHandler() {

	slot := t.slot[t.currentPos]
	t.doJob(slot)

	if t.currentPos == t.slotNum-1 {
		t.currentPos = 0
	} else {
		t.currentPos++
	}
}

func (t *TimeWheel) doJob(l *list.List) {

	for e := l.Front(); e != nil; {
		task := e.Value.(*task)

		if task.circleNum > 0 {
			task.circleNum--
			e = e.Next()
			continue
		}

		t.sendJobToWorkQueue(task.job)
		next := e.Next()
		l.Remove(e)
		e = next
		if task.cycle == 1 {
			t.addTask <- *task
		}
	}
}

func (t *TimeWheel) addTaskToSlot(task *task) {
	pos, circle := t.getPosAndCircle(task.delay)
	task.circleNum = circle
	t.slot[pos].PushBack(task)
}

// 初始化槽位
func (t *TimeWheel) initSlot() {
	for i := 0; i < t.slotNum; i++ {
		t.slot[i] = list.New()
	}
}

// 获取轮数 和 槽位
func (t *TimeWheel) getPosAndCircle(d time.Duration) (pos int, circle int) {

	delaySeconds := int(d.Seconds())
	stepSecond := int(t.step.Seconds())
	circle = int(delaySeconds / stepSecond / t.slotNum)
	pos = int(t.currentPos+delaySeconds/stepSecond) % t.slotNum
	return
}

func (t *TimeWheel) startWorkPool() {

	t.workQueue = make([]chan job, t.workPoolSize)
	t.closeWorkQueue = make([]chan bool, t.workPoolSize)

	for i := 0; i < t.workPoolSize; i++ {
		t.workQueue[i] = make(chan job, t.workerQueueSize)
		t.closeWorkQueue[i] = make(chan bool)
		go t.startOneHandler(t.workQueue[i], t.closeWorkQueue[i])
	}
}

func (t *TimeWheel) startOneHandler(workQueue chan job, closeWorkQueue chan bool) {

	for {
		select {
		case job := <-workQueue:
			job()
		case <-closeWorkQueue:
			return
		}
	}
}

func (t *TimeWheel) sendJobToWorkQueue(job job) {

	// 缺少队列任务分配 现在是随机分配的
	rand.Seed(time.Now().UnixNano())
	workId := rand.Intn(t.workPoolSize)
	t.workQueue[workId] <- job
}
