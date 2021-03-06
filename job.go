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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type Job struct {
	Raw    *JobResponse
	Client *Client
	Base   string
}

type JobBuild struct {
	Number int64
	URL    string
}

type InnerJob struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Color string `json:"color"`
}

type ParameterDefinition struct {
	DefaultParameterValue struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	} `json:"defaultParameterValue"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Type        string `json:"type"`
}

type JobResponse struct {
	Actions            []generalObj
	Buildable          bool `json:"buildable"`
	Builds             []JobBuild
	Color              string      `json:"color"`
	ConcurrentBuild    bool        `json:"concurrentBuild"`
	Description        string      `json:"description"`
	DisplayName        string      `json:"displayName"`
	DisplayNameOrNull  interface{} `json:"displayNameOrNull"`
	DownstreamProjects []InnerJob  `json:"downstreamProjects"`
	FirstBuild         JobBuild
	HealthReport       []struct {
		Description   string `json:"description"`
		IconClassName string `json:"iconClassName"`
		IconUrl       string `json:"iconUrl"`
		Score         int64  `json:"score"`
	} `json:"healthReport"`
	InQueue               bool       `json:"inQueue"`
	KeepDependencies      bool       `json:"keepDependencies"`
	LastBuild             JobBuild   `json:"lastBuild"`
	LastCompletedBuild    JobBuild   `json:"lastCompletedBuild"`
	LastFailedBuild       JobBuild   `json:"lastFailedBuild"`
	LastStableBuild       JobBuild   `json:"lastStableBuild"`
	LastSuccessfulBuild   JobBuild   `json:"lastSuccessfulBuild"`
	LastUnstableBuild     JobBuild   `json:"lastUnstableBuild"`
	LastUnsuccessfulBuild JobBuild   `json:"lastUnsuccessfulBuild"`
	Name                  string     `json:"name"`
	SubJobs               []InnerJob `json:"jobs"`
	NextBuildNumber       int64      `json:"nextBuildNumber"`
	Property              []struct {
		ParameterDefinitions []ParameterDefinition `json:"parameterDefinitions"`
	} `json:"property"`
	QueueItem        interface{} `json:"queueItem"`
	Scm              struct{}    `json:"scm"`
	UpstreamProjects []InnerJob  `json:"upstreamProjects"`
	URL              string      `json:"url"`
	Jobs             []InnerJob  `json:"jobs"`
	PrimaryView      *ViewData   `json:"primaryView"`
	Views            []ViewData  `json:"views"`
}

func (j *Job) parentBase() string {
	return j.Base[:strings.LastIndex(j.Base, "/job/")]
}

type History struct {
	BuildNumber    int
	BuildStatus    string
	BuildTimestamp int64
}

func (j *Job) GetName() string {
	return j.Raw.Name
}

func (j *Job) GetDescription() string {
	return j.Raw.Description
}

func (j *Job) GetDetails() *JobResponse {
	return j.Raw
}

func (j *Job) GetBuild(id int64) (*Build, error) {
	build := Build{Client: j.Client, Job: j, Raw: new(BuildResponse), Depth: 1, Base: "/job/" + j.GetName() + "/" + strconv.FormatInt(id, 10)}
	status, err := build.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &build, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}

func (j *Job) getBuildByType(buildType string) (*Build, error) {
	allowed := map[string]JobBuild{
		"lastStableBuild":     j.Raw.LastStableBuild,
		"lastSuccessfulBuild": j.Raw.LastSuccessfulBuild,
		"lastBuild":           j.Raw.LastBuild,
		"lastCompletedBuild":  j.Raw.LastCompletedBuild,
		"firstBuild":          j.Raw.FirstBuild,
		"lastFailedBuild":     j.Raw.LastFailedBuild,
	}
	number := ""
	if val, ok := allowed[buildType]; ok {
		number = strconv.FormatInt(val.Number, 10)
	} else {
		panic("No Such Build")
	}
	build := Build{
		Client: j.Client,
		Depth:  1,
		Job:    j,
		Raw:    new(BuildResponse),
		Base:   j.Base + "/" + number}
	status, err := build.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &build, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}

func (j *Job) GetLastSuccessfulBuild() (*Build, error) {
	return j.getBuildByType("lastSuccessfulBuild")
}

func (j *Job) GetFirstBuild() (*Build, error) {
	return j.getBuildByType("firstBuild")
}

func (j *Job) GetLastBuild() (*Build, error) {
	return j.getBuildByType("lastBuild")
}

func (j *Job) GetLastStableBuild() (*Build, error) {
	return j.getBuildByType("lastStableBuild")
}

func (j *Job) GetLastFailedBuild() (*Build, error) {
	return j.getBuildByType("lastFailedBuild")
}

func (j *Job) GetLastCompletedBuild() (*Build, error) {
	return j.getBuildByType("lastCompletedBuild")
}

// Returns All Builds with Number and URL
func (j *Job) GetAllBuildIds() ([]JobBuild, error) {
	var buildsResp struct {
		Builds []JobBuild `json:"allBuilds"`
	}
	_, err := j.Client.Requester.GetJSON(j.Base, &buildsResp, map[string]string{"tree": "allBuilds[number,url]"})
	if err != nil {
		return nil, err
	}
	return buildsResp.Builds, nil
}

func (j *Job) GetSubJobsMetadata() []InnerJob {
	return j.Raw.SubJobs
}

func (j *Job) GetUpstreamJobsMetadata() []InnerJob {
	return j.Raw.UpstreamProjects
}

func (j *Job) GetDownstreamJobsMetadata() []InnerJob {
	return j.Raw.DownstreamProjects
}

func (j *Job) GetSubJobs() ([]*Job, error) {
	jobs := make([]*Job, len(j.Raw.SubJobs))
	for i, job := range j.Raw.SubJobs {
		ji, err := j.Client.GetSubJob(j.GetName(), job.Name)
		if err != nil {
			return nil, err
		}
		jobs[i] = ji
	}
	return jobs, nil
}

func (j *Job) GetInnerJobsMetadata() []InnerJob {
	return j.Raw.Jobs
}

func (j *Job) GetUpstreamJobs() ([]*Job, error) {
	jobs := make([]*Job, len(j.Raw.UpstreamProjects))
	for i, job := range j.Raw.UpstreamProjects {
		ji, err := j.Client.GetJob(job.Name)
		if err != nil {
			return nil, err
		}
		jobs[i] = ji
	}
	return jobs, nil
}

func (j *Job) GetDownstreamJobs() ([]*Job, error) {
	jobs := make([]*Job, len(j.Raw.DownstreamProjects))
	for i, job := range j.Raw.DownstreamProjects {
		ji, err := j.Client.GetJob(job.Name)
		if err != nil {
			return nil, err
		}
		jobs[i] = ji
	}
	return jobs, nil
}

func (j *Job) GetInnerJob(id string) (*Job, error) {
	job := Job{Client: j.Client, Raw: new(JobResponse), Base: j.Base + "/job/" + id}
	status, err := job.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &job, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}

func (j *Job) GetInnerJobs() ([]*Job, error) {
	jobs := make([]*Job, len(j.Raw.Jobs))
	for i, job := range j.Raw.Jobs {
		ji, err := j.GetInnerJob(job.Name)
		if err != nil {
			return nil, err
		}
		jobs[i] = ji
	}
	return jobs, nil
}

func (j *Job) Enable() (bool, error) {
	resp, err := j.Client.Requester.Post(j.Base+"/enable", nil, nil, nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != 200 {
		return false, errors.New(strconv.Itoa(resp.StatusCode))
	}
	return true, nil
}

func (j *Job) Disable() (bool, error) {
	resp, err := j.Client.Requester.Post(j.Base+"/disable", nil, nil, nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != 200 {
		return false, errors.New(strconv.Itoa(resp.StatusCode))
	}
	return true, nil
}

func (j *Job) Delete() (bool, error) {
	resp, err := j.Client.Requester.Post(j.Base+"/doDelete", nil, nil, nil)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != 200 {
		return false, errors.New(strconv.Itoa(resp.StatusCode))
	}
	return true, nil
}

func (j *Job) Rename(name string) (bool, error) {
	data := url.Values{}
	data.Set("newName", name)
	_, err := j.Client.Requester.Post(j.Base+"/doRename", bytes.NewBufferString(data.Encode()), nil, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (j *Job) Create(config string, qr ...interface{}) (*Job, error) {
	var querystring map[string]string
	if len(qr) > 0 {
		querystring = qr[0].(map[string]string)
	}
	resp, err := j.Client.Requester.PostXML(j.parentBase()+"/createItem", config, j.Raw, querystring)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 200 {
		j.Poll()
		return j, nil
	}
	return nil, errors.New(strconv.Itoa(resp.StatusCode))
}

func (j *Job) Copy(destinationName string) (*Job, error) {
	qr := map[string]string{"name": destinationName, "from": j.GetName(), "mode": "copy"}
	resp, err := j.Client.Requester.Post(j.parentBase()+"/createItem", nil, nil, qr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 200 {
		newJob := &Job{Client: j.Client, Raw: new(JobResponse), Base: "/job/" + destinationName}
		_, err := newJob.Poll()
		if err != nil {
			return nil, err
		}
		return newJob, nil
	}
	return nil, errors.New(strconv.Itoa(resp.StatusCode))
}

func (j *Job) UpdateConfig(config string) error {

	var querystring map[string]string

	resp, err := j.Client.Requester.PostXML(j.Base+"/config.xml", config, nil, querystring)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		j.Poll()
		return nil
	}
	return errors.New(strconv.Itoa(resp.StatusCode))

}

func (j *Job) GetConfig() (string, error) {
	var data string
	_, err := j.Client.Requester.GetXML(j.Base+"/config.xml", &data, nil)
	if err != nil {
		return "", err
	}
	return data, nil
}

func (j *Job) GetParameters() ([]ParameterDefinition, error) {
	_, err := j.Poll()
	if err != nil {
		return nil, err
	}
	var parameters []ParameterDefinition
	for _, property := range j.Raw.Property {
		parameters = append(parameters, property.ParameterDefinitions...)
	}
	return parameters, nil
}

func (j *Job) IsQueued() (bool, error) {
	if _, err := j.Poll(); err != nil {
		return false, err
	}
	return j.Raw.InQueue, nil
}

func (j *Job) IsRunning() (bool, error) {
	if _, err := j.Poll(); err != nil {
		return false, err
	}
	lastBuild, err := j.GetLastBuild()
	if err != nil {
		return false, err
	}
	return lastBuild.IsRunning(), nil
}

func (j *Job) IsEnabled() (bool, error) {
	if _, err := j.Poll(); err != nil {
		return false, err
	}
	return j.Raw.Color != "disabled", nil
}

func (j *Job) HasQueuedBuild() {
	panic("Not Implemented yet")
}

func (j *Job) InvokeSimple(params map[string]string) (int64, error) {
	isQueued, err := j.IsQueued()
	if err != nil {
		return 0, err
	}
	if isQueued {
		Error.Printf("%s is already running", j.GetName())
		return 0, nil
	}

	endpoint := "/build"
	parameters, err := j.GetParameters()
	if err != nil {
		return 0, err
	}
	if len(parameters) > 0 {
		endpoint = "/buildWithParameters"
	}
	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}
	resp, err := j.Client.Requester.Post(j.Base+endpoint, bytes.NewBufferString(data.Encode()), nil, nil)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return 0, errors.New("Could not invoke job " + j.GetName())
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return 0, errors.New("Don't have key \"Location\" in response of header")
	}

	u, err := url.Parse(location)
	if err != nil {
		return 0, err
	}

	number, err := strconv.ParseInt(path.Base(u.Path), 10, 64)
	if err != nil {
		return 0, err
	}

	return number, nil
}

func (j *Job) Invoke(files []string, skipIfRunning bool, params map[string]string, cause string, securityToken string) (bool, error) {
	isQueued, err := j.IsQueued()
	if err != nil {
		return false, err
	}
	if isQueued {
		Error.Printf("%s is already running", j.GetName())
		return false, nil
	}
	isRunning, err := j.IsRunning()
	if err != nil {
		return false, err
	}
	if isRunning && skipIfRunning {
		return false, fmt.Errorf("Will not request new build because %s is already running", j.GetName())
	}

	base := "/build"

	// If parameters are specified - url is /builWithParameters
	if params != nil {
		base = "/buildWithParameters"
	} else {
		params = make(map[string]string)
	}

	// If files are specified - url is /build
	if files != nil {
		base = "/build"
	}
	reqParams := map[string]string{}
	buildParams := map[string]string{}
	if securityToken != "" {
		reqParams["token"] = securityToken
	}

	buildParams["json"] = string(makeJson(params))
	b, _ := json.Marshal(buildParams)
	resp, err := j.Client.Requester.PostFiles(j.Base+base, bytes.NewBuffer(b), nil, reqParams, files)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		return true, nil
	}
	return false, errors.New(strconv.Itoa(resp.StatusCode))
}

func (j *Job) Poll() (int, error) {
	response, err := j.Client.Requester.GetJSON(j.Base, j.Raw, nil)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}

func (j *Job) History() ([]*History, error) {
	resp, err := j.Client.Requester.Get(j.Base+"/buildHistory/ajax", nil, nil)
	if err != nil {
		return nil, err
	}
	return parseBuildHistory(resp.Body), nil
}

// Create a new job in the folder
// Example: jenkins.CreateJobInFolder("<config></config>", "newJobName", "myFolder", "parentFolder")
func (c *Client) CreateJobInFolder(config string, jobName string, parentIDs ...string) (*Job, error) {
	jobObj := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + strings.Join(append(parentIDs, jobName), "/job/")}
	qr := map[string]string{
		"name": jobName,
	}
	job, err := jobObj.Create(config, qr)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// Create a new job from config File
// Method takes XML string as first parameter, and if the name is not specified in the config file
// takes name as string as second parameter
// e.g jenkins.CreateJob("<config></config>","newJobName")
func (c *Client) CreateJob(config string, options ...interface{}) (*Job, error) {
	qr := make(map[string]string)
	if len(options) > 0 {
		qr["name"] = options[0].(string)
	} else {
		return nil, errors.New("Error Creating Job, job name is missing")
	}
	jobObj := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + qr["name"]}
	job, err := jobObj.Create(config, qr)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// Rename a job.
// First parameter job old name, Second parameter job new name.
func (c *Client) RenameJob(job string, name string) *Job {
	jobObj := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + job}
	jobObj.Rename(name)
	return &jobObj
}

// Create a copy of a job.
// First parameter Name of the job to copy from, Second parameter new job name.
func (c *Client) CopyJob(copyFrom string, newName string) (*Job, error) {
	job := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + copyFrom}
	_, err := job.Poll()
	if err != nil {
		return nil, err
	}
	return job.Copy(newName)
}

// Delete a job.
func (c *Client) DeleteJob(name string) (bool, error) {
	job := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + name}
	return job.Delete()
}

// Invoke a job.
// First parameter job name, second parameter is optional Build parameters.
func (c *Client) BuildJob(name string, options ...interface{}) (int64, error) {
	job := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + name}
	var params map[string]string
	if len(options) > 0 {
		params, _ = options[0].(map[string]string)
	}
	return job.InvokeSimple(params)
}

func (c *Client) GetJob(id string, parentIDs ...string) (*Job, error) {
	job := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + strings.Join(append(parentIDs, id), "/job/")}
	status, err := job.Poll()
	if err != nil {
		return nil, err
	}
	if status == 200 {
		return &job, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}

func (c *Client) GetSubJob(parentId string, childId string) (*Job, error) {
	job := Job{Client: c, Raw: new(JobResponse), Base: "/job/" + parentId + "/job/" + childId}
	status, err := job.Poll()
	if err != nil {
		return nil, fmt.Errorf("trouble polling job: %v", err)
	}
	if status == 200 {
		return &job, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}
