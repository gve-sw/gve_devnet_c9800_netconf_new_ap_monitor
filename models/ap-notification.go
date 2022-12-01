package models

import "encoding/xml"

type ApNotification struct {
	XMLName    xml.Name   `xml:"notification"`
	EventTime  string     `xml:"eventTime"`
	PushUpdate PushUpdate `xml:"push-update"`
}

type PushUpdate struct {
	XMLName        xml.Name         `xml:"push-update"`
	SubscriptionID int64            `xml:"subscription-id"`
	Content        DataStoreContent `xml:"datastore-contents-xml"`
}

type DataStoreContent struct {
	XMLName          xml.Name         `xml:"datastore-contents-xml"`
	ApGlobalOperData ApGlobalOperData `xml:"ap-global-oper-data"`
}

type ApGlobalOperData struct {
	XMLName     xml.Name      `xml:"ap-global-oper-data"`
	ApJoinStats []ApJoinStats `xml:"ap-join-stats"`
}

type ApJoinStats struct {
	XMLName    xml.Name   `xml:"ap-join-stats"`
	WtpMac     string     `xml:"wtp-mac"`
	ApJoinInfo ApJoinInfo `xml:"ap-join-info"`
}

type ApJoinInfo struct {
	XMLName       xml.Name `xml:"ap-join-info"`
	ApIpAddr      string   `xml:"ap-ip-addr"`
	ApEthernetMac string   `xml:"ap-ethernet-mac"`
	ApName        string   `xml:"ap-name"`
	IsJoined      string   `xml:"is-joined"`
	LastErrorType string   `xml:"last-error-type"`
}
