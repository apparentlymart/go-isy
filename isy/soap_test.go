package isy

import (
	"strings"
	"testing"
)

func TestFormatSOAPEnvelope(t *testing.T) {
	got, err := formatSOAPEnvelope(&testSOAPMessage{})
	want := `
<Envelope xmlns="http://www.w3.org/2003/05/soap-envelope">
  <Body xmlns="http://www.w3.org/2003/05/soap-envelope">
    <TestMessage xmlns="urn:udi-com:service:X_Insteon_Lighting_Service:1"></TestMessage>
  </Body>
</Envelope>
`
	if err != nil {
		t.Fatalf(err.Error())
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(want) {
		t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
	}
}

func TestGetSOAPAction(t *testing.T) {
	got := getSOAPAction(&testSOAPMessage{})
	want := "urn:udi-com:service:X_Insteon_Lighting_Service:1#TestMessage"

	if got != want {
		t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
	}
}

type testSOAPMessage struct {
	XMLName string `xml:"urn:udi-com:service:X_Insteon_Lighting_Service:1 TestMessage"`
}
