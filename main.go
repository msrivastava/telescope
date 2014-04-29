package main

import (
	"fmt"
	"labix.org/v2/mgo"
	"time"
)

const (
	db = "mongodb://demo:demo@oceanic.mongohq.com:10074/telescope"
)

func main() {
	session, err := mgo.Dial(db)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c := session.DB("").C("meter")
	meter := new(Veris)
	for _ := range time.Tick(5 * time.Second) {
		v, err := meter.Read()
		if err != nil {
			fmt.Println(err)
			return
		}
		err = c.Insert(v)
		if err != nil {
			fmt.Println(err)
			return
		}
		v.Print()
	}

}
