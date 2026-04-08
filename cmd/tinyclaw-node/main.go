package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/gorilla/websocket"
)

func main() {
	logger.InitLogger()

	gatewayWS := flag.String("gateway_ws", "ws://127.0.0.1:36060/gateway/nodes/ws", "gateway node websocket endpoint")
	nodeID := flag.String("node_id", "", "node id")
	nodeName := flag.String("node_name", "", "node name")
	nodeToken := flag.String("node_token", os.Getenv("NODE_PAIRING_TOKEN"), "node pairing token")
	flag.Parse()

	if *nodeID == "" {
		hostname, _ := os.Hostname()
		*nodeID = hostname
		if *nodeName == "" {
			*nodeName = hostname
		}
	}

	driver := node.NewLocalDriver()
	descriptor := &node.NodeDescriptor{
		ID:           *nodeID,
		Name:         *nodeName,
		Platform:     runtime.GOOS,
		Hostname:     hostName(),
		Version:      "v0.1.0",
		Capabilities: driver.Capabilities(),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	for {
		if err := runNode(ctx, *gatewayWS, *nodeToken, descriptor, driver); err != nil {
			logger.Warn("tinyclaw-node disconnected, retrying", "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func runNode(ctx context.Context, gatewayWS, token string, descriptor *node.NodeDescriptor, driver *node.LocalDriver) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, gatewayWS, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	var writeMu sync.Mutex

	connectFrame := gateway.NewConnectFrame("node", token, descriptor)
	if err := writeJSON(&writeMu, conn, connectFrame); err != nil {
		return err
	}

	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				frame, err := gateway.NewEventFrame("node.heartbeat", map[string]interface{}{
					"node_id": descriptor.ID,
				})
				if err == nil {
					_ = writeJSON(&writeMu, conn, frame)
				}
			}
		}
	}()

	for {
		var request gateway.RequestFrame
		if err := conn.ReadJSON(&request); err != nil {
			return err
		}
		if request.Action != "node.command" {
			response, respErr := gateway.NewResponseFrame(request.ID, false, nil, "unsupported node action")
			if respErr == nil {
				_ = writeJSON(&writeMu, conn, response)
			}
			continue
		}

		var command node.NodeCommandRequest
		if err := json.Unmarshal(request.Payload, &command); err != nil {
			response, respErr := gateway.NewResponseFrame(request.ID, false, nil, err.Error())
			if respErr == nil {
				_ = writeJSON(&writeMu, conn, response)
			}
			continue
		}
		if command.ID == "" {
			command.ID = request.ID
		}
		command.NodeID = descriptor.ID

		result, execErr := driver.Execute(ctx, command)
		response, respErr := gateway.NewResponseFrame(request.ID, execErr == nil, result, errorText(execErr))
		if respErr != nil {
			return respErr
		}
		if err := writeJSON(&writeMu, conn, response); err != nil {
			return err
		}
	}
}

func hostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func writeJSON(mu *sync.Mutex, conn *websocket.Conn, payload interface{}) error {
	mu.Lock()
	defer mu.Unlock()
	return conn.WriteJSON(payload)
}
