package models

type ApMQTTMessage struct {
	WLC  string `json:"wlc"`
	MAC  string `json:"mac"`
	TIME string `json:"time"`
}
