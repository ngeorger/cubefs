// Copyright 2018 The Chubao Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package proto

type QueryHTTPReply struct {
	Code int32        `json:"code"`
	Msg  string       `json:"msg"`
	Data []*QueryData `json:"data"`
}

type QueryData struct {
	Pid		uint64	`json:"PID"`
	Vol   	string  `json:"VOL"`
	IP		string	`json:"IP"`
	Op		string	`json:"OP"`
	Count	uint64	`json:"TOTAL_COUNT"`
	Size 	uint64  `json:"TOTAL_SIZE"`
}

type QueryView struct {
	Data []*QueryData
}