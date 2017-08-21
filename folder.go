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
	"strings"
	"fmt"
)

type Folder struct {
	Raw     *FolderResponse
	Jenkins *Client
	Base    string
}

type FolderResponse struct {
	Actions     []generalObj
	Description string     `json:"description"`
	DisplayName string     `json:"displayName"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Jobs        []InnerJob `json:"jobs"`
	PrimaryView *ViewData  `json:"primaryView"`
	Views       []ViewData `json:"views"`
}

func (f *Folder) parentBase() string {
	return f.Base[:strings.LastIndex(f.Base, "/job")]
}

func (f *Folder) GetName() string {
	return f.Raw.Name
}

func (f *Folder) Create(name string) (*Folder, error) {
	mode := "com.cloudbees.hudson.plugins.folder.Folder"
	data := map[string]string{
		"name":   name,
		"mode":   mode,
		"Submit": "OK",
		"json": makeJson(map[string]string{
			"name": name,
			"mode": mode,
		}),
	}
	r, err := f.Jenkins.Requester.Post(f.parentBase()+"/createItem", nil, f.Raw, data)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == 200 {
		f.Poll()
		return f, nil
	}
	return nil, errors.New(strconv.Itoa(r.StatusCode))
}

func (f *Folder) Poll() (int, error) {
	response, err := f.Jenkins.Requester.GetJSON(f.Base, f.Raw, nil)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, nil
}


// Create a new folder
// This folder can be nested in other parent folders
// Example: jenkins.CreateFolder("newFolder", "grandparentFolder", "parentFolder")
func (j *Client) CreateFolder(name string, parents ...string) (*Folder, error) {
	folderObj := &Folder{Jenkins: j, Raw: new(FolderResponse), Base: "/job/" + strings.Join(append(parents, name), "/job/")}
	folder, err := folderObj.Create(name)
	if err != nil {
		return nil, err
	}
	return folder, nil
}


func (j *Client) GetFolder(id string, parents ...string) (*Folder, error) {
	folder := Folder{Jenkins: j, Raw: new(FolderResponse), Base: "/job/" + strings.Join(append(parents, id), "/job/")}
	status, err := folder.Poll()
	if err != nil {
		return nil, fmt.Errorf("trouble polling folder: %v", err)
	}
	if status == 200 {
		return &folder, nil
	}
	return nil, errors.New(strconv.Itoa(status))
}

