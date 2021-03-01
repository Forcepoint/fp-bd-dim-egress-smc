package smc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"main/internal/channel"
	"main/internal/structs"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

type Session struct {
	Host     string
	Port     string
	Key      string
	Version  string
	client   *http.Client
	LoggedIn bool
}

func NewSMCSession(host, port, key string) (*Session, int, error) {
	if host == "" || port == "" || key == "" {
		return &Session{}, http.StatusInternalServerError, errors.New("missing parameters for SMC login")
	}
	// Create client and jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	client := &http.Client{
		Jar:     jar,
		Timeout: 120 * time.Second,
	}
	sesh := Session{
		Host:    host,
		Port:    port,
		Key:     key,
		Version: "",
		client:  client,
	}

	apiVersion, err := sesh.GetLatestApiVersion()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	//Set that to be the version in the session
	sesh.Version = apiVersion.Rel

	status := sesh.Login()

	sesh.LoggedIn = status == http.StatusOK

	return &sesh, status, nil
}

func (s *Session) Login() int {
	// Create login object.
	login := structs.Login{
		Domain:            "Shared Domain",
		AuthenticationKey: s.Key,
	}

	// Convert login object to json.
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(login)

	url := fmt.Sprintf("%s:%s/%s/%s", s.Host, s.Port, s.Version, "login")
	status, _, err := s.buildRequest(url, http.MethodPost, map[string]string{"Content-Type": "application/json"}, b)

	if err != nil {
		logrus.Error("Error logging in to smc: ", err)
	}

	s.LoggedIn = status == http.StatusOK

	return status
}

func (s *Session) AddToBlockList(batch structs.Request) {
	// Check if IP List Exists
	if batch.SafeList {
		if !viper.IsSet("dim_safelist") {
			err := s.createList("dim_safelist", "Safelist imported from the Dynamic Intelligence Manager.")
			if err != nil {
				logrus.Error(err)
				s.updateBatchStatus(batch.BatchID, "failed")
				return
			}
		}
	} else {
		if !viper.IsSet("dim_blocklist") {
			err := s.createList("dim_blocklist", "Blocklist imported from the Dynamic Intelligence Manager.")
			if err != nil {
				logrus.Error(err)
				s.updateBatchStatus(batch.BatchID, "failed")
				return
			}
		}
	}

	resp, err := s.retrieveList(batch.SafeList)

	if err != nil {
		logrus.Error(err)
		s.updateBatchStatus(batch.BatchID, "failed")
		return
	}

	err = s.updateList(resp, batch)
	if err != nil {
		logrus.Error(err)
		s.updateBatchStatus(batch.BatchID, "failed")
		return
	}

	// Build update to send to controller
	s.updateBatchStatus(batch.BatchID, "success")
}

func (s Session) Logout() {
	// Build logout request.
	url := fmt.Sprintf("%s:%s/%s/logout", s.Host, s.Port, s.Version)
	_, resp, err := s.buildRequest(url, http.MethodPut, map[string]string{}, nil)

	if err != nil {
		logrus.Error(err)
		return
	}

	// Handle response code.
	if resp.StatusCode != http.StatusNoContent {
		logrus.Error("The logout attempt was unsuccessful. Status Code: ", resp.StatusCode)
		return
	}
}

func (s *Session) updateBatchStatus(id int, status string) {
	// Retrieve environment variables.
	controllerSvcName := os.Getenv("CONTROLLER_SVC_NAME")
	controllerPort := os.Getenv("CONTROLLER_PORT")
	moduleSvcName := os.Getenv("MODULE_SVC_NAME")
	token := os.Getenv("INTERNAL_TOKEN")

	// Build update JSON
	update := structs.Update{
		ServiceName:   moduleSvcName,
		Status:        status,
		UpdateBatchId: id,
	}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(update)

	url := fmt.Sprintf("http://%s:%s/internal/update", controllerSvcName, controllerPort)
	_, resp, err := s.buildRequest(url, http.MethodPost, map[string]string{"x-internal-token": token, "Content-Type": "application/json"}, b)

	if err != nil {
		logrus.Error("There was an error sending the update to the controller: ", err)
		return
	}

	if resp.StatusCode != http.StatusAccepted {
		logrus.Error("There was an error sending the update to the controller. Status code received: ", resp.StatusCode)
	}
}

func HandleRequests(session *Session) {
	viper.OnConfigChange(func(in fsnotify.Event) {
		session, _, _ = NewSMCSession(
			viper.GetString("smc_endpoint"),
			viper.GetString("smc_port"),
			viper.GetString("smc_api_key"))
	})

	for {
		if session.LoggedIn {
			// Retrieve requests.
			request := <-channel.Requests

			// Send request to add to smc ip lists.
			session.AddToBlockList(request)
		}
	}
}

func (s *Session) buildRequest(url, method string, headers map[string]string, data io.Reader) (int, *http.Response, error) {
	// Build login request.
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Send client request.
	resp, err := s.client.Do(req)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	return resp.StatusCode, resp, nil
}

func (s *Session) createList(name, comment string) error {
MethodStart:
	// Create list
	createList := structs.List{
		Name:    name,
		Comment: comment,
		IPList:  nil,
	}

	// Convert list creation object to json.
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(createList)

	url := fmt.Sprintf("%s:%s/%s/%s", s.Host, s.Port, s.Version, "elements/ip_list")
	status, resp, err := s.buildRequest(url, http.MethodPost, map[string]string{"Content-Type": "application/json"}, b)

	if err != nil {
		return errors.Wrap(err, "There was an error creating the blocklist creation request")
	}

	if status == http.StatusUnauthorized {
		s.Login()
		goto MethodStart
	}

	// Handle response code.
	if status != http.StatusCreated {
		return errors.New(fmt.Sprintf("the list creation request was unsuccessful. Status Code: %d", status))
	}

	// Store list ID
	header := resp.Header.Get("Location")
	idIndex := strings.LastIndex(header, "/")
	listId := header[idIndex+1:]

	viper.Set(name, listId)
	if err := viper.WriteConfig(); err != nil {
		logrus.Error("error writing list ID to config", err)
	}

	fmt.Println("successfully created list with Status Code: ", resp.StatusCode)
	return nil
}

func (s *Session) retrieveList(safe bool) (*http.Response, error) {
MethodStart:

	var id string
	if safe {
		id = viper.GetString("dim_safelist")
	} else {
		id = viper.GetString("dim_blocklist")
	}

	url := fmt.Sprintf("%s:%s/%s/%s", s.Host, s.Port, s.Version, fmt.Sprintf("elements/ip_list/%s/ip_address_list", id))
	status, resp, err := s.buildRequest(
		url,
		http.MethodGet,
		map[string]string{"Content-Type": "application/json", "Accept": "application/json"},
		nil)

	if err != nil {
		return nil, errors.Wrap(err, "There was an error building the blocklist retrieval request.")
	}

	if status == http.StatusUnauthorized {
		s.Login()
		goto MethodStart
	}

	// Handle response code.
	if status != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("The blocklist retrieval request was unsuccessful. Status Code: %d", status))
	}

	return resp, nil
}

func (s *Session) updateList(resp *http.Response, batch structs.Request) error {

	var id string
	if batch.SafeList {
		id = viper.GetString("dim_safelist")
	} else {
		id = viper.GetString("dim_blocklist")
	}

	// Get elements and ETAG
	etag := resp.Header.Get("ETag")
	ipList := structs.List{}
	json.NewDecoder(resp.Body).Decode(&ipList)

	// Append to IP List
	for _, element := range batch.Items {
		ipList.IPList = append(ipList.IPList, element.Value)
	}

	// Convert blocklist update object to json.
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(ipList)

	url := fmt.Sprintf("%s:%s/%s/%s", s.Host, s.Port, s.Version, fmt.Sprintf("elements/ip_list/%s/ip_address_list", id))
	_, resp, err := s.buildRequest(
		url,
		http.MethodPost,
		map[string]string{"Content-Type": "application/json", "If-Match": etag},
		b)

	// Build blocklist update request.
	if err != nil {
		return errors.Wrap(err, "There was an error creating the blocklist update request")
	}

	// Handle response code.
	if resp.StatusCode != http.StatusAccepted {
		return errors.New(fmt.Sprintf("The blocklist update request was unsuccessful. Status Code: %d", resp.StatusCode))
	}

	return nil
}

func (s *Session) GetLatestApiVersion() (*ApiVersion, error) {
	status, resp, err := s.buildRequest(
		fmt.Sprintf("%s:%s/api", s.Host, s.Port),
		http.MethodGet,
		map[string]string{"Content-Type": "application/json"},
		http.NoBody)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, errors.New("bad status code")
	}
	var apiVersions ApiVersionWrapper
	err = json.NewDecoder(resp.Body).Decode(&apiVersions)
	if err != nil {
		return nil, err
	}
	if len(apiVersions.Version) == 0 {
		return nil, errors.New("returned API versions were empty")
	}
	// return the last item in the list which should be the most recent API version
	return &apiVersions.Version[len(apiVersions.Version)-1], nil
}

// Representation of data returned from SMC API version check
//{
//"href": "http://146.59.179.241:8082/6.9/api",
//"rel": "6.9"
//}
type ApiVersion struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type ApiVersionWrapper struct {
	Version []ApiVersion `json:"version"`
}
