package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// 飞书机器人认证
func (h *handler) feishuRegistry(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Decode request
	request := struct {
		Challenge string `json:"challenge"`
	}{
		Challenge: "",
	}

	fmt.Println("registry feishu bot", request.Challenge)

	err := json.NewDecoder(r.Body).Decode(&request)
	checkError(err)
	// Return JSON response
	resp := map[string]interface{}{
		"challenge": request.Challenge,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&resp)
	checkError(err)
}
