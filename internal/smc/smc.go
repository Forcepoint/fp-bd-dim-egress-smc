package smc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"main/internal/channel"
	"main/internal/structs"
	"main/internal/util"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type ListParams struct {
	UpdateType       structs.UpdateType
	items            []string
	item             string
	safe             bool
	batchID          int
	listType         ListType
	safelistName     LocalListName
	blocklistName    LocalListName
	safelistComment  Comment
	blocklistComment Comment
}

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

func (s *Session) UpdateLists(params ListParams) {
	switch params.UpdateType {
	case structs.ADD:
		created, err := s.createList(params)
		if err != nil {
			logrus.Error(err)
			go s.updateBatchStatus(params.batchID, structs.Failed)
			return
		}

		// URL Lists require values to be passed for list creation, so in that case we can just return here
		// as the next steps are unnecessary, but they will be run on subsequent updates.
		if created && params.listType == URLListType {
			go s.updateBatchStatus(params.batchID, structs.Success)
			return
		}

		err = s.updateList(params)
		if err != nil {
			logrus.Error(err)
			go s.updateBatchStatus(params.batchID, structs.Failed)
			return
		}
	case structs.DELETE:
		err := s.updateList(params)
		if err != nil {
			logrus.Error(err)
			go s.updateBatchStatus(params.batchID, structs.Failed)
			return
		}
	default:
		// Build update to send to controller
		go s.updateBatchStatus(params.batchID, structs.Success)
	}
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
		// Retrieve requests.
		request := <-channel.Requests

		switch request.UpdateType {
		case structs.ADD:
			var ips []string
			var urls []string
			var snorts []string

			for _, item := range request.Items {
				switch item.Type {
				case structs.IP, structs.RANGE:
					ips = append(ips, item.Value)
				case structs.URL, structs.DOMAIN:
					urls = append(urls, item.Value)
				case structs.SNORT:
					snorts = append(snorts, item.Value)
				}
			}

			// Send request to add to smc ip lists.
			if len(ips) > 0 {
				params := ListParams{
					UpdateType:       request.UpdateType,
					items:            ips,
					safe:             request.SafeList,
					batchID:          request.BatchID,
					listType:         IPListType,
					safelistName:     IPSafelist,
					blocklistName:    IPBlocklist,
					safelistComment:  IPSafelistComment,
					blocklistComment: IPBlocklistComment,
				}
				session.UpdateLists(params)
			}

			if len(urls) > 0 {
				params := ListParams{
					UpdateType:       request.UpdateType,
					items:            urls,
					safe:             request.SafeList,
					batchID:          request.BatchID,
					listType:         URLListType,
					safelistName:     URLSafelist,
					blocklistName:    URLBlocklist,
					safelistComment:  URLSafelistComment,
					blocklistComment: URLBlocklistComment,
				}
				session.UpdateLists(params)
			}

			if len(snorts) > 0 {
				exportDirPath, err := session.RetrieveGlobalSnortConfig()

				if err != nil {
					logrus.Error("Error in retrieving global snort config: ", err)
				}

				snortFileName := fmt.Sprintf("%s%s%s", "dim_snorts_", strconv.FormatInt(time.Now().Unix(), 10), ".config")

				err = util.SaveListToAfile(exportDirPath, snortFileName, snorts)

				if err != nil {
					logrus.Error("Error in saving new snorts: ", err)
				}

				err = util.SmcRulesInclude(exportDirPath, snortFileName)

				if err != nil {
					logrus.Error("Error in adding the snort file to the rule include file", err)
				}

				err = session.ImportGlobalSnortConfig(exportDirPath)

				if err != nil {
					logrus.Error("Error in importing global snort config: ", err)
				}
			}

			// clean up resources
			ips = nil
			urls = nil
			snorts = nil

		case structs.DELETE:
			switch request.Item.Type {
			case structs.URL, structs.DOMAIN:
				params := ListParams{
					UpdateType:       request.UpdateType,
					item:             request.Item.Value,
					safe:             request.SafeList,
					batchID:          request.BatchID,
					listType:         URLListType,
					safelistName:     URLSafelist,
					blocklistName:    URLBlocklist,
					safelistComment:  URLSafelistComment,
					blocklistComment: URLBlocklistComment,
				}
				session.UpdateLists(params)
			}
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

// ApiVersion Representation of data returned from SMC API version check
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
