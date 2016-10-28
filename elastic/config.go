package elastic

import (
	"io/ioutil"
	"log"

	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

var configuration options

type options struct {
	CollectInterval            int                 `yaml:"collect_interval"`
	ElasticClientsFrom         []elasticHostConfig `yaml:"elastic_clients_from"`
	ElasticClientsTo           []string            `yaml:"elastic_clients_to"`
	NotifyOnNodeCountChange    bool                `yaml:"notify_on_node_count_change"`
	NotifyOnClusterYellow      bool                `yaml:"notify_on_cluster_yellow"`
	NotifyOnClusterRed         bool                `yaml:"notify_on_cluster_red"`
	NotifyOnClusterUnavailable bool                `yaml:"notify_on_cluster_unavailable"`
	Notifications              []string            `yaml:"notifications"`
	IndexPrefix                string              `yaml:"index_prefix"`
	DefaultSlackWebookURI      string              `yaml:"slack_webhook_uri"`
	DefaultSlackWebookChannel  string              `yaml:"slack_webhook_channel"`
	DefaultSlackWebookSender   string              `yaml:"slack_webhook_sender"`
	DefaultSlackWebookEmoji    string              `yaml:"slack_webhook_emoji"`
	DefaultSMTPServer          string              `yaml:"smtp_server"`
	DefaultSMTPPort            int                 `yaml:"smtp_port"`
	DefaultSMTPAuthUser        string              `yaml:"smtp_auth_user"`
	DefaultSMTPAuthPassword    string              `yaml:"smtp_auth_password"`
	DefaultSMTPFromAddress     string              `yaml:"smtp_from_address"`
	DefaultSMTPToAddresses     []string            `yaml:"smtp_to_addresses"`
	DefaultHipChatAuthToken    string              `yaml:"hipchat_auth_token"`
	DefaultHipBaseURL          string              `yaml:"hipchat_base_url"`
	DefaultHipChatRoom         string              `yaml:"hipchat_room"`
}

type elasticHostConfig struct {
	Hosts             []string `yaml:"hosts"`
	ClusterName       string   `yaml:"cluster_name"`
	ExpectedNodeCount int      `yaml:"expected_node_count"`
}

func loadConfiguration() {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "gwylio.log",
		MaxSize:    50, // megabytes
		MaxBackups: 1,
		MaxAge:     28, //days
	})

	log.Print("Starting up...")
	log.Print("Loading Config...")

	yamlFile, err := ioutil.ReadFile("gwylio.yml")
	if err != nil {
		log.Fatal("yamlFile.Get err ", err)
	}

	err = yaml.Unmarshal(yamlFile, &configuration)
	if err != nil {
		log.Fatal("error: ", err)
	}

	for _, hostCollection := range configuration.ElasticClientsFrom {
		for _, host := range hostCollection.Hosts {
			log.Print("Configured for Host: ", host)
		}
	}
}
