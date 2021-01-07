package main

import (
	"context"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/wcharczuk/go-chart"
	"google.golang.org/api/fitness/v1"
	"google.golang.org/api/option"
)

type Activity struct {
	Name         string
	Duration     int64
	Distance     float64
	Description  string
	Date         time.Time
	ActivityType int64
}

type Activities []Activity

func (e Activities) Len() int {
	return len(e)
}

func (e Activities) Less(i, j int) bool {
	return e[i].Date.Before(e[j].Date)
}

func (e Activities) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func removeDuplicates(activities Activities) Activities {
	var dedupe Activities
	seen := map[string]bool{}
	for _, activity := range activities {
		if seen[activity.Date.String()] == false {
			dedupe = append(dedupe, activity)
			seen[activity.Date.String()] = true
		}
	}
	return dedupe
}

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("unable to find config dir: %v\n", err)
	}
	path := filepath.Join(configDir, "gem/fitness/client_secret.json")
	client := getFullClient(path)
	fitnessService, err := fitness.NewService(context.TODO(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("%v\n", err.Error())
	}

	datasetService := fitness.NewUsersDatasetService(fitnessService)
	sessionService := fitness.NewUsersSessionsService(fitnessService)

	call := sessionService.List("me")
	call.StartTime("2020-01-01T00:00:00.000Z")
	call.EndTime("2020-12-31T23:59:59.000Z")
	// https://developers.google.com/fit/rest/v1/reference/activity-types
	//  1 = Biking
	// 15 = Mountain Biking
	// 16 = Road Biking
	// 17 = Spinning
	// 18 = Stationary Biking
	// 19 = Utility Biking even though I don't think I've ever done this
	//  8 = Running
	call.ActivityType(1, 15, 16, 17, 18, 19, 8)
	resp, err := call.Do()
	if err != nil {
		log.Fatalf("%v", err.Error())
	}

	var aggregates []*fitness.AggregateBy
	aggregates = append(aggregates, &fitness.AggregateBy{
		DataTypeName: "com.google.activity.segment",
	})
	aggregates = append(aggregates, &fitness.AggregateBy{
		DataTypeName: "com.google.distance.delta",
	})

	var activities Activities

	for _, session := range resp.Session {
		var c = datasetService.Aggregate("me", &fitness.AggregateRequest{
			AggregateBy: aggregates,
			BucketBySession: &fitness.BucketBySession{
				MinDurationMillis: 100,
			},
			EndTimeMillis:   session.EndTimeMillis,
			StartTimeMillis: session.StartTimeMillis,
		})
		r, err := c.Do()
		if err != nil {
			log.Fatalf("error getting dataset: %v\n", err)
		}

		for _, bucket := range r.Bucket {
			timestamp := time.Unix(bucket.StartTimeMillis/1000, 0)

			activity := Activity{
				Name:         session.Name,
				Duration:     (bucket.EndTimeMillis - bucket.StartTimeMillis) / 1000 / 60,
				Distance:     0,
				Description:  bucket.Session.Description,
				Date:         timestamp,
				ActivityType: session.ActivityType,
			}
			for _, dataset := range bucket.Dataset {
				if dataset.DataSourceId == "derived:com.google.distance.delta:com.google.android.gms:aggregated" {
					for _, points := range dataset.Point {
						for _, v := range points.Value {
							// convert meters to miles and round
							var round float64
							dist := v.FpVal / 1609.344
							pow := math.Pow(10, 2.0)
							digit := pow * dist
							_, div := math.Modf(digit)
							if div >= 0.5 {
								round = math.Ceil(digit)
							} else {
								round = math.Floor(digit)
							}
							activity.Distance = round / pow
						}
					}
				}
			}
			activities = append(activities, activity)
		}
	}

	activities = removeDuplicates(activities)
	sort.Sort(activities)

	var ys []float64
	var xs []float64
	totalDist := 0.0

	for _, activity := range activities {
		if activity.Distance != 0 {
			totalDist = totalDist + activity.Distance
			ys = append(ys, totalDist)
			xs = append(xs, float64(activity.Date.Unix()))
		}
	}

	jan := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	graph := chart.Chart{
		YAxis: chart.YAxis{
			Name: "Miles",
			Ticks: []chart.Tick{
				{Value: 0, Label: "0"},
				{Value: 100, Label: "100"},
				{Value: 200, Label: "200"},
				{Value: 300, Label: "300"},
				{Value: 400, Label: "400"},
				{Value: 500, Label: "500"},
				{Value: 600, Label: "600"},
				{Value: 700, Label: "700"},
				{Value: 800, Label: "800"},
				{Value: 900, Label: "900"},
				{Value: 1000, Label: "1000"},
			},
		},
		XAxis: chart.XAxis{
			Name: "Date",
			Ticks: []chart.Tick{
				{Value: float64(jan.Unix()), Label: "2020-01"},
				{Value: float64(jan.AddDate(0, 1, 0).Unix()), Label: "2020-02"},
				{Value: float64(jan.AddDate(0, 2, 0).Unix()), Label: "2020-03"},
				{Value: float64(jan.AddDate(0, 3, 0).Unix()), Label: "2020-04"},
				{Value: float64(jan.AddDate(0, 4, 0).Unix()), Label: "2020-05"},
				{Value: float64(jan.AddDate(0, 5, 0).Unix()), Label: "2020-06"},
				{Value: float64(jan.AddDate(0, 6, 0).Unix()), Label: "2020-07"},
				{Value: float64(jan.AddDate(0, 7, 0).Unix()), Label: "2020-08"},
				{Value: float64(jan.AddDate(0, 8, 0).Unix()), Label: "2020-09"},
				{Value: float64(jan.AddDate(0, 9, 0).Unix()), Label: "2020-10"},
				{Value: float64(jan.AddDate(0, 10, 0).Unix()), Label: "2020-11"},
				{Value: float64(jan.AddDate(0, 11, 0).Unix()), Label: "2020-12"},
				{Value: float64(jan.AddDate(0, 12, 0).Unix()), Label: "2021-01"},
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: xs,
				YValues: ys,
			},
		},
	}

	err = graph.Render(chart.SVG, os.Stdout)
	if err != nil {
		log.Fatalf("error rending graph: %v", err.Error())
	}
}
