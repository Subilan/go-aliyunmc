package models

import "gorm.io/gorm"

type Instance struct {
	gorm.Model
	InstanceId   string `gorm:"not null" json:"instanceId"`
	InstanceType string `gorm:"not null" json:"instanceType"`
	RegionId     string `gorm:"not null" json:"regionId"`
	ZoneId       string `gorm:"not null" json:"zoneId"`
	VSwitchId    string `gorm:"not null" json:"vSwitchId"`
	Ip           string `gorm:"default:null" json:"ip"`
	IsDeployed   bool   `gorm:"default:false" json:"isDeployed"`
}
