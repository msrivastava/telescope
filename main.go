package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strconv"
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
	c := session.DB("").C("meter")
	defer session.Close()
	err = c.EnsureIndex(mgo.Index{
		Key:        []string{"t"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	})
	if err != nil {
		panic(err)
	}
	// poll data
	go func() {
		meter := new(Eaton)
		for _ = range time.Tick(5 * time.Second) {
			v, err := meter.Read()
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = c.Insert(v)
			if err != nil {
				fmt.Println(err)
				continue
			}
			v.Print()
		}
	}()

	// server
	m := martini.Classic()
	m.Get("/", func() string {
		return fmt.Sprintf("/%d/%d", time.Now().Add(-time.Minute).Unix(), time.Now().Unix())
	})
	m.Get("/:t1/:t2", func(params martini.Params) (s string) {
		t1, err := strconv.ParseInt(params["t1"], 10, 64)
		if err != nil {
			return
		}
		t2, err := strconv.ParseInt(params["t2"], 10, 64)
		if err != nil {
			return
		}
		var results []EatonValue
		err = c.Find(bson.M{"t": bson.M{"$lte": time.Unix(t2, 0), "$gt": time.Unix(t1, 0)}}).Sort("t").All(&results)
		if err != nil {
			return
		}
		b, err := json.Marshal(results)
		if err != nil {
			return
		}
		s = string(b)
		return
	})
	m.Run()
}
