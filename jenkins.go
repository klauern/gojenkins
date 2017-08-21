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

// Gojenkins is a Jenkins Client in Go, that exposes the jenkins REST api in a more developer friendly way.
package gojenkins

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// Basic Authentication
type BasicAuth struct {
	Username string
	Password string
}

type Client struct {
	auth       *Authentication
	HTTPClient *http.Client
	Log        *logrus.Logger
	BaseURL    string
	Server     string
	Version    string
	Raw        *ExecutorResponse
	Requester  *Requester
}

// Loggers
var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

// Init Method. Should be called after creating a Client Instance.
// e.g jenkins := CreateJenkins("url").Init()
// HTTP Client is set here, Connection to jenkins is tested here.
func (j *Client) Init() (*Client, error) {
	j.initLoggers()

	// Check Connection
	j.Raw = new(ExecutorResponse)
	rsp, err := j.Requester.GetJSON("/", j.Raw, nil)
	if err != nil {
		return nil, err
	}

	j.Version = rsp.Header.Get("X-Jenkins")
	if j.Raw == nil {
		return nil, errors.New("Connection Failed, Please verify that the host and credentials are correct.")
	}

	return j, nil
}

func (j *Client) initLoggers() {
	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(os.Stdout,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(os.Stderr,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

// Get Basic Information About Jenkins
func (j *Client) Info() (*ExecutorResponse, error) {
	_, err := j.Requester.Get("/", j.Raw, nil)

	if err != nil {
		return nil, err
	}
	return j.Raw, nil
}

func (j *Client) GetNode(name string) (*Node, error) {
	node := Node{Client: j, Raw: new(NodeResponse), Base: "/computer/" + name}
	status, err := node.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &node, nil
	}
	return nil, errors.New("No node found")
}

func (j *Client) GetLabel(name string) (*Label, error) {
	label := Label{Jenkins: j, Raw: new(LabelResponse), Base: "/label/" + name}
	status, err := label.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &label, nil
	}
	return nil, errors.New("No label found")
}

func (j *Client) GetBuild(jobName string, number int64) (*Build, error) {
	job, err := j.GetJob(jobName)
	if err != nil {
		return nil, err
	}
	build, err := job.GetBuild(number)

	if err != nil {
		return nil, err
	}
	return build, nil
}

func (j *Client) GetAllNodes() ([]*Node, error) {
	computers := new(Computers)

	qr := map[string]string{
		"depth": "1",
	}

	_, err := j.Requester.GetJSON("/computer", computers, qr)
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, len(computers.Computers))
	for i, node := range computers.Computers {
		nodes[i] = &Node{Client: j, Raw: node, Base: "/computer/" + node.DisplayName}
	}

	return nodes, nil
}

// Get all builds Numbers and URLS for a specific job.
// There are only build IDs here,
// To get all the other info of the build use jenkins.GetBuild(job,buildNumber)
// or job.GetBuild(buildNumber)
func (j *Client) GetAllBuildIds(job string) ([]JobBuild, error) {
	jobObj, err := j.GetJob(job)
	if err != nil {
		return nil, err
	}
	return jobObj.GetAllBuildIds()
}

// Get Only Array of Job Names, Color, URL
// Does not query each single Job.
func (j *Client) GetAllJobNames() ([]InnerJob, error) {
	exec := Executor{Raw: new(ExecutorResponse), Jenkins: j}
	_, err := j.Requester.GetJSON("/", exec.Raw, nil)

	if err != nil {
		return nil, err
	}

	return exec.Raw.Jobs, nil
}

// Get All Possible Job Objects.
// Each job will be queried.
func (j *Client) GetAllJobs() ([]*Job, error) {
	exec := Executor{Raw: new(ExecutorResponse), Jenkins: j}
	_, err := j.Requester.GetJSON("/", exec.Raw, nil)

	if err != nil {
		return nil, err
	}

	jobs := make([]*Job, len(exec.Raw.Jobs))
	for i, job := range exec.Raw.Jobs {
		ji, err := j.GetJob(job.Name)
		if err != nil {
			return nil, err
		}
		jobs[i] = ji
	}
	return jobs, nil
}

// Returns a Queue
func (j *Client) GetQueue() (*Queue, error) {
	q := &Queue{Jenkins: j, Raw: new(queueResponse), Base: j.GetQueueUrl()}
	_, err := q.Poll()
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (j *Client) GetQueueUrl() string {
	return "/queue"
}

// Get Artifact data by Hash
func (j *Client) GetArtifactData(id string) (*FingerPrintResponse, error) {
	fp := FingerPrint{Jenkins: j, Base: "/fingerprint/", Id: id, Raw: new(FingerPrintResponse)}
	return fp.GetInfo()
}

// Returns the list of all plugins installed on the Jenkins server.
// You can supply depth parameter, to limit how much data is returned.
func (j *Client) GetPlugins(depth int) (*Plugins, error) {
	p := Plugins{Jenkins: j, Raw: new(PluginResponse), Base: "/pluginManager", Depth: depth}
	_, err := p.Poll()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Check if the plugin is installed on the server.
// Depth level 1 is used. If you need to go deeper, you can use GetPlugins, and iterate through them.
func (j *Client) HasPlugin(name string) (*Plugin, error) {
	p, err := j.GetPlugins(1)

	if err != nil {
		return nil, err
	}
	return p.Contains(name), nil
}

// Verify FingerPrint
func (j *Client) ValidateFingerPrint(id string) (bool, error) {
	fp := FingerPrint{Jenkins: j, Base: "/fingerprint/", Id: id, Raw: new(FingerPrintResponse)}
	valid, err := fp.Valid()
	if err != nil {
		return false, err
	}
	if valid {
		return true, nil
	}
	return false, nil
}

func (j *Client) GetView(name string) (*View, error) {
	url := "/view/" + name
	view := View{Jenkins: j, Raw: new(ViewResponse), Base: url}
	_, err := view.Poll()
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (j *Client) GetAllViews() ([]*View, error) {
	_, err := j.Poll()
	if err != nil {
		return nil, err
	}
	views := make([]*View, len(j.Raw.Views))
	for i, v := range j.Raw.Views {
		views[i], _ = j.GetView(v.Name)
	}
	return views, nil
}

// Create View
// First Parameter - name of the View
// Second parameter - Type
// Possible Types:
// 		gojenkins.LIST_VIEW
// 		gojenkins.NESTED_VIEW
// 		gojenkins.MY_VIEW
// 		gojenkins.DASHBOARD_VIEW
// 		gojenkins.PIPELINE_VIEW
// Example: jenkins.CreateView("newView",gojenkins.LIST_VIEW)
func (j *Client) CreateView(name string, viewType string) (*View, error) {
	view := &View{Jenkins: j, Raw: new(ViewResponse), Base: "/view/" + name}
	endpoint := "/createView"
	data := map[string]string{
		"name":   name,
		"mode":   viewType,
		"Submit": "OK",
		"json": makeJson(map[string]string{
			"name": name,
			"mode": viewType,
		}),
	}
	r, err := j.Requester.Post(endpoint, nil, view.Raw, data)

	if err != nil {
		return nil, err
	}

	if r.StatusCode == 200 {
		return j.GetView(name)
	}
	return nil, errors.New(strconv.Itoa(r.StatusCode))
}

func (j *Client) Poll() (int, error) {
	resp, err := j.Requester.GetJSON("/", j.Raw, nil)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

// Creates a new Client Instance
// Optional parameters are: client, username, password
// After creating an instance call init method.
func CreateJenkins(client *http.Client, base string, auth ...interface{}) *Client {
	j := &Client{}
	if strings.HasSuffix(base, "/") {
		base = base[:len(base)-1]
	}
	j.Server = base
	j.Requester = &Requester{Base: base, SslVerify: true, Client: client}
	if j.Requester.Client == nil {
		j.Requester.Client = http.DefaultClient
	}
	if len(auth) == 2 {
		j.Requester.BasicAuth = &BasicAuth{Username: auth[0].(string), Password: auth[1].(string)}
	}
	return j
}
