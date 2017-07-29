package isy

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/davecgh/go-spew/spew"
)

var servicePath *url.URL

// Client represents a connection to a particular ISY.
type Client struct {
	*client
}

type client struct {
	ServiceURL string
	Username   string
	Password   string
}

// ClientConfig is used to instantiate a client using NewClient.
type ClientConfig struct {
	BaseURL  string
	Username string
	Password string
}

// NewClient creates a new client with the given configuration.
func NewClient(config *ClientConfig) (Client, error) {
	urlObj, err := url.Parse(config.BaseURL)
	if err != nil {
		return Client{}, fmt.Errorf("invalid base URL: %s", err)
	}
	serviceURLObj := urlObj.ResolveReference(servicePath)

	return Client{
		&client{
			ServiceURL: serviceURLObj.String(),
			Username:   config.Username,
			Password:   config.Password,
		},
	}, nil
}

func (c *client) GetAllFunctions() ([]*Function, error) {
	body, err := c.request(getAllD2DReq{})
	if err != nil {
		return nil, err
	}

	dec := xml.NewDecoder(bytes.NewReader(body))
	var start *xml.StartElement
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if open, isOpen := tok.(xml.StartElement); isOpen {
			if open.Name.Local == "triggers" {
				start = &open
				break
			}
		}
	}

	if start == nil {
		return nil, errors.New("'triggers' element not found in response")
	}

	var raw triggersRaw
	err = dec.DecodeElement(&raw, start)
	if err != nil {
		return nil, err
	}

	spew.Dump(raw)

	return nil, nil
}

func (c *client) request(obj interface{}) ([]byte, error) {
	req, err := c.formatRequest(obj)
	if err != nil {
		return nil, err
	}

	httpC := &http.Client{}
	resp, err := httpC.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)

	return body, nil
}

func (c *client) formatRequest(obj interface{}) (*http.Request, error) {
	msg, err := makeSOAPMessage(obj)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.ServiceURL, bytes.NewReader(msg.Body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.ContentLength = int64(len(msg.Body))
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("SOAPACTION", msg.Action)
	req.Header.Set("User-Agent", "go-isy")
	return req, nil
}

type getAllD2DReq struct {
	XMLName string `xml:"urn:udi-com:service:X_Insteon_Lighting_Service:1 GetAllD2D"`
}

func init() {
	servicePath, _ = url.Parse("./services")
}
