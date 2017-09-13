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

// CaseState defines the Case running state
type CaseState int

// Enum values of the CaseState type,
const (
	CaseNew CaseState = iota
	CaseStopped
	CaseStarting
	CaseRunning
	CaseRestarting
	CaseStopping
	CaseKilling
	CaseTerminating
	CaseExited
	CaseKilled
	CaseFatal
	CaseTimeout
	CaseStartError
	CaseCreatePodError
	CaseUnknown
)

func (s CaseState) String() string {
	var ret string

	switch s {
	case CaseNew:
		ret = "NEW"
	case CaseStopped:
		ret = "STOPPED"
	case CaseStarting:
		ret = "STARTING"
	case CaseRunning:
		ret = "RUNNING"
	case CaseRestarting:
		ret = "RESTARTING"
	case CaseStopping:
		ret = "STOPPING"
	case CaseKilling:
		ret = "KILLING"
	case CaseTerminating:
		ret = "TERMINATING"
	case CaseExited:
		ret = "EXITED"
	case CaseKilled:
		ret = "KILLED"
	case CaseFatal:
		ret = "FATAL"
	case CaseTimeout:
		ret = "TIMEOUT"
	case CaseStartError:
		ret = "START ERROR"
	case CaseCreatePodError:
		ret = "CREATE POD ERROR"
	case CaseUnknown:
		ret = "UNKNOWN"
	default:
		ret = "UNKNOWN"
	}

	return ret
}

func (c *Case) changeToState(s CaseState) {
	c.Lock()
	defer c.Unlock()
	c.state = s
	// todo: emit an event here
}

// CaseState returns the current process state
func (c *Case) State() CaseState {
	c.Lock()
	defer c.Unlock()
	return c.state
}
