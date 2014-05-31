package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/render"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
	"time"
)

const (
	db = "mongodb://demo:demo@oceanic.mongohq.com:10074/telescope"
)

var (
	Meters []Eaton
	ACL    []AccessControl
)

func readMeterConfig() {
	b, err := ioutil.ReadFile("meters.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &Meters)
	if err != nil {
		panic(err)
	}
}

func readACLConfig() {
	b, err := ioutil.ReadFile("access.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &ACL)
	if err != nil {
		panic(err)
	}
}

func getCollection() *mgo.Collection {
	session, err := mgo.Dial(db)
	if err != nil {
		panic(err)
	}
	c := session.DB("").C("meter")
	c.EnsureIndex(mgo.Index{
		Key:        []string{"t"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	})
	return c
}

type MapReduceValue struct {
	Id    string             `bson:"_id" json:"_id"`
	Value map[string]float64 `bson:"value" json:"value"`
}

func main() {
	meterCollection := getCollection()
	readMeterConfig()
	readACLConfig()
	// poll data
	for _, meter := range Meters {
		go func(m Eaton) {
			fmt.Printf("start meter %v\n", m.Addr)
			for _ = range time.Tick(time.Minute) {
				v, err := m.Read()
				if err != nil {
					fmt.Println(err)
					continue
				}
				err = meterCollection.Insert(v)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}(meter)
	}
	// get power stat
	go func() {
		var results []MapReduceValue
		for _ = range time.Tick(time.Minute * 10) {
			job := &mgo.MapReduce{
				Map: `function() {
					if (new Date(new Date() - 1000 * 60 * 60) > this.t) {
						return
					}
					var n = Math.abs(this.v[9])
					emit(this.m, // Or put a GROUP BY key here
					{
					 	sum: n, // the field you want stats for
						min: n,
						max: n,
						count:1,
						diff: 0,
					});
				}`,
				Reduce: `function(key, values) {
					var a = values[0]; // will reduce into here
					for (var i=1; i < values.length; i++){
						var b = values[i]; // will merge 'b' into 'a'
						// temp helpers
						var delta = a.sum/a.count - b.sum/b.count; // a.mean - b.mean
						var weight = (a.count * b.count)/(a.count + b.count);
						
						// do the reducing
						a.diff += b.diff + delta*delta*weight;
						a.sum += b.sum;
						a.count += b.count;
						a.min = Math.min(a.min, b.min);
						a.max = Math.max(a.max, b.max);
					}
					return a;
				}`,
				Finalize: `function(key, value) { 
					value.avg = value.sum / value.count;
					value.variance = value.diff / value.count;
					value.stddev = Math.sqrt(value.variance);
					return value;
				}`,
			}
			meterCollection.Find(nil).MapReduce(job, &results)
			for _, result := range results {
				for i := range Meters {
					if result.Id == Meters[i].Name {
						Meters[i].Avg = result.Value["avg"]
						Meters[i].Max = result.Value["max"]
						Meters[i].Min = result.Value["min"]
						Meters[i].Stddev = result.Value["stddev"]
						break
					}
				}
			}
		}
	}()

	// server
	m := martini.Classic()
	m.Use(gzip.All())
	m.Use(render.Renderer())
	m.Use(martini.Recovery())

	m.Get("/", func() string {
		return "Built By Tai-Lin Chu. Released under GPL2. :)"
	})

	m.Get("/:meter/:start/:stop/:step", func(params martini.Params, r render.Render, req *http.Request) {
		if !canAccess(req.RemoteAddr, params["meter"]) {
			fmt.Printf("access deny(%v): %v\n", params["meter"], req.RemoteAddr)
			r.JSON(http.StatusBadRequest, nil)
			return
		}
		start, _ := strconv.ParseInt(params["start"], 10, 64)
		stop, _ := strconv.ParseInt(params["stop"], 10, 64)
		step, _ := strconv.ParseInt(params["step"], 10, 64)
		var results []EatonValue
		err := meterCollection.Find(bson.M{
			"t": bson.M{
				"$lt":  time.Unix(stop, 0),
				"$gte": time.Unix(start, 0),
			},
			"m": params["meter"],
		}).Select(bson.M{
			"t": 1,
			"v": bson.M{"$slice": []int{9, 1}},
		}).Sort("t").All(&results)
		if err != nil {
			r.JSON(http.StatusBadRequest, nil)
			return
		}
		r.JSON(http.StatusOK, resample(start, stop, step, results))
	})

	m.Get("/list", func(r render.Render) {
		r.JSON(http.StatusOK, Meters)
	})

	m.Run()
}

func canAccess(addr, meter string) bool {
	for _, a := range ACL {
		if a.Match(addr) {
			return a.HasMeter(meter)
		}
	}
	return false
}

func resample(start, stop, step int64, data []EatonValue) (values []float64) {
	var j int
	var v float64
	for i := start; i < stop; i += step {
		for j < len(data) && data[j].Time().Unix() < i {
			j++
		}
		if j >= len(data) {
			values = append(values, v)
			continue
		}
		t := data[j].Time().Unix()
		if i <= t && t < i+step {
			read := data[j].Power()
			if read != 0 {
				v = read
			}
		}
		values = append(values, v)
	}
	return
}
