package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"net/url"
	"os"
	"time"

	"github.com/jordan-wright/email"
)

type slackNotification struct {
	Message string `json:"text"`
}

type countQueryResult struct {
	Count int `json:"count"`
}

type searchQueryResult struct {
	Took      int       `json:"took"`
	HitHeader hitHeader `json:"hits"`
}

type hitHeader struct {
	Total     int             `json:"took"`
	Documents json.RawMessage `json:"hits"`
}

type slackNotificationSetting struct {
	URI     string `json:"uri"`
	Channel string `json:"channel"`
	Sender  string `json:"sender"`
	Emoji   string `json:"emoji"`
}

type slackAttachment struct {
	FallbackText string `json:"fallback"`
	Text         string `json:"text"`
}

type slackMessage struct {
	Message     string            `json:"text"`
	Channel     string            `json:"channel"`
	Sender      string            `json:"username"`
	Emoji       string            `json:"icon_emoji"`
	Attachments []slackAttachment `json:"attachments"`
}

type emailNotificationSetting struct {
	SMTPServer       string   `json:"smtp_server"`
	SMTPPort         int      `json:"smtp_port"`
	SMTPAuthUser     string   `json:"smtp_auth_user"`
	SMTPAuthPassword string   `json:"smtp_auth_password"`
	FromAddress      string   `json:"smtp_from_address"`
	ToAddresses      []string `json:"smtp_to_addresses"`
}

type notificationOverrides struct {
	Slack slackNotificationSetting `json:"slack"`
	Email emailNotificationSetting `json:"email"`
}

type notificationRule struct {
	Name                  string                `json:"rule_name"`
	Type                  string                `json:"rule_type"`
	NotificationMessage   string                `json:"notification_message"`
	ClusterName           string                `json:"cluster_name"`
	IndexName             string                `json:"index_name"`
	DocumentType          string                `json:"document_type"`
	Enabled               bool                  `json:"enabled"`
	Operator              string                `json:"operator"`
	Threshold             int                   `json:"threshold"`
	Interval              int                   `json:"interval"`
	NotificationInterval  int                   `json:"notification_interval"`
	NotificationOverrides notificationOverrides `json:"notification_overrides"`
	Query                 json.RawMessage       `json:"query"`
	LastProcessedTime     time.Time
	LastNotificationSent  time.Time
}

var reloadNotifications bool
var notificationRules []notificationRule

func readNotificationRules() []notificationRule {
	var readRules []notificationRule
	if _, err := os.Stat("rules"); err != nil {
		if os.IsNotExist(err) {
			// folder does not exist. Since no rules are expected to run, log and continue
			log.Print("rules folder does not exist. No rules are loaded.")
			return readRules
		}

		// if the error is something else, like possibly permissions, log the error and kill the process
		log.Fatal("Error loading rules folder ", err)

	}

	files, err := ioutil.ReadDir("rules")
	if err != nil {
		log.Fatal("Error loading rules folder ", err)
	}

	for _, file := range files {
		log.Print("Loading notification rule from ", file.Name())

		fileJSON, err := ioutil.ReadFile("rules/" + file.Name())
		if err != nil {
			log.Fatal("Error reading rule file: ", file.Name(), err)
		}

		var rule notificationRule

		err = json.Unmarshal(fileJSON, &rule)
		if err != nil {
			log.Fatalf("Error parsing rule file: %v %v", file.Name(), err)
		}

		hasClusterConfig := false
		for _, cluster := range configuration.ElasticClientsFrom {
			if cluster.ClusterName == rule.ClusterName {
				hasClusterConfig = true
			}
		}

		if !hasClusterConfig && rule.Enabled {
			log.Fatal("Rules ensabled but no cluster configuration could be found for: ", rule.ClusterName)
		}

		readRules = append(readRules, rule)
	}

	return readRules
}

func loadNotificationRules() {

	notificationRules = readNotificationRules()

	reloadNotifications = false
}

func reloadNotificationRules() {
	log.Print("Reloading rules due to a change in a rule file")
	reloadedRules := readNotificationRules()

	for i := 0; i < len(reloadedRules); i++ {
		for j := 0; j < len(notificationRules); j++ {
			if reloadedRules[i].Name == notificationRules[j].Name {
				reloadedRules[i].LastNotificationSent = notificationRules[j].LastNotificationSent
				reloadedRules[i].LastProcessedTime = notificationRules[j].LastProcessedTime
			}
		}
	}

	notificationRules = reloadedRules

	reloadNotifications = false
}

func runNotificationRules() {
	if reloadNotifications {
		reloadNotificationRules()
	}

	for i := 0; i < len(notificationRules); i++ {
		// determine if the rule should be run.
		if notificationRules[i].Enabled {
			if notificationRules[i].LastProcessedTime.Before(time.Now().
				Add(time.Minute * time.Duration(notificationRules[i].Interval) * -1)) {

				log.Print("Running rule: ", notificationRules[i].Name)
				notificationRules[i].LastProcessedTime = time.Now().Add(time.Second * -1)

				processNotificationRule(&notificationRules[i])
			}
		}
	}
}

func processNotificationRule(rule *notificationRule) {
	var hosts []string

	for _, cluster := range configuration.ElasticClientsFrom {
		if cluster.ClusterName == rule.ClusterName {
			hosts = cluster.Hosts
		}
	}

	var urlBuffer bytes.Buffer
	if rule.IndexName == "" {
		urlBuffer.WriteString("*")
	} else {
		urlBuffer.WriteString(rule.IndexName)
		if rule.DocumentType != "" {
			urlBuffer.WriteString("/")
			urlBuffer.WriteString(rule.DocumentType)
		}
	}
	if rule.Type == "count" {
		urlBuffer.WriteString("/_count")
	}

	if rule.Type == "search" {
		urlBuffer.WriteString("/_search")
	}

	body, err := failoverHTTPRequest(hosts, "POST", urlBuffer.String(), bytes.NewBuffer([]byte(string(rule.Query))))
	if err != nil {
		log.Printf("Error running query for rule %v : %v", rule.Name, err)
	} else {
		var hitCount int
		var queryResults []byte
		var parseErr error

		if rule.Type == "count" {
			hitCount, parseErr = parseCountQuery(body)
		}

		if rule.Type == "search" {
			hitCount, queryResults, parseErr = parseSearchQuery(body)
		}

		if parseErr != nil {
			log.Printf("Error parsing result of query for rule %v : %v", rule.Name, parseErr)
			return
		}

		notify := false

		switch rule.Operator {
		case "eq", "==":
			notify = hitCount == rule.Threshold
			break
		case "neq", "!=", "<>":
			notify = hitCount != rule.Threshold
			break
		case "gt", ">":
			notify = hitCount > rule.Threshold
			break
		case "gte", ">=":
			notify = hitCount >= rule.Threshold
			break
		case "lt", "<":
			notify = hitCount < rule.Threshold
			break
		case "lte", "<=":
			notify = hitCount <= rule.Threshold
			break
		}

		if rule.LastNotificationSent.After(time.Now().Add(time.Hour * time.Duration(rule.NotificationInterval) * -1)) {
			notify = false
		}

		if notify {
			rule.LastNotificationSent = time.Now()

			sendNotification(fmt.Sprintf("%v Result count was %v", rule.NotificationMessage, hitCount),
				queryResults, rule.NotificationOverrides)
		}
	}
}

func parseCountQuery(result []byte) (int, error) {
	var countResult countQueryResult
	err := json.Unmarshal(result, &countResult)
	if err != nil {
		return -1, err
	}
	return countResult.Count, nil
}

func parseSearchQuery(result []byte) (int, []byte, error) {
	var queryResult searchQueryResult
	err := json.Unmarshal(result, &queryResult)
	if err != nil {
		return -1, nil, err
	}

	return queryResult.Took, []byte(queryResult.HitHeader.Documents), nil
}

func buildSlackSettings(overrideSettings slackNotificationSetting) slackNotificationSetting {
	var slackSettings slackNotificationSetting
	slackSettings.URI = configuration.DefaultSlackWebookURI
	slackSettings.Channel = configuration.DefaultSlackWebookChannel
	slackSettings.Sender = configuration.DefaultSlackWebookSender
	slackSettings.Emoji = configuration.DefaultSlackWebookEmoji

	if overrideSettings.URI != "" {
		slackSettings.URI = overrideSettings.URI
	}

	if overrideSettings.Channel != "" {
		slackSettings.Channel = overrideSettings.Channel
	}

	if overrideSettings.Sender != "" {
		slackSettings.Sender = overrideSettings.Sender
	}

	if overrideSettings.Emoji != "" {
		slackSettings.Emoji = overrideSettings.Emoji
	}
	return slackSettings
}

func buildEmailSettings(overrideSettings emailNotificationSetting) emailNotificationSetting {
	var emailSettings emailNotificationSetting

	emailSettings.SMTPServer = configuration.DefaultSMTPServer
	emailSettings.SMTPPort = configuration.DefaultSMTPPort
	emailSettings.SMTPAuthUser = configuration.DefaultSMTPAuthUser
	emailSettings.SMTPAuthPassword = configuration.DefaultSMTPAuthPassword
	emailSettings.FromAddress = configuration.DefaultSMTPFromAddress
	emailSettings.ToAddresses = configuration.DefaultSMTPToAddresses

	if overrideSettings.SMTPServer != "" {
		emailSettings.SMTPServer = overrideSettings.SMTPServer
	}

	if overrideSettings.SMTPPort > 0 {
		emailSettings.SMTPPort = overrideSettings.SMTPPort
	}

	if overrideSettings.SMTPAuthUser != "" {
		emailSettings.SMTPAuthUser = overrideSettings.SMTPAuthUser
	}

	if overrideSettings.SMTPAuthPassword != "" {
		emailSettings.SMTPAuthPassword = overrideSettings.SMTPAuthPassword
	}

	if overrideSettings.FromAddress != "" {
		emailSettings.FromAddress = overrideSettings.FromAddress
	}

	if len(overrideSettings.ToAddresses) > 0 {
		emailSettings.ToAddresses = overrideSettings.ToAddresses
	}

	return emailSettings
}

func sendNotification(message string, attachment []byte, overrides notificationOverrides) {

	for _, notifyMethod := range configuration.Notifications {
		if notifyMethod == "slack" {
			slackSettings := buildSlackSettings(overrides.Slack)
			sendSlackNotification(slackSettings, message)
		}
		if notifyMethod == "email" {
			emailSettings := buildEmailSettings(overrides.Email)
			sendEmailNotification(emailSettings, message, attachment)
		}
	}

	log.Print("Notification Posted: ", message)
}

func sendSlackNotification(settings slackNotificationSetting, message string) {
	var slackNotificationMessage slackMessage
	slackNotificationMessage.Message = message
	slackNotificationMessage.Sender = settings.Sender
	slackNotificationMessage.Channel = settings.Channel
	slackNotificationMessage.Emoji = settings.Emoji

	if settings.URI == "" {
		log.Print("Slack URI not valid")
		return
	}

	uri, err := url.Parse(settings.URI)
	if err != nil {
		log.Print("Slack URI not valid")
		return
	}

	messageBody, _ := json.Marshal(slackNotificationMessage)

	log.Print(string(messageBody))

	hostURI := fmt.Sprintf("%v://%v", uri.Scheme, uri.Host)

	hosts := []string{hostURI}

	failoverHTTPRequest(hosts, "POST", uri.Path, bytes.NewBuffer(messageBody))

}

func sendEmailNotification(settings emailNotificationSetting, message string, attachment []byte) {
	if settings.SMTPServer == "" || settings.FromAddress == "" || len(settings.ToAddresses) == 0 {
		log.Print("Email settings are not valid")
		return
	}

	msg := email.NewEmail()
	msg.From = settings.FromAddress
	msg.To = settings.ToAddresses
	msg.Subject = fmt.Sprint("Gwylio Notification: ", message)
	msg.Text = []byte(fmt.Sprint(message, "\r\n"))
	if len(attachment) > 0 {
		msg.Attach(bytes.NewBuffer(attachment), "results.json", "text/json")
	}

	var auth smtp.Auth
	if settings.SMTPAuthUser != "" && settings.SMTPAuthPassword != "" {
		auth = smtp.PlainAuth("", settings.SMTPAuthUser, settings.SMTPAuthPassword, settings.SMTPServer)
	}

	err := msg.Send(fmt.Sprintf("%v:%v", settings.SMTPServer, settings.SMTPPort), auth)
	if err != nil {
		log.Print("Error sending email notification: ", err)
	}

}
