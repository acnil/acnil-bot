package ilog

import (
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
)

func WithTelegramUser(log *logrus.Entry, u *tele.User) *logrus.Entry {
	return log.
		WithField(FieldUsername, u.Username).
		WithField(FieldName, u.FirstName+" "+u.LastName).
		WithField(FieldID, u.ID)
}
