package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

var connections = make(map[int]*websocket.Conn)
var lock = sync.RWMutex{}
var counter int32
var droppedSeqs = make([]int, 0)

func main() {
    r := gin.Default()

    r.GET("/ws", func(c *gin.Context) {
        conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            http.Error(c.Writer, "Could not open websocket connection", http.StatusInternalServerError)
            return
        }
        defer conn.Close()

        var seq int
        lock.Lock()
        if len(droppedSeqs) > 0 {
            seq, droppedSeqs = droppedSeqs[len(droppedSeqs)-1], droppedSeqs[:len(droppedSeqs)-1]
        } else {
            seq = int(atomic.AddInt32(&counter, 1))
        }
        connections[seq] = conn
        lock.Unlock()

        fmt.Println("Player ", seq, " connected")

        for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Player ", seq, " disconnected", "error:", err)
				break
			}
		
			lock.RLock()
			for id, connection := range connections {
				if id != seq {
					if err := connection.WriteMessage(websocket.TextMessage, p); err != nil {
						fmt.Println("Error broadcasting message to player ", id, "error:", err)
					}
				}
			}
			lock.RUnlock()
		}
    })

    r.Run(":8080")
}