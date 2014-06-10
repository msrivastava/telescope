package main

import (
	"labix.org/v2/mgo"
)

const (
	DB_URL = "mongodb://demo:demo@oceanic.mongohq.com:10074/telescope"
)

func session() *mgo.Session {
	session, err := mgo.Dial(DB_URL)
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)
	return session
}

func meterCollection() *mgo.Collection {
	c := MainSession.DB("").C("meter")
	c.EnsureIndex(mgo.Index{
		Key:        []string{"m", "t"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	})
	return c
}
