// Copyright 2015 Vadim Kravcenko
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package gojenkins

import (
	"errors"
	"strconv"
)

type Computers struct {
	BusyExecutors  int             `json:"busyExecutors"`
	Computers      []*NodeResponse `json:"computer"`
	DisplayName    string          `json:"displayName"`
	TotalExecutors int             `json:"totalExecutors"`
}

type Node struct {
	Raw    *NodeResponse
	Client *Client
	Base   string
}

type NodeResponse struct {
	Actions     []interface{} `json:"actions"`
	DisplayName string        `json:"displayName"`
	Executors   []struct {
		CurrentExecutable struct {
			Number    int    `json:"number"`
			URL       string `json:"url"`
			SubBuilds []struct {
				Abort             bool        `json:"abort"`
				Build             interface{} `json:"build"`
				BuildNumber       int         `json:"buildNumber"`
				Duration          string      `json:"duration"`
				Icon              string      `json:"icon"`
				JobName           string      `json:"jobName"`
				ParentBuildNumber int         `json:"parentBuildNumber"`
				ParentJobName     string      `json:"parentJobName"`
				PhaseName         string      `json:"phaseName"`
				Result            string      `json:"result"`
				Retry             bool        `json:"retry"`
				URL               string      `json:"url"`
			} `json:"subBuilds"`
		} `json:"currentExecutable"`
	} `json:"executors"`
	Icon                string   `json:"icon"`
	IconClassName       string   `json:"iconClassName"`
	Idle                bool     `json:"idle"`
	JnlpAgent           bool     `json:"jnlpAgent"`
	LaunchSupported     bool     `json:"launchSupported"`
	LoadStatistics      struct{} `json:"loadStatistics"`
	ManualLaunchAllowed bool     `json:"manualLaunchAllowed"`
	MonitorData         struct {
		Hudson_NodeMonitors_ArchitectureMonitor interface{} `json:"hudson.node_monitors.ArchitectureMonitor"`
		Hudson_NodeMonitors_ClockMonitor        interface{} `json:"hudson.node_monitors.ClockMonitor"`
		Hudson_NodeMonitors_DiskSpaceMonitor    interface{} `json:"hudson.node_monitors.DiskSpaceMonitor"`
		Hudson_NodeMonitors_ResponseTimeMonitor struct {
			Average int64 `json:"average"`
		} `json:"hudson.node_monitors.ResponseTimeMonitor"`
		Hudson_NodeMonitors_SwapSpaceMonitor      interface{} `json:"hudson.node_monitors.SwapSpaceMonitor"`
		Hudson_NodeMonitors_TemporarySpaceMonitor interface{} `json:"hudson.node_monitors.TemporarySpaceMonitor"`
	} `json:"monitorData"`
	NumExecutors       int64         `json:"numExecutors"`
	Offline            bool          `json:"offline"`
	OfflineCause       struct{}      `json:"offlineCause"`
	OfflineCauseReason string        `json:"offlineCauseReason"`
	OneOffExecutors    []interface{} `json:"oneOffExecutors"`
	TemporarilyOffline bool          `json:"temporarilyOffline"`
}

func (n *Node) Info() (*NodeResponse, error) {
	_, err := n.Poll()
	if err != nil {
		return nil, err
	}
	return n.Raw, nil
}

func (n *Node) GetName() string {
	return n.Raw.DisplayName
}

func (n *Node) Delete() (bool, error) {
	resp, err := n.Client.Requester.Post(n.Base+"/doDelete", nil, nil, nil)
	if err != nil {
		return false, err
	}
	return resp.StatusCode == 200, nil
}

func (n *Node) IsOnline() (bool, error) {
	_, err := n.Poll()
	if err != nil {
		return false, err
	}
	return !n.Raw.Offline, nil
}

func (n *Node) IsTemporarilyOffline() (bool, error) {
	_, err := n.Poll()
	if err != nil {
		return false, err
	}
	return n.Raw.TemporarilyOffline, nil
}

func (n *Node) IsIdle() (bool, error) {
	_, err := n.Poll()
	if err != nil {
		return false, err
	}
	return n.Raw.Idle, nil
}

func (n *Node) IsJnlpAgent() (bool, error) {
	_, err := n.Poll()
	if err != nil {
		return false, err
	}
	return n.Raw.JnlpAgent, nil
}

func (n *Node) SetOnline() (bool, error) {
	_, err := n.Poll()

	if err != nil {
		return false, err
	}

	if n.Raw.Offline && !n.Raw.TemporarilyOffline {
		return false, errors.New("Node is Permanently offline, can't bring it up")
	}

	if n.Raw.Offline && n.Raw.TemporarilyOffline {
		return n.ToggleTemporarilyOffline()
	}

	return true, nil
}

func (n *Node) SetOffline(options ...interface{}) (bool, error) {
	if !n.Raw.Offline {
		return n.ToggleTemporarilyOffline(options...)
	}
	return false, errors.New("Node already Offline")
}

func (n *Node) ToggleTemporarilyOffline(options ...interface{}) (bool, error) {
	state_before, err := n.IsTemporarilyOffline()
	if err != nil {
		return false, err
	}
	qr := map[string]string{"offlineMessage": "requested from gojenkins"}
	if len(options) > 0 {
		qr["offlineMessage"] = options[0].(string)
	}
	_, err = n.Client.Requester.Post(n.Base+"/toggleOffline", nil, nil, qr)
	if err != nil {
		return false, err
	}
	new_state, err := n.IsTemporarilyOffline()
	if err != nil {
		return false, err
	}
	if state_before == new_state {
		return false, errors.New("Node state not changed")
	}
	return true, nil
}

func (n *Node) Poll() (int, error) {
	response, err := n.Client.Requester.GetJSON(n.Base, n.Raw, nil)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func (n *Node) LaunchNodeBySSH() (int, error) {
	qr := map[string]string{
		"json":   "",
		"Submit": "Launch slave agent",
	}
	response, err := n.Client.Requester.Post(n.Base+"/launchSlaveAgent", nil, nil, qr)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func (n *Node) Disconnect() (int, error) {
	qr := map[string]string{
		"offlineMessage": "",
		"json":           makeJson(map[string]string{"offlineMessage": ""}),
		"Submit":         "Yes",
	}
	response, err := n.Client.Requester.Post(n.Base+"/doDisconnect", nil, nil, qr)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func (n *Node) GetLogText() (string, error) {
	var log string

	_, err := n.Client.Requester.Post(n.Base+"/log", nil, nil, nil)
	if err != nil {
		return "", err
	}

	qr := map[string]string{"start": "0"}
	_, err = n.Client.Requester.GetJSON(n.Base+"/logText/progressiveHtml/", &log, qr)
	if err != nil {
		return "", nil
	}

	return log, nil
}

// Create a new Node
// Can be JNLPLauncher or SSHLauncher
// Example : jenkins.CreateNode("nodeName", 1, "Description", "/var/lib/jenkins", "jdk8 docker", map[string]string{"method": "JNLPLauncher"})
// By Default JNLPLauncher is created
// Multiple labels should be separated by blanks
func (c *Client) CreateNode(name string, numExecutors int, description string, remoteFS string, label string, options ...interface{}) (*Node, error) {
	params := map[string]string{"method": "JNLPLauncher"}

	if len(options) > 0 {
		params, _ = options[0].(map[string]string)
	}

	if _, ok := params["method"]; !ok {
		params["method"] = "JNLPLauncher"
	}

	method := params["method"]
	var launcher map[string]string
	switch method {
	case "":
		fallthrough
	case "JNLPLauncher":
		launcher = map[string]string{"stapler-class": "hudson.slaves.JNLPLauncher"}
	case "SSHLauncher":
		launcher = map[string]string{
			"stapler-class":        "hudson.plugins.sshslaves.SSHLauncher",
			"$class":               "hudson.plugins.sshslaves.SSHLauncher",
			"host":                 params["host"],
			"port":                 params["port"],
			"credentialsId":        params["credentialsId"],
			"jvmOptions":           params["jvmOptions"],
			"javaPath":             params["javaPath"],
			"prefixStartSlaveCmd":  params["prefixStartSlaveCmd"],
			"suffixStartSlaveCmd":  params["suffixStartSlaveCmd"],
			"maxNumRetries":        params["maxNumRetries"],
			"retryWaitTime":        params["retryWaitTime"],
			"launchTimeoutSeconds": params["launchTimeoutSeconds"],
			"type":                 "hudson.slaves.DumbSlave",
			"stapler-class-bag":    "true"}
	default:
		return nil, errors.New("launcher method not supported")
	}

	node := &Node{Client: c, Raw: new(NodeResponse), Base: "/computer/" + name}
	NODE_TYPE := "hudson.slaves.DumbSlave$DescriptorImpl"
	MODE := "NORMAL"
	qr := map[string]string{
		"name": name,
		"type": NODE_TYPE,
		"json": makeJson(map[string]interface{}{
			"name":               name,
			"nodeDescription":    description,
			"remoteFS":           remoteFS,
			"numExecutors":       numExecutors,
			"mode":               MODE,
			"type":               NODE_TYPE,
			"labelString":        label,
			"retentionsStrategy": map[string]string{"stapler-class": "hudson.slaves.RetentionStrategy$Always"},
			"nodeProperties":     map[string]string{"stapler-class-bag": "true"},
			"launcher":           launcher,
		}),
	}

	resp, err := c.Requester.Post("/computer/doCreateItem", nil, nil, qr)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 400 {
		_, err := node.Poll()
		if err != nil {
			return nil, err
		}
		return node, nil
	}
	return nil, errors.New(strconv.Itoa(resp.StatusCode))
}
