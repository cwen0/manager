// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/juju/errors"
	"github.com/ngaut/log"
)

type Response struct {
	Action     string  `json:"action"`
	StatusCode int     `json:"status_code"`
	Message    string  `json:"message,omitempty"`
	Payload    Payload `json:"payload,omitempty"`
}

func xpost(url string, body []byte) (*Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Errorf("[url: %s] new request failed: %v", url, err)
		return nil, errors.Trace(err)
	}
	return request(req)
}

func request(req *http.Request) (*Response, error) {
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("issue request error %v", err)
		return nil, errors.Trace(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Errorf("fail to request: status code %d", resp.StatusCode)
		return nil, errors.Trace(err)
	}

	bodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("fail to read body %v", err)
		return nil, errors.Trace(err)
	}

	response := &Response{}
	err = json.Unmarshal(bodyByte, response)
	if err != nil {
		log.Errorf("unmarshal error %v", err)
		return nil, errors.Trace(err)
	}

	if response.StatusCode != 200 {
		log.Errorf("fail to request %v", response)
		return nil, errors.Trace(err)
	}

	return response, nil
}
