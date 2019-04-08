package main

import (
	"gopkg.in/mgo.v2"
)

var session *mgo.Session
var sessionErr error

// GetDatabaseSessionCopy create a copy of opened session
func GetDatabaseSessionCopy() (*mgo.Database, *mgo.Session) {
	sessionCopy := session.Copy()
	return sessionCopy.DB("arena"), sessionCopy
}
