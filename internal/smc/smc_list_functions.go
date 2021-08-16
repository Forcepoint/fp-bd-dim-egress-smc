package smc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"main/internal/structs"
	"main/internal/util"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var ErrPatchNotSupported = errors.New("patch not supported for this type")

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

func (s *Session) updateList(params ListParams) error {
	if params.UpdateType == structs.DELETE && params.listType == IPListType {
		return ErrPatchNotSupported
	}
	var id string
	if params.safe {
		id = viper.GetString(params.safelistName)
	} else {
		id = viper.GetString(params.blocklistName)
	}

	url := fmt.Sprintf("%s:%s/%s/%s/%s", s.Host, s.Port, s.Version, params.listType, id)

	// Use an interface here for updateObject as the types for IP List and URL List are different
	var updateObject interface{}
	// IP List is POST, URL is PATCH
	var updateMethod string
	// 202 Accepted for IP list, 200 OK for URL list
	var successfulUpdateStatusCode int

	var headers = map[string]string{"Content-Type": "application/json"}

	// We need to switch on the list type as the update methods are different for each
	switch params.listType {
	case IPListType:
		updateMethod = http.MethodPost
		successfulUpdateStatusCode = http.StatusAccepted

		resp, err := s.retrieveList(params)
		if err != nil {
			logrus.Error(err)
			go s.updateBatchStatus(params.batchID, structs.Failed)
		}
		// Set ETAG
		headers["If-Match"] = resp.Header.Get("ETag")
		list := structs.SMCList{}
		json.NewDecoder(resp.Body).Decode(&list)

		// We need to append an extra identifier for IP list types
		url = fmt.Sprintf("%s/%s", url, IPAddressListType)

		// Append the items to the IP list
		list.IPList = append(list.IPList, params.items...)
		updateObject = list
	case URLListType:
		// The URL list update is a PATCH instead of a POST like the IP List
		updateMethod = http.MethodPatch
		successfulUpdateStatusCode = http.StatusOK

		headers["Accept"] = "application/json-patch+json"
		headers["If-Match"] = "*"
		var patchList []structs.SMCPatch
		if params.UpdateType == structs.DELETE {
			resp, err := s.retrieveList(params)
			if err != nil {
				logrus.Error(err)
				go s.updateBatchStatus(params.batchID, structs.Failed)
			}
			list := structs.SMCList{}
			json.NewDecoder(resp.Body).Decode(&list)

			var indexes []int

			for i, val := range list.URLEntry {
				if val == params.item {
					indexes = append(indexes, i+1)
				}
			}

			for _, val := range indexes {
				patchList = append(patchList, structs.SMCPatch{
					Op:    structs.DELETE,
					Path:  fmt.Sprintf("/url_entry/%d", val),
					Value: "",
				})
			}
		} else if params.UpdateType == structs.ADD {
			for _, val := range params.items {
				patchList = append(patchList, structs.SMCPatch{
					Op:    structs.ADD,
					Path:  "/url_entry/1",
					Value: val,
				})
			}
		}

		updateObject = patchList
	}

	// Convert list update object to json.
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(updateObject)

	if err != nil {
		return errors.Wrap(err, "There was an error marshalling the list update payload")
	}

	_, resp, err := s.buildRequest(
		url,
		updateMethod,
		headers,
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

func (s *Session) RetrieveGlobalSnortConfig() (string, error) {

	snortTempDirPath := filepath.Join("/temp", "snort")

	// exists
	if _, err := os.Stat(snortTempDirPath); !os.IsNotExist(err) {
		// clear it
		err := os.RemoveAll(snortTempDirPath)
		if err != nil {
			return "", errors.Wrap(err, "error removing temp snort export dir content")
		}
	}

	// create export dir
	snortExportTempDirPath := filepath.Join(snortTempDirPath, "export")
	err := os.MkdirAll(snortExportTempDirPath, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "error creating a temp dir for the snorts export")
	}

	snortUrl, err := s.getSnortUrl()

	if err != nil {
		return "", err
	}

	if snortUrl == "" {
		return "", errors.New("error snortUrl is empty")
	}

	snortExportUrl := fmt.Sprintf("%s/snort_configuration_file/export", snortUrl)
	_, resp, err := s.buildRequest(snortExportUrl, http.MethodGet, map[string]string{"Content-Type": "application/json", "Accept": "application/json"}, nil)

	if err != nil {
		return "", errors.Wrap(err, "error exporting snort configuration file")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {

		// Create the file
		globalSnortsZipPath := filepath.Join(snortTempDirPath, "global-smc-snorts.zip")
		out, err := os.Create(globalSnortsZipPath)
		if err != nil {
			return "", errors.Wrap(err, "creating global-smc-snorts.zip")
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return "", errors.Wrap(err, "writing resp.Body into global-snorts.zip")
		}

		_, err = util.Unzip(globalSnortsZipPath, snortExportTempDirPath)
		if err != nil {
			return "", errors.Wrap(err, "unzipping global-smc-snorts.zip")
		}

	} else if resp.StatusCode != http.StatusNotFound {
		return "", errors.New(fmt.Sprintf("error exporting global snort configurations, status code is: %s", resp.StatusCode))
	}

	err = util.CreateFileIfNotExist(filepath.Join(snortExportTempDirPath, "rules_include.config"))

	if err != nil {
		return "", errors.Wrap(err, "creating rules_include.config")
	}

	return snortExportTempDirPath, nil
}

func (s *Session) getSystemProperties() (structs.SMCSystemProperties, error) {
	url := fmt.Sprintf("%s:%s/%s/system/system_properties", s.Host, s.Port, s.Version)
	_, resp, err := s.buildRequest(url, http.MethodGet, map[string]string{"Content-Type": "application/json"}, nil)

	if err != nil {
		return structs.SMCSystemProperties{}, errors.Wrap(err, "There was an error in system_properties request")
	}

	result := structs.SMCSystemProperties{}
	err = json.NewDecoder(resp.Body).Decode(&result)

	if err != nil {
		return structs.SMCSystemProperties{}, errors.Wrap(err, "There was an error in decoding system_properties request")
	}

	return result, nil
}

func (s *Session) getSnortUrl() (string, error) {

	result, err := s.getSystemProperties()

	if err != nil {
		return "", err
	}

	snortUrl := ""

	for _, property := range result.Results {
		switch property.Name {
		case "snort_global_config":
			snortUrl = property.Href
			break
		}
	}

	return snortUrl, nil
}

func (s *Session) ImportGlobalSnortConfig(exportDirPath string) error {

	zipDestPath := filepath.Join("/temp", "snort")
	zipDirPath, err := util.Zip(exportDirPath, zipDestPath, "global-snorts")
	if err != nil {
		return errors.Wrap(err, "zipping global-snorts.zip")
	}

	file, err := os.Open(zipDirPath)
	if err != nil {
		return errors.Wrap(err, "error in oppening global-snorts.zip for upload")
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return errors.Wrap(err, "error in preparing global-snorts.zip for upload")
	}
	io.Copy(part, file)
	writer.Close()

	snortUrl, err := s.getSnortUrl()
	if err != nil {
		return err
	}
	if snortUrl == "" {
		return errors.New("error snortUrl is empty")
	}

	snortImportUrl := fmt.Sprintf("%s/snort_configuration_file/import", snortUrl)
	_, resp, err := s.buildRequest(snortImportUrl, http.MethodPost, map[string]string{"Content-Type": writer.FormDataContentType()}, body)
	if err != nil {
		return errors.Wrap(err, "error importing snort configuration file")
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New(fmt.Sprintf("error importing snort global configurations, status code is: %s", resp.StatusCode))
	}

	return nil
}
