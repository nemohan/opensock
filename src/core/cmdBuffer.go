//GetHead get element at tail
package core

import "time"
import "utility"

//max cmd live time 30 second
const cmdTTL = 30

type CmdList struct {
	head *listNode
	tail *listNode
	size int
	name string
	log  *utility.LogContext
	//waitGroup *sync.WaitGroup
}
type CmdContainor interface {
}
type listNode struct {
	next        *listNode
	cmd         CmdContainer
	enQueueTime time.Time
}

func (l *CmdList) deQueue() {
	if l.size == 0 {
		return
	}
	l.size--
	//cmd := l.head.cmd
	l.head = l.head.next
}

/*
func (l *CmdList) pushFront(cmd *Client_cmd) {
	node := &listNode{cmd: cmd, next: nil, enQueueTime: time.Now()}
	l.size++
	if l.head == nil {
		l.head = node
		l.tail = node
		return
	}
	node.next = l.head
	l.head = node
	l.size++
}
*/
func (l *CmdList) getFront() *listNode {
	if l.size == 0 {
		return nil
	}
	l.size--
	return l.head
}

// AddTail add element to tail
func (l *CmdList) enQueue(cmd CmdContainer) {
	node := &listNode{cmd: cmd, next: nil, enQueueTime: time.Now()}
	l.size++
	if l.head == nil {
		l.head = node
		l.tail = node
		return
	}

	l.tail.next = node
	l.tail = node
}

//Length how many element in list
func (l *CmdList) length() int {
	return l.size
}

//PumpCmd forward cmd from one channel to another channel
func (l *CmdList) PumpCmd(inChan <-chan CmdContainer, outChan chan<- CmdContainer) {
	log := l.log
	sendFunc := func(cmd CmdContainer) bool {
		select {
		case <-time.Tick(time.Millisecond * 200):
			log.LogWarn("cmd to module:%s timeout", l.name)
			return false
		case outChan <- cmd:
		}
		return true
	}

	for {
	next:
		select {
		case cmd := <-inChan:
			now := time.Now()
			//log.LogDebug("module:%s got cmd:%d from:%d", l.name, cmd.cmdType, cmd.GetFromID())
			for oldCmd := l.getFront(); oldCmd != nil; oldCmd = l.getFront() {
				if now.After(oldCmd.enQueueTime.Add(cmdTTL * time.Second)) {
					l.deQueue()
					log.LogWarn("discard cmd:%d", oldCmd.cmd.GetFromID())
					continue
				}
				if !sendFunc(oldCmd.cmd) {
					l.enQueue(cmd)
					//log.LogInfo("cmd:%d enqueue id:%d. buffered cmd number:%d", cmd.GetFromID(), cmd., l.length())
					break next

				}
				l.deQueue()
			}
			if !sendFunc(cmd) {
				l.enQueue(cmd)
				log.LogWarn("bufferd cmd number:%d", l.length())
			}

		}
	}
}
