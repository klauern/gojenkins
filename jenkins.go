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
func (c *Client) Init() (*Client, error) {
	c.initLoggers()

	// Check Connection
	c.Raw = new(ExecutorResponse)
	rsp, err := c.Requester.GetJSON("/", c.Raw, nil)
	if err != nil {
		return nil, err
	}

	c.Version = rsp.Header.Get("X-Jenkins")
	if c.Raw == nil {
		return nil, errors.New("Connection Failed, Please verify that the host and credentials are correct.")
	}

	return c, nil
}

func (c *Client) initLoggers() {
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
func (c *Client) Info() (*ExecutorResponse, error) {
	_, err := c.Requester.Get("/", c.Raw, nil)

	if err != nil {
		return nil, err
	}
	return c.Raw, nil
}

func (c *Client) GetNode(name string) (*Node, error) {
	node := Node{Client: c, Raw: new(NodeResponse), Base: "/computer/" + name}
	status, err := node.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &node, nil
	}
	return nil, errors.New("No node found")
}

func (c *Client) GetLabel(name string) (*Label, error) {
	label := Label{Client: c, Raw: new(LabelResponse), Base: "/label/" + name}
	status, err := label.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &label, nil
	}
	return nil, errors.New("No label found")
}

func (c *Client) GetBuild(jobName string, number int64) (*Build, error) {
	job, err := c.GetJob(jobName)
	if err != nil {
		return nil, err
	}
	build, err := job.GetBuild(number)

	if err != nil {
		return nil, err
	}
	return build, nil
}

func (c *Client) GetAllNodes() ([]*Node, error) {
	computers := new(Computers)

	qr := map[string]string{
		"depth": "1",
	}

	_, err := c.Requester.GetJSON("/computer", computers, qr)
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, len(computers.Computers))
	for i, node := range computers.Computers {
		nodes[i] = &Node{Client: c, Raw: node, Base: "/computer/" + node.DisplayName}
	}

	return nodes, nil
}

// Get all builds Numbers and URLS for a specific job.
// There are only build IDs here,
// To get all the other info of the build use jenkins.GetBuild(job,buildNumber)
// or job.GetBuild(buildNumber)
func (c *Client) GetAllBuildIds(job string) ([]JobBuild, error) {
	jobObj, err := c.GetJob(job)
	if err != nil {
		return nil, err
	}
	return jobObj.GetAllBuildIds()
}

// Get Only Array of Job Names, Color, URL
// Does not query each single Job.
func (c *Client) GetAllJobNames() ([]InnerJob, error) {
	exec := Executor{Raw: new(ExecutorResponse), Client: c}
	_, err := c.Requester.GetJSON("/", exec.Raw, nil)

	if err != nil {
		return nil, err
	}

	return exec.Raw.Jobs, nil
}

// Get All Possible Job Objects.
// Each job will be queried.
func (c *Client) GetAllJobs() ([]*Job, error) {
	exec := Executor{Raw: new(ExecutorResponse), Client: c}
	_, err := c.Requester.GetJSON("/", exec.Raw, nil)

	if err != nil {
		return nil, err
	}

	jobs := make([]*Job, len(exec.Raw.Jobs))
	for i, job := range exec.Raw.Jobs {
		ji, err := c.GetJob(job.Name)
		if err != nil {
			return nil, err
		}
		jobs[i] = ji
	}
	return jobs, nil
}

// Returns a Queue
func (c *Client) GetQueue() (*Queue, error) {
	q := &Queue{Client: c, Raw: new(queueResponse), Base: c.GetQueueUrl()}
	_, err := q.Poll()
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (c *Client) GetQueueUrl() string {
	return "/queue"
}

// Get Artifact data by Hash
func (c *Client) GetArtifactData(id string) (*FingerPrintResponse, error) {
	fp := FingerPrint{Client: c, Base: "/fingerprint/", Id: id, Raw: new(FingerPrintResponse)}
	return fp.GetInfo()
}

func (c *Client) Poll() (int, error) {
	resp, err := c.Requester.GetJSON("/", c.Raw, nil)
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
