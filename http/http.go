package http

import (
	"../common"
	"../node"
	"../tog"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
)

// 启动HTTP服务器
func StartHttpServer(port uint) {
	// 显示服务器信息
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		if r.Method != http.MethodGet {
			_, err := fmt.Fprintln(w, "Only allow method [GET].")
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		_, err := fmt.Fprintf(w, "Build TimeStamp : %s\n", common.BuildStamp)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fmt.Fprintf(w, "Version         : %s\n", common.Version)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fmt.Fprintf(w, "Go Version      : %s\n", common.GoVersion)
		if err != nil {
			log.Fatal(err)
		}
	})

	// 显示所有的节点信息
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		if r.Method != http.MethodGet {
			_, err := fmt.Fprintln(w, "Only allow method [GET].")
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err := fmt.Fprintln(w, "       NodeId      Host")
		if err != nil {
			log.Fatal(err)
		}

		nodes := node.GetNodes()
		sort.Sort(nodes) // 对节点列表进行排序
		for _, n := range nodes {
			star := " "
			if n.NodeId == common.LeaderNodeId {
				star = "*"
			}
			me := " "
			if n.NodeId == common.LocalNodeId {
				me = "▴"
			}
			_, err := fmt.Fprintf(w, "%s%s %10s %15s:%-5d\n", star, me, n.NodeId, n.Ip, n.HTTPPort)
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	// 读写数据
	http.HandleFunc("/entries", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		switch r.Method {
		case http.MethodGet:
			key := r.URL.Query().Get("key")
			_, err := fmt.Fprintf(w, `{"%s": "%s"}`, key, "233333")
			if err != nil {
				log.Fatal(err)
			}
		case http.MethodPost:
			// 仅可以向leader写数据
			if common.LocalNodeId != common.LeaderNodeId {
				// TODO 可以由follower直接转发HTTP请求到leader
				leaderHttp := ""
				for _, n := range node.GetNodes() {
					if n.NodeId == common.LeaderNodeId {
						leaderHttp = fmt.Sprintf("http://%s:%d/", n.Ip, n.HTTPPort)
					}
				}
				_, err := fmt.Fprintf(w, `This node is not leader, please post data to leader %s: %s`,
					common.LeaderNodeId, leaderHttp)
				if err != nil {
					log.Fatal(err)
				}
				return
			}

			if r.Body == nil {
				http.Error(w, "Please send a request body", 400)
				return
			}

			bodyBuf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Fatal(err)
			}
			body := string(bodyBuf)

			var entries []common.Entry
			err = json.Unmarshal(bodyBuf, &entries) // 优先解析JSON数组
			if err != nil {
				var entry common.Entry
				err := json.Unmarshal(bodyBuf, &entry) // 如果数组解析失败，则解析JSON对象
				if err != nil {
					//http.Error(w, err.Error(), 400)
					http.Error(w, "Post body can't be decode to json: "+body, 400)
					return
				}
				entries = make([]common.Entry, 1)
				entries[0] = entry
			}

			for _, e := range entries {
				_, err = fmt.Fprintf(w, "Post Success: {\"%s\": \"%s\"}\n", e.Key, e.Value)
				if err != nil {
					log.Fatal(err)
				}
			}

			// 把entries加入到leader本地的log[]中
			entries = common.AppendEntryList(entries)

			// leader向follower发送数据，此周期内不再需要主动发送心跳
			common.LeaderSendEntryCh <- true
			// leader向follower发送消息
			node.SendAppendEntries(entries)

			<-common.LeaderAppendSuccess // 如果大部分的follower返回，则leader返回给client
		default:
			_, err := fmt.Fprintln(w, "Only allow method [GET, POST].")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	if tog.LogLevel(tog.INFO) {
		log.Println("HTTP Server Listening Port", port)
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil))
}
