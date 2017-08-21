package gojenkins

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultAPISuffix is used to retrieve information using the
	// Remote access API: https://wiki.jenkins.io/display/JENKINS/Remote+access+API
	DefaultAPISuffix = "/api/json"
	DefaultAPIPort = "8080"
	DefaultBaseURL = "http://localhost:"+DefaultAPIPort

)

// JenkinsServer represents the Client server itself.  This information is requested at
// the JENKINS_BASE url '/api/json'.
type JenkinsServer struct {
	HudsonVersion        string
	JenkinsVersion       string
	HudsonCLIPort        string
	JenkinsCLIPort       string
	JenkinsCLIPort2      string
	HasCSRFProtection    bool
	CSRFProtectionHeader map[string]string
	Class                string        `json:"_class"`
	AssignedLabels       []string      `json:"assignedLabels"`
	Description          string        `json:"description"`
	Jobs                 []interface{} `json:"jobs"`
	Mode                 string        `json:"mode"`
	NodeDescription      string        `json:"nodeDescription"`
	NodeName             string        `json:"nodeName"`
	NumExecutors         int           `json:"numExecutors"`
	OverallLoad          struct{}      `json:"overallLoad"`
	PrimaryView          struct {
		Class string `json:"_class"`
		Name  string `json:"name"`
		URL   string `json:"url"`
	} `json:"primaryView"`
	QuietingDown   bool `json:"quietingDown"`
	SlaveAgentPort int  `json:"slaveAgentPort"`
	UnlabeledLoad  struct {
		Class string `json:"_class"`
	} `json:"unlabeledLoad"`
	UseCrumbs   bool `json:"useCrumbs"`
	UseSecurity bool `json:"useSecurity"`
	Views       []struct {
		Class string `json:"_class"`
		Name  string `json:"name"`
		URL   string `json:"url"`
	} `json:"views"`
}

type csrfProtectionSettings struct {
	CrumbClass        string `json:"_class"`
	Crumb             string `json:"crumb"`
	CrumbRequestField string `json:"crumbRequestField"`
}

type Authentication struct {
	Username      string
	Password      string
	Token         string
	UsesTokenAuth bool
}

func NewClient(auth *Authentication) (*Client, error) {
	client := &Client{
		auth:       auth,
		HTTPClient: http.DefaultClient,
		Log:        logrus.New(),
		BaseURL:    DefaultBaseURL,
	}
	client.Log.Out = os.Stdout
	return client, nil
}

// Information returns basic information about the Client server itself.
func (c *Client) Information() (*JenkinsServer, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + DefaultAPISuffix)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to get Jenkins Info at "+c.BaseURL)
	}

	var server JenkinsServer
	err = json.NewDecoder(resp.Body).Decode(&server)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to parse Jenkins Information response")
	}

	server.HudsonVersion = resp.Header.Get("X-Hudson")
	server.JenkinsVersion = resp.Header.Get("X-Jenkins")
	server.HudsonCLIPort = resp.Header.Get("X-Hudson-CLI-Port")
	server.JenkinsCLIPort = resp.Header.Get("X-Jenkins-CLI-Port")
	server.JenkinsCLIPort2 = resp.Header.Get("X-Jenkins-CLI-Port2")

	return &server, nil
}

// ConfigureCSRFProtection configures the JenkinsServer with the appropriate CSRF tokens
func (c *Client) ConfigureCSRFProtection(server *JenkinsServer) (*JenkinsServer, error) {
	if server.UseCrumbs {
		resp, err := c.HTTPClient.Get(c.BaseURL + "/crumbIssuer" + DefaultAPISuffix)
		if resp.StatusCode == 404 || err != nil {
			server.HasCSRFProtection = false
			return server, nil
		}
		server.HasCSRFProtection = true
		var settings csrfProtectionSettings
		err = json.NewDecoder(resp.Body).Decode(&settings)
		if err != nil {
			return server, errors.WithMessage(err, "Unable to unmarshal CSRF token response")
		}
		server.CSRFProtectionHeader = map[string]string{
			settings.CrumbRequestField: settings.Crumb,
		}
		return server, nil
	}
	return nil, errors.New("Jenkins server does not use CSRF crumbs")
}
