/*
Package pacer provides a utility to limit the rate at which concurrent
goroutines begin execution.  This addresses situations where running the
concurrent goroutines is OK, as long as their execution does not start at the
same time.

The pacer package is independent of the workerpool package.  Paced functions
can be submitted to a workerpool or can be run as goroutines, and execution
will be paced in both cases.

*/
package pacer

import "time"

// Pacer is a goroutine rate limiter.  When concurrent goroutines call
// Pacer.Next(), the call returns in a single goroutine at a time, at a rate no
// faster than one per delay time.
//
// To use Pacer, create a new Pacer giving the interval that must elapse
// between the start of one task the start of the next task.  Then call Pace(),
// passing your task function.  A new paced task function is returned that can
// then be passed to WorkerPool's Submit() or SubmitWait(), or called as a go
// routine.  Paced functions, that are run as goroutines, are also paced.  For
// example:
//
//     pacer := pacer.NewPacer(time.Second)
//
//     pacedTask := pacer.Pace(func() {
//         fmt.Println("Hello World")
//     })
//
//     wp := workerpool.New(5)
//     wp.Submit(pacedTask)
//
//     go pacedTask()
//
// NOTE: Do not call Pacer.Stop() until all paced tasks have completed, or
// paced tasks will hang waiting for pacer to unblock them.
type Pacer struct {  //先定义结构体.
	delay  time.Duration
	gate   chan struct{}
	pause  chan struct{}
	paused chan struct{}
}

// NewPacer creates and runs a new Pacer.
func NewPacer(delay time.Duration) *Pacer {
	p := &Pacer{
		delay:  delay,
		gate:   make(chan struct{}),
		pause:  make(chan struct{}, 1),
		paused: make(chan struct{}, 1),
	}

	go p.run()
	return p
}

// Pace wraps a function in a paced function.  The returned paced function can
// then be submitted to the workerpool, using Submit or SubmitWait, and
// starting the tasks is paced according to the pacer's delay.
func (p *Pacer) Pace(task func()) func() { // 放入一个函数.返回一个新函数.新函数调用p.Next然后再调用task
	return func() {
		p.Next()
		task()
	}
}










// Next submits a run request to the gate and returns when it is time to run.
func (p *Pacer) Next() {
	// Wait for item to be read from gate.
	p.gate <- struct{}{}
}

// Stop stops the Pacer from running.  Do not call until all paced tasks have
// completed, or paced tasks will hang waiting for pacer to unblock them.
func (p *Pacer) Stop() {  //关闭所有任务.
	close(p.gate)
}

// IsPaused returns true if execution is paused.
func (p *Pacer) IsPaused() bool {   //当paused进程>0的时候说明有进程paused了.
	return len(p.paused) != 0
}

// Pause suspends execution of any tasks by the pacer.
func (p *Pacer) Pause() {
	p.pause <- struct{}{}  // block this channel
	p.paused <- struct{}{} // set flag to indicate paused
}

// Resume continues execution after Pause.
func (p *Pacer) Resume() {
	<-p.paused // clear flag to indicate paused
	<-p.pause  // unblock this channel
}

func (p *Pacer) run() {
	// Read item from gate no faster than one per delay.
	// Reading from the unbuffered channel serves as a "tick"
	// and unblocks the writer.
	for _ = range p.gate {  // 用range来遍历一个channel
		time.Sleep(p.delay)
		p.pause <- struct{}{} // will wait here if channel blocked  这两行用于阻塞这个for循环.
								//阻塞会阻塞掉当前代码所在的go语句的上下文.所以就实现了,在gate中每跑一个等待一段时间.
		<-p.pause             // clear channel
	}
}

//
//从无缓存的 channel 中读取消息会阻塞，直到有 goroutine 向该 channel 中发送消息；同理，向无缓存的 channel 中发送消息也会阻塞，直到有 goroutine 从 channel 中读取消息。
//
//作者：不智鱼
//链接：https://www.jianshu.com/p/24ede9e90490
//来源：简书
//著作权归作者所有。商业转载请联系作者获得授权，非商业转载请注明出处。
