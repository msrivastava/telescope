package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
	//"strings"
	"time"
)

const (
	db = "mongodb://demo:demo@oceanic.mongohq.com:10074/telescope"
)

var (
	Meters = []Eaton{
		{
			Name: "meter0",
			Addr: "128.97.11.100:4660",
		},
		{
			Name: "meter1",
			Addr: "128.97.11.101:4660",
		},
		{
			Name: "meter2",
			Addr: "128.97.11.102:4660",
		},
	}
)

func MustEncode(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func main() {
	session, err := mgo.Dial(db)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c := session.DB("").C("meter")
	c.EnsureIndex(mgo.Index{
		Key:        []string{"t"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	})

	// poll data
	for _, meter := range Meters {
		go func(m Eaton) {
			fmt.Printf("start meter %v\n", m.Addr)
			for _ = range time.Tick(5 * time.Second) {
				v, err := m.Read()
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
		}(meter)
	}

	// server
	m := martini.Classic()
	m.Use(martini.Recovery())

	m.Get("/", func() string {
		return fmt.Sprintf("/%d/%d", time.Now().Add(-time.Minute).Unix(), time.Now().Unix())
	})

	m.Get("/:meter/:start/:stop", func(params martini.Params) (int, string) {
		start, _ := strconv.ParseInt(params["start"], 10, 64)
		stop, _ := strconv.ParseInt(params["stop"], 10, 64)
		var results []EatonValue
		err := c.Find(bson.M{
			"t": bson.M{
				"$lt":  time.Unix(stop, 0),
				"$gte": time.Unix(start, 0),
			},
			"m": params["meter"],
		}).Sort("t").All(&results)
		if err != nil {
			return http.StatusBadRequest, ""
		}
		return http.StatusOK, MustEncode(results)
	})

	m.Get("/list", func() (int, string) {
		return http.StatusOK, MustEncode(Meters)
	})

	m.Get("/avg", func() (int, string) {
		type MeterResp struct {
			M string  `json:"_id"`
			V float64 `json:"v"`
		}
		var value []MeterResp
		c.Pipe([]bson.M{
			{"$group": bson.M{
				"_id": "$m",
				"v":   bson.M{"$avg": "$v[9]"},
			}}}).All(&value)
		return http.StatusOK, MustEncode(value)

	})

	m.Run()
}
