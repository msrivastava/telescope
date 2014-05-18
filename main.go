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
	db   = "mongodb://demo:demo@oceanic.mongohq.com:10074/telescope"
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
			meter := Eaton{Addr: addr}
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
	m.Get("/:meter/:t1/:t2", func(params martini.Params) (s string) {
		t1, err := strconv.ParseInt(params["t1"], 10, 64)
		if err != nil {
			return
		}
		t2, err := strconv.ParseInt(params["t2"], 10, 64)
		if err != nil {
			return
		}
		c := session.DB("").C(params["meter"])
		var results []EatonValue
		err = c.Find(bson.M{"t": bson.M{"$lte": time.Unix(t2, 0), "$gt": time.Unix(t1, 0)}}).Sort("t").All(&results)
		if err != nil {
			fmt.Println(err)
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
