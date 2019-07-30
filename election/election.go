package election

import (
	"../common"
	"../node"
	"../tog"
	"log"
	"time"
)

// 给所有的节点发送数据
func sendDataToFollowers(nodes []node.Node, data []byte) {
	for _, n := range nodes {
		if n.Conn != nil {
			_, err := n.Conn.Write(data)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func Do() {
	for {
		switch common.Role {
		case common.Follower:
			heartbeatTimeout := common.RandomInt(common.HeartbeatTimeoutMin, common.HeartbeatTimeoutMax)
			select {
			case <-common.HeartbeatTimeoutCh:
				if tog.LogLevel(tog.DEBUG) {
					//log.Printf("%s(me) get heartbeat and reset timer\n", common.LocalNodeId)
				}
			case <-time.After(time.Duration(heartbeatTimeout) * time.Millisecond):
				common.Role = common.Candidate // 更新为候选人
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) heartbeat timeout and become candidate\n", common.LocalNodeId)
				}
			}
		case common.Candidate: // 最复杂 1.成为leader; 2.成为follower; 3.继续下一轮选举
			common.CurrentTerm += 1
			common.Votes = 1 // 首先投票给自己

			nodes := node.GetNodes()
			if len(nodes) == 1 {
				common.VoteSuccessCh <- true
			} else {
				// 发送选举请求
				data := append([]byte{common.VoteRequest}, common.Uint32ToBytes(common.CurrentTerm)...)
				sendDataToFollowers(nodes, data)
			}

			electionTimeout := common.RandomInt(common.ElectionTimeoutMin, common.ElectionTimeoutMax)
			select {
			case success := <-common.VoteSuccessCh:
				if success {
					common.Role = common.Leader
					if tog.LogLevel(tog.DEBUG) {
						log.Printf("%s(me) Vote success and become leader\n", common.LocalNodeId)
					}
					common.LeaderNodeId = common.LocalNodeId // 当前节点为leader节点

					// 选举成功立即发送心跳
					data := append([]byte{common.AppendEntries}, common.Uint32ToBytes(common.CurrentTerm)...)
					nodes := node.GetNodes()
					sendDataToFollowers(nodes, data)

					common.LeaderSendEntryCh <- true
				} else {
					common.Role = common.Follower
					if tog.LogLevel(tog.DEBUG) {
						log.Printf("%s(me) Vote failed and becmoe follower\n", common.LocalNodeId)
					}
				}
			case <-time.After(time.Duration(electionTimeout) * time.Millisecond):
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) this turn election failed, next turn election will start soon\n",
						common.LocalNodeId)
				}
			}
		case common.Leader:
			select {
			case <-common.LeaderSendEntryCh:
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) leader has sent data to followers\n", common.LocalNodeId)
				}
			case <-time.After(common.LeaderCycleTimeout * time.Millisecond):
				if tog.LogLevel(tog.DEBUG) {
					//log.Printf("%s(me) leader not send data, send empty data as heartbeat\n", common.LocalNodeId)
				}
				// 超时则发送心跳
				data := append([]byte{common.AppendEntries}, common.Uint32ToBytes(common.CurrentTerm)...)
				nodes := node.GetNodes()
				sendDataToFollowers(nodes, data)
			}
		}
	}
}
