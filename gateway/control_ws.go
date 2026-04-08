package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/LittleSongxx/TinyClaw/node"
)

func (s *Service) HandleControlWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var connect ConnectFrame
	if err := conn.ReadJSON(&connect); err != nil {
		return
	}
	if connect.Type != FrameTypeConnect || connect.Role != "control" || !s.authorizeToken(connect.Token) {
		return
	}

	readyFrame, err := NewEventFrame("gateway.ready", map[string]interface{}{
		"nodes": s.ListNodes(r.Context()),
	})
	if err == nil {
		_ = conn.WriteJSON(readyFrame)
	}

	for {
		var request RequestFrame
		if err := conn.ReadJSON(&request); err != nil {
			return
		}

		var response ResponseFrame
		switch request.Action {
		case "nodes.list":
			response, err = NewResponseFrame(request.ID, true, s.ListNodes(r.Context()), "")
		case "sessions.list":
			items, listErr := s.ListSessionMeta(100)
			if listErr != nil {
				response, err = NewResponseFrame(request.ID, false, nil, listErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, items, "")
			}
		case "node.exec":
			var execReq node.NodeCommandRequest
			if len(request.Payload) > 0 {
				if decodeErr := json.Unmarshal(request.Payload, &execReq); decodeErr != nil {
					response, err = NewResponseFrame(request.ID, false, nil, decodeErr.Error())
					break
				}
			}
			result, execErr := s.ExecuteNodeCommand(r.Context(), execReq)
			if execErr != nil {
				response, err = NewResponseFrame(request.ID, false, result, execErr.Error())
			} else {
				response, err = NewResponseFrame(request.ID, true, result, "")
			}
		default:
			response, err = NewResponseFrame(request.ID, false, nil, "unsupported gateway action")
		}

		if err != nil {
			return
		}
		if err := conn.WriteJSON(response); err != nil {
			return
		}
	}
}
