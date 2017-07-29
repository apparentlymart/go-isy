package isy

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
)

func TestClientFormatRequest(t *testing.T) {
	client, err := NewClient(&ClientConfig{
		BaseURL:  "http://127.0.0.1/",
		Username: "test",
		Password: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := client.formatRequest(&testSOAPMessage{})
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	req.Write(buf)

	got := strings.Replace(strings.TrimSpace(buf.String()), "\r", "", -1)
	want := strings.TrimSpace(`
POST /services HTTP/1.1
Host: 127.0.0.1
User-Agent: go-isy
Content-Length: 226
Authorization: Basic dGVzdDp0ZXN0
Content-Type: text/xml; charset="utf-8"
Soapaction: urn:udi-com:service:X_Insteon_Lighting_Service:1#TestMessage

<Envelope xmlns="http://www.w3.org/2003/05/soap-envelope">
  <Body xmlns="http://www.w3.org/2003/05/soap-envelope">
    <TestMessage xmlns="urn:udi-com:service:X_Insteon_Lighting_Service:1"></TestMessage>
  </Body>
</Envelope>
`)

	if got != want {
		t.Errorf("wrong result\n%s", diff.LineDiff(want, got))
	}
}
