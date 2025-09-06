package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const BUFFER = 1024 * 1024 * 10

type Session struct {
	instanceUrl string
	token       string
}

func (session *Session) post(url string, contentType string, size int, body io.Reader) (*http.Response, error) {
	fmt.Println("POST", url, "with content type", contentType)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Auth", session.token)
	req.Header.Set("Upload-length", strconv.Itoa(size))
	client := &http.Client{}
	return client.Do(req)
}

func (session *Session) head(url string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth", session.token)
	client := &http.Client{}
	return client.Do(req)
}

func (session *Session) chunkedPatch(url string, offset string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/offset+octet-stream")
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("X-Auth", session.token)
	req.Header.Set("Upload-Offset", offset)
	client := &http.Client{}
	return client.Do(req)
}

func (session *Session) Share(fileUrl string) (string, error) {
	postBody, _ := json.Marshal(map[string]string{
		"expires":  "30",
		"password": "",
		"unit":     "days",
	})
	res, err := session.post(fileUrl, "text/plain", 0, bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalln("Error during share post:", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var mapping map[string]interface{}
	err = json.Unmarshal(body, &mapping)
	if err != nil {
		log.Fatalln(err)
	}

	downloadUrl := session.instanceUrl + "/api/public/dl/" + mapping["hash"].(string) + url.QueryEscape(mapping["path"].(string))

	fmt.Println("Download link:", downloadUrl)
	return downloadUrl, nil
}

func (session *Session) Upload(filePath string) (string, error) {
	fileName := filepath.Base(filePath)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.New("Error opening file: " + err.Error())
	}
	defer file.Close()

	// Get file stats
	fileInfo, err := file.Stat()
	if err != nil {
		return "", errors.New("Error getting file stats: " + err.Error())
	}
	totalBytes := fileInfo.Size()

	// Calculate sha256 hash
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", errors.New("Error computing sha256: " + err.Error())
	}
	hash := hex.EncodeToString(hasher.Sum(nil))
	file.Seek(0, 0)

	remotePath := hash + "/" + fileName

	res, err := session.post(session.instanceUrl+"/api/tus/"+remotePath+"?override=false", "application/json", int(totalBytes), nil)
	if err != nil {
		return "", errors.New("Error during upload post: " + err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		// print body
		body, _ := io.ReadAll(res.Body)
		log.Println("Upload post response body:", string(body))
		// return error with status code
		return "", errors.New("During upload post: status code " + res.Status + " != 201")
	}

	res, err = session.head(session.instanceUrl + "/api/tus/" + remotePath)
	if err != nil {
		return "", errors.New("Error during upload head: " + err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("During upload head: status code " + res.Status + " != 200")
	}

	offset := "0"
	for {
		buffer := make([]byte, BUFFER)
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", errors.New("Error reading file: " + err.Error())
		}

		totalBytes -= int64(n)
		fmt.Println("Uploading", n, "bytes ("+strconv.Itoa(int(totalBytes)), "bytes remaining)")

		res, err := session.chunkedPatch(session.instanceUrl+"/api/tus/"+remotePath, offset, bytes.NewReader(buffer[:n]))
		if err != nil {
			return "", errors.New("Error during upload patch: " + err.Error())
		}
		if res.StatusCode != 204 {
			return "", errors.New("During upload patch: status code " + res.Status + " != 204")
		}
		offset = res.Header.Get("upload-offset")
		res.Body.Close()
	}

	return session.Share(session.instanceUrl + "/api/share/" + remotePath)
}

func Login(instanceUrl, username, password string) (*Session, error) {
	postBody, _ := json.Marshal(map[string]string{
		"username":  username,
		"recaptcha": "",
		"password":  password,
	})

	resp, err := http.Post(instanceUrl+"/api/login", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalln(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	sb := string(body)
	return &Session{instanceUrl: instanceUrl, token: sb}, nil
}
