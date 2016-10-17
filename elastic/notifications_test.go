package elastic

import "testing"

func TestSlackSettingsDefaults(t *testing.T) {

	slackWebHookURI := "http://test.url"
	slackChannel := "#TestChannel"
	slackSender := "Gwylio"
	slackEmoji := ":test:"

	configuration.DefaultSlackWebookURI = slackWebHookURI
	configuration.DefaultSlackWebookChannel = slackChannel
	configuration.DefaultSlackWebookSender = slackSender
	configuration.DefaultSlackWebookEmoji = slackEmoji

	var slackOverrides slackNotificationSetting

	overriddenSettings := buildSlackSettings(slackOverrides)

	if overriddenSettings.URI != slackWebHookURI {
		t.Fail()
		t.Logf("Slack URI shouldn't be overridden if it was blank")
	}

	if overriddenSettings.Channel != slackChannel {
		t.Fail()
		t.Logf("Slack Channel shouldn't be overridden if it was blank")
	}

	if overriddenSettings.Sender != slackSender {
		t.Fail()
		t.Logf("Slack Sender shouldn't be overridden if it was blank")
	}

	if overriddenSettings.Emoji != slackEmoji {
		t.Fail()
		t.Logf("Slack Emoji shouldn't be overridden if it was blank")
	}

}

func TestSlackSettingsOverrides(t *testing.T) {

	slackWebHookURI := "http://test.url"
	slackChannel := "#TestChannel"
	slackSender := "Gwylio"
	slackEmoji := ":test:"

	overriddenSlackWebhookURI := "http://test.override"
	overriddenSlackChannel := "#OverrideChannel"
	overriddenSlackSender := "Gwylio Override"
	overriddenSlackEmoji := ":overridden:"

	configuration.DefaultSlackWebookURI = slackWebHookURI
	configuration.DefaultSlackWebookChannel = slackChannel
	configuration.DefaultSlackWebookSender = slackSender
	configuration.DefaultSlackWebookEmoji = slackEmoji

	var slackOverrides slackNotificationSetting
	slackOverrides.URI = overriddenSlackWebhookURI
	slackOverrides.Channel = overriddenSlackChannel
	slackOverrides.Sender = overriddenSlackSender
	slackOverrides.Emoji = overriddenSlackEmoji

	overriddenSettings := buildSlackSettings(slackOverrides)

	if overriddenSettings.URI != overriddenSlackWebhookURI {
		t.Fail()
		t.Logf("Slack URI should be overridden")
	}

	if overriddenSettings.Channel != overriddenSlackChannel {
		t.Fail()
		t.Logf("Slack Channel should be overridden")
	}

	if overriddenSettings.Sender != overriddenSlackSender {
		t.Fail()
		t.Logf("Slack Sender should be overridden")
	}

	if overriddenSettings.Emoji != overriddenSlackEmoji {
		t.Fail()
		t.Logf("Slack Emoji should be overridden")
	}

}

func TestEmailSettingsDefaults(t *testing.T) {
	smtpServer := "10.0.0.1"
	smtpPort := 25
	smtpAuthUser := "AuthUser"
	smtpAuthPassword := "NotASecurePassword"
	fromAddress := "test@test.com"
	toAddresses := []string{"user1@test.com", "user2@test.com"}

	configuration.DefaultSMTPServer = smtpServer
	configuration.DefaultSMTPPort = smtpPort
	configuration.DefaultSMTPAuthUser = smtpAuthUser
	configuration.DefaultSMTPAuthPassword = smtpAuthPassword
	configuration.DefaultSMTPFromAddress = fromAddress
	configuration.DefaultSMTPToAddresses = toAddresses

	var emailOverrides emailNotificationSetting

	overriddenSettings := buildEmailSettings(emailOverrides)

	if overriddenSettings.SMTPServer != smtpServer {
		t.Fail()
		t.Logf("SMTPServer shouldn't be overridden if it was blank")
	}

	if overriddenSettings.SMTPPort != smtpPort {
		t.Fail()
		t.Logf("SMTPPort shouldn't be overridden if it was blank")
	}

	if overriddenSettings.SMTPAuthUser != smtpAuthUser {
		t.Fail()
		t.Logf("SMTPAuthUser shouldn't be overridden if it was blank")
	}

	if overriddenSettings.SMTPAuthPassword != smtpAuthPassword {
		t.Fail()
		t.Logf("SMTPAuthPassword shouldn't be overridden if it was blank")
	}

	if overriddenSettings.FromAddress != fromAddress {
		t.Fail()
		t.Logf("FromAddress shouldn't be overridden if it was blank")
	}

	if len(overriddenSettings.ToAddresses) != len(toAddresses) {
		t.Fail()
		t.Logf("ToAddresses shouldn't be overridden if it was blank")
	} else {
		for i := 0; i < len(toAddresses); i++ {
			if overriddenSettings.ToAddresses[i] != toAddresses[i] {
				t.Fail()
				t.Log("ToAddresses shouldn't be overridden if it was blank: ", toAddresses[i])
			}
		}
	}
}

func TestEmailSettingsOverrides(t *testing.T) {
	smtpServer := "10.0.0.1"
	smtpPort := 25
	smtpAuthUser := "AuthUser"
	smtpAuthPassword := "NotASecurePassword"
	fromAddress := "test@test.com"
	toAddresses := []string{"user1@test.com", "user2@test.com"}

	overriddenSMTPServer := "10.0.0.2"
	overriddenSMTPPort := 26
	overriddenSMTPAuthUser := "OverriddenAuthUser"
	overriddenSMTPAuthPassword := "OverriddenNotASecurePassword"
	overriddenFromAddress := "overridefrom@test.com"
	overriddenToAddresses := []string{"overrideuser1@test.com", "overrideuser2@test.com", "overrideuser3@test.com"}

	configuration.DefaultSMTPServer = smtpServer
	configuration.DefaultSMTPPort = smtpPort
	configuration.DefaultSMTPAuthUser = smtpAuthUser
	configuration.DefaultSMTPAuthPassword = smtpAuthPassword
	configuration.DefaultSMTPFromAddress = fromAddress
	configuration.DefaultSMTPToAddresses = toAddresses

	var emailOverrides emailNotificationSetting
	emailOverrides.SMTPServer = overriddenSMTPServer
	emailOverrides.SMTPPort = overriddenSMTPPort
	emailOverrides.SMTPAuthUser = overriddenSMTPAuthUser
	emailOverrides.SMTPAuthPassword = overriddenSMTPAuthPassword
	emailOverrides.FromAddress = overriddenFromAddress
	emailOverrides.ToAddresses = overriddenToAddresses

	overriddenSettings := buildEmailSettings(emailOverrides)

	if overriddenSettings.SMTPServer != overriddenSMTPServer {
		t.Fail()
		t.Logf("SMTPServer should be overridden")
	}

	if overriddenSettings.SMTPPort != overriddenSMTPPort {
		t.Fail()
		t.Logf("SMTPPort should be overridden")
	}

	if overriddenSettings.SMTPAuthUser != overriddenSMTPAuthUser {
		t.Fail()
		t.Logf("SMTPAuthUser should be overridden")
	}

	if overriddenSettings.SMTPAuthPassword != overriddenSMTPAuthPassword {
		t.Fail()
		t.Logf("SMTPAuthPassword should be overridden")
	}

	if overriddenSettings.FromAddress != overriddenFromAddress {
		t.Fail()
		t.Logf("FromAddress should be overridden")
	}

	if len(overriddenSettings.ToAddresses) != len(overriddenToAddresses) {
		t.Fail()
		t.Logf("ToAddresses should be overridden")
	} else {
		for i := 0; i < len(toAddresses); i++ {
			if overriddenSettings.ToAddresses[i] != overriddenToAddresses[i] {
				t.Fail()
				t.Log("ToAddresses should be overridden: ", overriddenToAddresses[i])
			}
		}
	}
}
