package models

type Configuration struct {
	WirelessControllers []WirelessController `json:"wireless-controllers"`
	Netconf             NetconfConfig        `json:"netconf"`
	MQTTConfig          MQTTConfig           `json:"mqtt"`
	MongoConfig         Mongo                `json:"mongo"`
}

type WirelessController struct {
	Name string `json:"name"`
	Port int32  `json:"port"`
}

type NetconfConfig struct {
	XpathFilter        string `json:"xpath-filter"`
	SubscriptionPeriod int    `json:"subscription-period"`
}

type MQTTConfig struct {
	Broker   string `json:"broker"`
	Port     int32  `json:"port"`
	ClientId string `json:"client-id"`
	Topic    string `json:"topic"`
}

type Mongo struct {
	Url string `json:"url"`
}
