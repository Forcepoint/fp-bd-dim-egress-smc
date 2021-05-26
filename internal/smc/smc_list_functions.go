package smc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"main/internal/structs"
	"net/http"
	"strings"
)

func (s *Session) createList(params ListParams) (bool, error) {
	var name, comment string
	switch params.safe {
	case true:
		name = params.safelistName
		comment = params.safelistComment
	case false:
		name = params.blocklistName
		comment = params.blocklistComment
	}
	if viper.IsSet(name) {
		return false, nil
	}
MethodStart:
	// Create list
	createList := structs.SMCList{
		Name:     name,
		Comment:  comment,
		URLEntry: nil,
		IPList:   nil,
	}

	// URL lists need to have initial values to create the list, IP Lists don't
	if params.listType == URLListType {
		createList.URLEntry = params.items
	}

	// Convert list creation object to json.
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(createList)

	url := fmt.Sprintf("%s:%s/%s/%s", s.Host, s.Port, s.Version, params.listType)
	status, resp, err := s.buildRequest(url, http.MethodPost, map[string]string{"Content-Type": "application/json"}, b)

	if err != nil {
		return false, errors.Wrap(err, "There was an error creating the list creation request")
	}

	if status == http.StatusUnauthorized {
		s.Login()
		goto MethodStart
	}

	// Handle response code.
	if status != http.StatusCreated {
		return false, errors.New(fmt.Sprintf("the list creation request was unsuccessful. Status Code: %d", status))
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
	return true, nil
}

func (s *Session) retrieveList(params ListParams) (*http.Response, error) {
MethodStart:
	var id string
	if params.safe {
		id = viper.GetString(params.safelistName)
	} else {
		id = viper.GetString(params.blocklistName)
	}

	url := fmt.Sprintf("%s:%s/%s/%s/%s", s.Host, s.Port, s.Version, params.listType, id)

	// We need to append an extra identifier for IP list types
	if params.listType == IPListType {
		url = fmt.Sprintf("%s/%s", url, IPAddressListType)
	}

	status, resp, err := s.buildRequest(
		url,
		http.MethodGet,
		map[string]string{"Content-Type": "application/json", "Accept": "application/json"},
		nil)

	if err != nil {
		return nil, errors.Wrap(err, "There was an error building the list retrieval request.")
	}

	if status == http.StatusUnauthorized {
		s.Login()
		goto MethodStart
	}

	// Handle response code.
	if status != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("The list retrieval request was unsuccessful. Status Code: %d", status))
	}

	return resp, nil
}

func (s *Session) updateList(resp *http.Response, params ListParams) error {
	var id string
	if params.safe {
		id = viper.GetString(params.safelistName)
	} else {
		id = viper.GetString(params.blocklistName)
	}

	// Get elements and ETAG
	etag := resp.Header.Get("ETag")
	list := structs.SMCList{}
	json.NewDecoder(resp.Body).Decode(&list)

	url := fmt.Sprintf("%s:%s/%s/%s/%s", s.Host, s.Port, s.Version, params.listType, id)

	var updateMethod string
	var successfulUpdateStatusCode int
	switch params.listType {
	case IPListType:
		// We need to append an extra identifier for IP list types
		url = fmt.Sprintf("%s/%s", url, IPAddressListType)
		updateMethod = http.MethodPost
		successfulUpdateStatusCode = http.StatusAccepted
		list.IPList = append(list.IPList, params.items...)
	case URLListType:
		// The URL list update is a PUT instead of a POST like the IP List
		successfulUpdateStatusCode = http.StatusOK
		updateMethod = http.MethodPut
		list.URLEntry = append(list.URLEntry, params.items...)
	}

	// Convert list update object to json.
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(list)

	_, resp, err := s.buildRequest(
		url,
		updateMethod,
		map[string]string{"Content-Type": "application/json", "If-Match": etag},
		b)

	// Build list update request.
	if err != nil {
		return errors.Wrap(err, "There was an error creating the list update request")
	}

	// Handle response code.
	if resp.StatusCode != successfulUpdateStatusCode {
		return errors.New(fmt.Sprintf("The list update request was unsuccessful. Status Code: %d", resp.StatusCode))
	}

	return nil
}
