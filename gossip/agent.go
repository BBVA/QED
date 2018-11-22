/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, n.A.
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package gossip

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/bbva/qed/gossip/member"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/protocol"
	"github.com/hashicorp/memberlist"
)

type Alert struct {
	id  string
	msg string
}

type Agent struct {
	config *Config
	Self   *member.Peer

	memberlist *memberlist.Memberlist
	broadcasts *memberlist.TransmitLimitedQueue

	Topology *Topology

	stateLock sync.Mutex

	processors []Processor

	In     chan *protocol.BatchSnapshots
	Out    chan *protocol.BatchSnapshots
	Alerts chan Alert
	quit   chan bool
}

func NewAgent(conf *Config, p []Processor) (agent *Agent, err error) {

	agent = &Agent{
		config:   conf,
		Topology: NewTopology(),
		In:       make(chan *protocol.BatchSnapshots, 1000),
		Out:      make(chan *protocol.BatchSnapshots, 1000),
		Alerts:   make(chan Alert, 100),
		quit:     make(chan bool),
	}

	bindIP, bindPort, err := conf.AddrParts(conf.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("Invalid bind address: %s", err)
	}

	var advertiseIP string
	var advertisePort int
	if conf.AdvertiseAddr != "" {
		advertiseIP, advertisePort, err = conf.AddrParts(conf.AdvertiseAddr)
		if err != nil {
			return nil, fmt.Errorf("Invalid advertise address: %s", err)
		}
	}

	conf.MemberlistConfig = memberlist.DefaultLocalConfig()
	conf.MemberlistConfig.BindAddr = bindIP
	conf.MemberlistConfig.BindPort = bindPort
	conf.MemberlistConfig.AdvertiseAddr = advertiseIP
	conf.MemberlistConfig.AdvertisePort = advertisePort
	conf.MemberlistConfig.Name = conf.NodeName
	conf.MemberlistConfig.Logger = log.GetLogger()

	// Configure delegates
	conf.MemberlistConfig.Delegate = newAgentDelegate(agent)
	conf.MemberlistConfig.Events = &eventDelegate{agent}

	agent.memberlist, err = memberlist.Create(conf.MemberlistConfig)
	if err != nil {
		return nil, err
	}

	// Print local member info
	agent.Self = member.ParsePeer(agent.memberlist.LocalNode())
	log.Infof("Local member %+v", agent.Self)

	// Set broadcast queue
	agent.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return agent.memberlist.NumMembers()
		},
		RetransmitMult: 2,
	}

	return agent, nil
}

func (a *Agent) Start() {

	outTicker := time.NewTimer(2 * time.Second)
	alertTicker := time.NewTimer(1 * time.Second)

	for {
		select {
		case batch := <-a.In:
			for _, p := range a.processors {
				go p.Process(batch)
			}

			a.Out <- batch
		case <-outTicker.C:
			a.sendOutQueue()
		case <-alertTicker.C:
			a.processAlertQueue()
		case <-a.quit:
			return
		}

	}

}

func (a *Agent) processAlertQueue() {

}

func (a *Agent) sendOutQueue() {
	var batch *protocol.BatchSnapshots
	for {
		select {
		case batch = <-a.Out:
		default:
			return
		}

		if batch.TTL <= 0 {
			continue
		}

		batch.TTL--

		for _, dst := range a.route(batch.From) {
			msg, _ := batch.Encode()
			a.memberlist.SendReliable(dst, msg)
		}
	}

}

func (a Agent) route(s *member.Peer) []*memberlist.Node {
	return nil
}

// Join asks the Agent instance to join.
func (a *Agent) Join(addrs []string) (int, error) {

	if a.State() != member.Alive {
		return 0, fmt.Errorf("Agent can't join after Leave or Shutdown")
	}

	if len(addrs) > 0 {
		log.Debugf("Trying to join the cluster using members: %v", addrs)
		return a.memberlist.Join(addrs)
	}
	return 0, nil
}

func (a *Agent) Leave() error {

	// Check the current state
	a.stateLock.Lock()
	switch a.Self.Status {
	case member.Left:
		a.stateLock.Unlock()
		return nil
	case member.Leaving:
		a.stateLock.Unlock()
		return fmt.Errorf("Leave already in progress")
	case member.Shutdown:
		a.stateLock.Unlock()
		return fmt.Errorf("Leave called after Shutdown")
	default:
		a.Self.Status = member.Leaving
		a.stateLock.Unlock()
	}

	// Attempt the memberlist leave
	err := a.memberlist.Leave(a.config.BroadcastTimeout)
	if err != nil {
		return err
	}

	// Wait for the leave to propagate through the cluster. The broadcast
	// timeout is how long we wait for the message to go out from our own
	// queue, but this wait is for that message to propagate through the
	// cluster. In particular, we want to stay up long enough to service
	// any probes from other agents before they learn about us leaving.
	time.Sleep(a.config.LeavePropagateDelay)

	// Transition to Left only if we not already shutdown
	a.stateLock.Lock()
	if a.Self.Status != member.Shutdown {
		a.Self.Status = member.Left
	}
	a.stateLock.Unlock()
	return nil

}

// Shutdown forcefully shuts down the Agent instance, stopping all network
// activity and background maintenance associated with the instance.
//
// This is not a graceful shutdown, and should be preceded by a call
// to Leave. Otherwise, other agents in the cluster will detect this agent's
// exit as a agent failure.
//
// It is safe to call this method multiple times.
func (a *Agent) Shutdown() error {

	a.stateLock.Lock()
	defer a.stateLock.Unlock()

	if a.Self.Status == member.Shutdown {
		return nil
	}

	if a.Self.Status != member.Left {
		log.Info("agent: Shutdown without a Leave")
	}

	a.Self.Status = member.Shutdown
	err := a.memberlist.Shutdown()
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) Memberlist() *memberlist.Memberlist {
	return a.memberlist
}

func (a *Agent) Broadcasts() *memberlist.TransmitLimitedQueue {
	return a.broadcasts
}

func (a *Agent) GetAddrPort() (net.IP, uint16) {
	n := a.memberlist.LocalNode()
	return n.Addr, n.Port
}

func (a *Agent) State() member.Status {
	a.stateLock.Lock()
	defer a.stateLock.Unlock()
	return a.Self.Status
}
