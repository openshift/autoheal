/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awx

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/jhernand/openshift-monitoring/pkg/awx/internal/data"
)

// Version is the version of the client.
//
const Version = "0.0.0"

type ConnectionBuilder struct {
	url      string
	proxy    string
	username string
	password string
	agent    string
	token    string
	insecure bool
}

type Connection struct {
	// Basic data:
	base     string
	username string
	password string
	agent    string
	version  string
	token    string

	// The underlying HTTP client:
	client *http.Client
}

func NewConnectionBuilder() *ConnectionBuilder {
	// Create an empty builder:
	b := new(ConnectionBuilder)

	// Set default values:
	b.agent = "AWXClient/" + Version

	return b
}

func (b *ConnectionBuilder) Url(url string) *ConnectionBuilder {
	b.url = url
	return b
}

func (b *ConnectionBuilder) Proxy(proxy string) *ConnectionBuilder {
	b.proxy = proxy
	return b
}

func (b *ConnectionBuilder) Username(username string) *ConnectionBuilder {
	b.username = username
	return b
}

func (b *ConnectionBuilder) Password(password string) *ConnectionBuilder {
	b.password = password
	return b
}

// Agent sets the value of the HTTP user agent header that the client will use in all
// the requests sent to the server. This is optional, and the default value is the name
// of the client followed by the version number, for example 'GoClient/0.0.1'.
//
func (b *ConnectionBuilder) Agent(agent string) *ConnectionBuilder {
	b.agent = agent
	return b
}

func (b *ConnectionBuilder) Token(token string) *ConnectionBuilder {
	b.token = token
	return b
}

func (b *ConnectionBuilder) Insecure(insecure bool) *ConnectionBuilder {
	b.insecure = insecure
	return b
}

func (b *ConnectionBuilder) Build() (c *Connection, err error) {
	// Check the URL:
	if b.url == "" {
		err = fmt.Errorf("The URL is mandatory")
	}
	_, err = url.Parse(b.url)
	if err != nil {
		err = fmt.Errorf("The URL '%s' isn't valid: %s", b.url, err.Error())
		return
	}

	// Check the proxy:
	var proxy *url.URL
	if b.proxy != "" {
		proxy, err = url.Parse(b.proxy)
		if err != nil {
			err = fmt.Errorf("The proxy URL '%s' isn't valid: %s", b.proxy, err.Error())
			return
		}
	}

	// Check the credentials:
	if b.username != "" && b.token != "" {
		err = fmt.Errorf("User name and token are mutually exclusive")
		return
	}
	if b.username == "" && b.token == "" {
		err = fmt.Errorf("Either user name or token must be provided")
		return
	}

	// Create the HTTP client:
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: b.insecure,
			},
			Proxy: func(request *http.Request) (result *url.URL, err error) {
				result = proxy
				return
			},
		},
	}

	// Allocate the connection and save all the objects that will be required later:
	c = new(Connection)
	c.base = b.url
	c.username = b.username
	c.password = b.password
	c.version = "v2"
	c.client = client

	// Ensure that the base URL has an slash at the end:
	if !strings.HasSuffix(c.base, "/") {
		c.base = c.base + "/"
	}

	return
}

func (c *Connection) JobTemplates() *JobTemplatesResource {
	return NewJobTemplatesResource(c, "job_templates")
}

func (c *Connection) Close() {
	c.token = ""
}

// ensureToken makes sure that there is a token available. If there isn't, then it will request a
// new onw to the server.
//
func (c *Connection) ensureToken() error {
	if c.token != "" {
		return nil
	}
	return c.getToken()
}

// getToken requests a new authentication token.
//
func (c *Connection) getToken() error {
	var request data.AuthTokenPostRequest
	var response data.AuthTokenPostResponse
	request.Username = c.username
	request.Password = c.password
	err := c.post("authtoken", nil, &request, &response)
	if err != nil {
		return err
	}
	c.token = response.Token
	return nil
}

// makeUrl calculates the absolute URL for the given relative path and query.
//
func (c *Connection) makeUrl(path string, query url.Values) string {
	// Allocate a buffer large enough for the longest possible URL:
	buffer := new(bytes.Buffer)
	buffer.Grow(len(c.base) + len(c.version) + 1 + len(path) + 1)

	// Write the componentes of the URL:
	buffer.WriteString(c.base)
	buffer.WriteString(c.version)
	if path != "" {
		buffer.WriteString("/")
		buffer.WriteString(path)
	}

	// Make sure that the URL always ends with an slash, as otherwise the API server will send a
	// redirect:
	buffer.WriteString("/")

	// Add the query:
	if query != nil && len(query) > 0 {
		buffer.WriteString("?")
		buffer.WriteString(query.Encode())
	}

	return buffer.String()
}

func (c *Connection) authenticatedGet(path string, query url.Values, output interface{}) error {
	err := c.ensureToken()
	if err != nil {
		return err
	}
	return c.get(path, query, output)
}

func (c *Connection) get(path string, query url.Values, output interface{}) error {
	outputBytes, err := c.rawGet(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(outputBytes, output)
}

func (c *Connection) rawGet(path string, query url.Values) (output []byte, err error) {
	// Send the request:
	address := c.makeUrl(path, query)
	if glog.V(2) {
		glog.Infof("Sending GET request to '%s'.", address)
	}
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return
	}
	c.setAgent(request)
	c.setCredentials(request)
	c.setAccept(request)
	response, err := c.client.Do(request)
	if err != nil {
		return
	}
	body := response.Body
	defer body.Close()

	// Read the response body:
	output, err = ioutil.ReadAll(body)
	if err != nil {
		return
	}
	if glog.V(2) {
		glog.Infof("Response body:\n%s", c.indent(output))
	}

	return
}

func (c *Connection) authenticatedPost(path string, query url.Values, input interface{}, output interface{}) error {
	err := c.ensureToken()
	if err != nil {
		return err
	}
	return c.post(path, query, input, output)
}

func (c *Connection) post(path string, query url.Values, input interface{}, output interface{}) error {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	outputBytes, err := c.rawPost(path, query, inputBytes)
	if err != nil {
		return err
	}
	return json.Unmarshal(outputBytes, output)
}

func (c *Connection) rawPost(path string, query url.Values, input []byte) (output []byte, err error) {
	// Post the input bytes:
	address := c.makeUrl(path, query)
	if glog.V(2) {
		glog.Infof("Sending POST request to '%s'.", address)
		glog.Infof("Request body:\n%s", c.indent(input))
	}
	buffer := bytes.NewBuffer(input)
	request, err := http.NewRequest(http.MethodPost, address, buffer)
	if err != nil {
		return
	}
	c.setAgent(request)
	c.setCredentials(request)
	c.setContentType(request)
	c.setAccept(request)
	response, err := c.client.Do(request)
	if err != nil {
		return
	}
	body := response.Body
	defer body.Close()

	// Read the response body:
	output, err = ioutil.ReadAll(body)
	if err != nil {
		return
	}
	if glog.V(2) {
		glog.Infof("Response body:\n%s", c.indent(output))
	}

	return
}

func (c *Connection) setAgent(request *http.Request) {
	request.Header.Set("User-Agent", c.agent)
}

func (c *Connection) setCredentials(request *http.Request) {
	if c.token != "" {
		request.Header.Set("Authorization", "Token "+c.token)
	}
}

func (c *Connection) setContentType(request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
}

func (c *Connection) setAccept(request *http.Request) {
	request.Header.Set("Accept", "application/json")
}

func (c *Connection) indent(data []byte) []byte {
	buffer := new(bytes.Buffer)
	err := json.Indent(buffer, data, "", "  ")
	if err != nil {
		return data
	}
	return buffer.Bytes()
}
