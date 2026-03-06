package config

import (
	"os"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

func TestGenConfig(t *testing.T) {
	out, err := toml.Marshal(&Config{
		Base: BaseConfig{
			Expose:    33761,
			JwtSecret: "",
			Cors: CorsConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
				AllowHeaders:     []string{"Content-Length", "Content-Type", "Authorization", "Last-Event-Id"},
				AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			},
			Autotls: AutotlsConfig{
				Enabled: false,
			},
		},
		Aliyun: AliyunConfig{
			RegionId:        "",
			AccessKeyId:     "",
			AccessKeySecret: "",
			Ecs: AliyunEcsConfig{
				InternetMaxBandwidthOut: 5,
				ImageId:                 "",
				SystemDisk: EcsDiskConfig{
					Category: "cloud_essd",
					Size:     20,
				},
				DataDisk: EcsDiskConfig{
					Category: "cloud_essd",
					Size:     40,
				},
				HostName:                 "examplehost",
				RootPassword:             "",
				ProdPassword:             "",
				SpotInterruptionBehavior: "Stop",
				SecurityGroupId:          "",
			},
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Username: "",
			Password: "",
			Database: "",
		},
		Monitor: MonitorConfig{
			ActiveInstance: ActiveInstance{
				Interval: 5,
			},
			PublicIP: PublicIP{
				Interval: 5,
				Timeout:  20,
			},
			InstanceCharge: InstanceCharge{
				Interval:            600,
				RetryInterval:       300,
				Timeout:             240,
				MemChoices:          []int{16},
				CpuCoreCountChoices: []int{4},
				Filters: InstanceChargeFilters{
					MaxTradePrice:         0.6,
					InstanceTypeExclusion: "^ecs\\.(e|s6|xn4|n4|mn4|e4|t|d).*$",
				},
				CacheFile: "latest_preferred_instance_charge.json",
			},
			Backup: Backup{
				Interval:      600,
				RetryInterval: 60,
				Timeout:       120,
			},
			EmptyServer: EmptyServer{
				EmptyTimeout: 3600,
			},
			ServerStatus: ServerStatus{
				Interval: 5,
				Timeout:  10,
			},
			StartInstance: StartInstance{
				Interval: 5,
				Timeout:  120,
			},
			BssSync: BssSync{
				Interval:    3600,
				Timeout:     30,
				InitialTime: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
			},
			Whitelist: Whitelist{
				Interval:  60,
				Timeout:   15,
				CacheFile: "whitelist.json",
			},
			PlayerProfile: PlayerProfile{
				Interval:       300,
				Timeout:        60,
				EssentialsPath: "/home/mc/archive/plugins/Essentials/userdata/*.yml",
				StatsPath:      "/home/mc/archive/world/stats/*.json",
				EssentialsDir:  "playerprofiles/essentials",
				StatsDir:       "playerprofiles/stats",
			},
		},
		Deploy: DeployConfig{
			Packages:     []string{"screen", "unzip", "zip", "screenfetch", "vim", "htop"},
			SSHPublicKey: "",
			JavaVersion:  21,
			OSSRoot:      "oss://mybucket",
			BackupPath:   "/backups",
			ArchivePath:  "/archive",
		},
		Server: ServerConfig{
			Port:         25565,
			RconPort:     25575,
			RconPassword: "",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile("../config.example.toml", out, 0644)

	if err != nil {
		t.Fatal(err)
	}
}
