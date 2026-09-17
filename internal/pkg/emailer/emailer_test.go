package emailer

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestNewEmailer(t *testing.T) {
    e := NewEmailer("smtp.example.com", 587, "user", "pass", "from@example.com", "MeteorX")
    assert.Equal(t, "smtp.example.com", e.Host)
    assert.Equal(t, 587, e.Port)
    assert.Equal(t, "user", e.Username)
    assert.Equal(t, "pass", e.Password)
    assert.Equal(t, "from@example.com", e.From)
    assert.Equal(t, "MeteorX", e.FromName)
}

func TestSend_HostNotConfigured(t *testing.T) {
    e := NewEmailer("", 587, "user", "pass", "from@example.com", "MeteorX")
    err := e.Send("to@example.com", "Test Subject", "Test Body")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "SMTP host not configured")
}

func TestSendResetPasswordEmail_Formatting(t *testing.T) {
    e := NewEmailer("", 587, "user", "pass", "from@example.com", "MeteorX")
    err := e.SendResetPasswordEmail("to@example.com", "https://example.com/reset?token=abc", "testuser")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "SMTP host not configured")
}

func TestSendVerificationEmail_Formatting(t *testing.T) {
    e := NewEmailer("", 587, "user", "pass", "from@example.com", "MeteorX")
    err := e.SendVerificationEmail("to@example.com", "123456", "testuser")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "SMTP host not configured")
}

func TestSend_WithFromName(t *testing.T) {
    e := NewEmailer("", 587, "user", "pass", "from@example.com", "MeteorX")
    assert.NotNil(t, e)
    assert.Equal(t, "from@example.com", e.From)
    assert.Equal(t, "MeteorX", e.FromName)
}
