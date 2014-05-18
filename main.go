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

var (
	Addr = []string{
		"128.97.93.90:4661",
		"128.97.93.90:4662",
		"128.97.93.90:4663",
	}
)

func main() {
	session, err := mgo.Dial(db)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// poll data
	for id, addr := range Addr {
		go func(idS, addr string) {
			c := session.DB("").C("meter" + idS)
			c.EnsureIndex(mgo.Index{
				Key:        []string{"t"},
				Unique:     true,
				DropDups:   true,
				Background: true,
				Sparse:     true,
			})
			meter := Eaton{Addr: addr}
			fmt.Printf("start meter %v\n", addr)
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
			}
		}(strconv.Itoa(id), addr)
	}

	// server
	m := martini.Classic()
	m.Get("/", func() string {
		return fmt.Sprintf("/%d/%d", time.Now().Add(-time.Minute).Unix(), time.Now().Unix())
	})
	m.Get("/:meter/:t1/:t2", func(params martini.Params) string {
		t1, _ := strconv.ParseInt(params["t1"], 10, 64)
		t2, _ := strconv.ParseInt(params["t2"], 10, 64)
		c := session.DB("").C(params["meter"])
		var results []EatonValue
		c.Find(bson.M{"t": bson.M{"$lte": time.Unix(t2, 0), "$gt": time.Unix(t1, 0)}}).Sort("t").All(&results)
		b, err := json.Marshal(results)
		if err != nil {
			return "null"
		}
		return string(b)
	})
	m.Run()
}
