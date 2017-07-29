package isy

import (
	"encoding/xml"
	"reflect"
	"strings"
)

type soapMessage struct {
	Action string
	Body   []byte
}

type soapEnvelope struct {
	XMLName string `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Body    soapBody
}

type soapBody struct {
	XMLName string `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	Content interface{}
}

func makeSOAPMessage(obj interface{}) (soapMessage, error) {
	action := getSOAPAction(obj)
	body, err := formatSOAPEnvelope(obj)
	return soapMessage{
		Action: action,
		Body:   body,
	}, err
}

func formatSOAPEnvelope(obj interface{}) ([]byte, error) {
	env := soapEnvelope{
		Body: soapBody{
			Content: obj,
		},
	}
	return xml.MarshalIndent(&env, "", "  ")
}

func getSOAPAction(obj interface{}) string {
	ty := reflect.TypeOf(obj)
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	if ty.Kind() != reflect.Struct {
		return ""
	}

	nameField, exists := ty.FieldByName("XMLName")
	if !exists {
		return ""
	}

	return strings.Replace(nameField.Tag.Get("xml"), " ", "#", 1)
}
