package simba_test

import (
	"testing"

	"github.com/saisona/simba"
)

func TestInitConfigChannelIdIsMissing(t *testing.T) {
	t.Helper()
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "CHANNEL_ID is not set" {
		t.Fatalf("got: %s,wanted : CHANNEL_ID is not set", err.Error())
	}
}

func TestInitConfigSlackApiTokenMissing(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "")
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "SLACK_API_TOKEN is not set" {
		t.Fatalf("got: %s,wanted : SLACK_API_TOKEN is not set", err.Error())
	}
}

func TestInitConfigAppPortMissing(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "xob-xxxxxxx")
	t.Setenv("APP_PORT", "")
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "APP_PORT is not set" {
		t.Fatalf("got: %s,wanted : APP_PORT is not set", err.Error())
	}
}

func TestInitConfigDbFailedUser(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "xob-xxxxxxx")
	t.Setenv("APP_PORT", "1337")
	t.Setenv("DB_USER", "")
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "initDbConfig failed : DB_USER is not set" {
		t.Fatalf("got: %s,wanted : initDbConfig failed : DB_USER is not set", err.Error())
	}
}

func TestInitDbConfigDbPasswordMissing(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "xob-xxxxxxx")
	t.Setenv("APP_PORT", "1337")
	t.Setenv("DB_USER", "fake_user")
	t.Setenv("DB_PASSWORD", "")
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "initDbConfig failed : DB_PASSWORD is not set" {
		t.Fatalf("got: %s,wanted : initDbConfig failed : DB_PASSWORD is not set", err.Error())
	}
}

func TestInitDbConfigDbHostMissing(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "xob-xxxxxxx")
	t.Setenv("APP_PORT", "1337")
	t.Setenv("DB_USER", "fake_user")
	t.Setenv("DB_PASSWORD", "fake_password")
	t.Setenv("DB_HOST", "")
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "initDbConfig failed : DB_HOST is not set" {
		t.Fatalf("got: %s,wanted : initDbConfig failed : DB_HOST is not set", err.Error())
	}
}

func TestInitDbConfigDbNameMissing(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "xob-xxxxxxx")
	t.Setenv("APP_PORT", "1337")
	t.Setenv("DB_USER", "fake_user")
	t.Setenv("DB_PASSWORD", "fake_password")
	t.Setenv("DB_HOST", "fake_host")
	t.Setenv("DB_NAME", "")
	_, err := simba.InitConfig(true)
	if err == nil || err.Error() != "initDbConfig failed : DB_NAME is not set" {
		t.Fatalf("got: %s,wanted : initDbConfig failed : DB_NAME is not set", err.Error())
	}
}

func TestInitDbSuccess(t *testing.T) {
	t.Setenv("CHANNEL_ID", "toto")
	t.Setenv("SLACK_API_TOKEN", "xob-xxxxxxx")
	t.Setenv("APP_PORT", "1337")
	t.Setenv("DB_USER", "fake_user")
	t.Setenv("DB_PASSWORD", "fake_password")
	t.Setenv("DB_HOST", "fake_host")
	t.Setenv("DB_NAME", "fake_name")
	config, err := simba.InitConfig(true)
	if err != nil {
		t.Fatalf("got: %s,wanted : nil", err.Error())
	}

	if config.CHANNEL_ID != "toto" {
		t.FailNow()
	} else if config.APP_PORT != "1337" {
		t.FailNow()
	} else if config.SLACK_API_TOKEN != "xob-xxxxxxx" {
		t.FailNow()
	} else if config.DB.Host != "fake_host" {
		t.FailNow()
	} else if config.DB.Name != "fake_name" {
		t.FailNow()
	} else if config.DB.Username != "fake_user" {
		t.FailNow()
	} else if config.DB.Password != "fake_password" {
		t.FailNow()
	} else if config.SLACK_MESSAGE_CHANNEL == nil {
		t.FailNow()
	}
}
