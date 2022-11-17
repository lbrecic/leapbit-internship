package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
	db "gitlab.com/leapbit-practice/tweety-lib-db/db"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Application struct {
	DB      *sql.DB
	Server  *http.Server
	Metrics Metrics
}

type Metrics struct {
	TotalRequests    *prometheus.CounterVec
	RequestsDuration *prometheus.HistogramVec
}

func (application *Application) lastTweetHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Last Tweet Handler starting..."
	com.TweetyLog(com.INFO, msg)

	lastTweetTimer := prometheus.NewTimer(application.Metrics.RequestsDuration.WithLabelValues("/user_last_tweet"))

	application.Metrics.TotalRequests.WithLabelValues("/user_last_tweet").Inc()
	defer lastTweetTimer.ObserveDuration()

	startTime := time.Now()
	dbLog := &db.DBLog{AppName: "", Address: r.RemoteAddr, ArrivedAt: startTime, SentAt: time.Time{}, Req: db.Request{Method: r.Method, URI: r.RequestURI, Body: ""}, Resp: ""}

	defer db.SaveLog(dbLog, application.DB)
	defer r.Body.Close()
	defer com.TweetyLog(com.INFO, "Last Tweet Handler finished.")

	var userId com.ReqUserId

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "400 - Cannot read body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	bodyStr := string(body)
	dbLog.Req.Body = bodyStr

	err = json.Unmarshal(body, &userId)

	if err != nil {
		msg := "400 - Cannot unmarshal body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s. Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	dbLog.SentAt = userId.SentAt
	dbLog.AppName = userId.AppName

	tweetId, err := db.GetLastTweet(userId.UserId, application.DB)
	if err != nil {
		msg := "500 - Cannot get last tweet!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Cannot get last tweet!"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	js, err := json.Marshal(tweetId)
	if err != nil {
		msg := "500 - Cannot marshal data!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	if tweetId.Id == "" {
		com.TweetyLog(com.INFO, fmt.Sprintf("No last tweets for user id = %s.", userId.UserId))
	} else {
		com.TweetyLog(com.INFO, fmt.Sprintf("Returning last tweet with id = %s.", tweetId.Id))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(js)
	dbLog.Resp = "200 - OK!"
}

func (application *Application) tweetsSavingHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Tweets Saving Handler starting..."
	com.TweetyLog(com.INFO, msg)

	tweetsTimer := prometheus.NewTimer(application.Metrics.RequestsDuration.WithLabelValues("/user_tweets"))

	application.Metrics.TotalRequests.WithLabelValues("/user_tweets").Inc()
	defer tweetsTimer.ObserveDuration()

	dbLog := &db.DBLog{AppName: "", Address: r.RemoteAddr, ArrivedAt: time.Now(), SentAt: time.Time{}, Req: db.Request{Method: r.Method, URI: r.RequestURI, Body: ""}, Resp: ""}

	defer db.SaveLog(dbLog, application.DB)
	defer r.Body.Close()
	defer com.TweetyLog(com.INFO, "Tweets Saving Handler finished.")

	var tweets com.ReqTweetsForDB

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "400 - Cannot read body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	bodyStr := string(body)

	dbLog.Req.Body = bodyStr

	err = json.Unmarshal(body, &tweets)

	if err != nil {
		msg := "400 - Cannot unmarshal body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	dbLog.SentAt = tweets.SentAt
	dbLog.AppName = tweets.AppName

	for _, t := range tweets.Tweets {
		tweet := db.Tweet{Id: t.Id, Id_str: t.Id_str, UserId: tweets.UserId, Text: t.Text, Created_at: t.Created_at.Time, Url: t.Url}
		err = db.SaveTweet(tweet, application.DB)
		if err != nil {
			msg := "500 - Cannot insert tweet! id = " + t.Id_str
			dbLog.Resp = msg
			com.TweetyLog(com.ERROR, fmt.Sprintf("%s. Error: %s", msg, err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Cannot insert tweet"))
			continue
		}
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("Tweets saved for user with id = %s", tweets.UserId))

	err = db.UpdateWordCount(tweets.UserId, tweets.WordCount, application.DB)
	if err != nil {
		msg := "500 - Cannot update word count!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("Updated word count for user with id = %s", tweets.UserId))

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	dbLog.Resp = "200 - OK!"
}

func (application *Application) existsHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Exists Handler starting..."
	com.TweetyLog(com.INFO, msg)

	existsTimer := prometheus.NewTimer(application.Metrics.RequestsDuration.WithLabelValues("/user_exists"))
	application.Metrics.TotalRequests.WithLabelValues("/user_exists").Inc()

	defer existsTimer.ObserveDuration()

	dbLog := &db.DBLog{AppName: "", Address: r.RemoteAddr, ArrivedAt: time.Now(), SentAt: time.Time{}, Req: db.Request{Method: r.Method, URI: r.RequestURI, Body: ""}, Resp: ""}

	defer db.SaveLog(dbLog, application.DB)
	defer r.Body.Close()
	defer com.TweetyLog(com.INFO, "Exists Handler finished.")

	var userId com.ReqUserId

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "400 - Cannot read body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	bodyStr := string(body)
	dbLog.Req.Body = bodyStr

	err = json.Unmarshal(body, &userId)

	if err != nil {
		msg := "400 - Cannot unmarshal body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	dbLog.SentAt = userId.SentAt
	dbLog.AppName = userId.AppName

	existsResponse, err := db.UserExists(userId.UserId, application.DB)
	if err != nil {
		msg := "500 - Cannot check user with id = " + userId.UserId
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s. Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	js, err := json.Marshal(existsResponse)
	if err != nil {
		msg := "500 - Cannot marshal data!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("User exists returning value %t.", existsResponse.Exists))

	w.WriteHeader(http.StatusOK)
	w.Write(js)
	dbLog.Resp = "200 - OK!"
}

func (application *Application) metadataHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Metadata Handler starting..."
	com.TweetyLog(com.INFO, msg)

	metadataTimer := prometheus.NewTimer(application.Metrics.RequestsDuration.WithLabelValues("/user_metadata"))
	application.Metrics.TotalRequests.WithLabelValues("/user_metadata").Inc()
	defer metadataTimer.ObserveDuration()

	dbLog := &db.DBLog{AppName: "", Address: r.RemoteAddr, ArrivedAt: time.Now(), SentAt: time.Time{}, Req: db.Request{Method: r.Method, URI: r.RequestURI, Body: ""}, Resp: ""}

	defer db.SaveLog(dbLog, application.DB)
	defer r.Body.Close()
	defer com.TweetyLog(com.INFO, "Metadata Handler finished.")

	var user com.ReqUser

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "400 - Cannot read body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	bodyStr := string(body)
	dbLog.Req.Body = bodyStr

	err = json.Unmarshal(body, &user)

	if err != nil {
		msg := "400 - Cannot unmarshal body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	dbLog.SentAt = user.Sent_at
	dbLog.AppName = user.App_name

	err = db.SaveUserMetadata(user, application.DB)
	if err != nil {
		msg := "500 - Cannot save the user!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("Saved metadata for user with id = %s and name = %s.", user.Id_str, user.Name))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	dbLog.Resp = "200 - OK!"
}

func (application *Application) locationHandler(w http.ResponseWriter, r *http.Request) {
	msg := "Location Handler starting..."
	com.TweetyLog(com.INFO, msg)

	locationTimer := prometheus.NewTimer(application.Metrics.RequestsDuration.WithLabelValues("/location"))

	application.Metrics.TotalRequests.WithLabelValues("/location").Inc()
	defer locationTimer.ObserveDuration()

	dbLog := &db.DBLog{AppName: "", Address: r.RemoteAddr, ArrivedAt: time.Now(), SentAt: time.Time{}, Req: db.Request{Method: r.Method, URI: r.RequestURI, Body: ""}, Resp: ""}

	defer db.SaveLog(dbLog, application.DB)
	defer r.Body.Close()
	defer com.TweetyLog(com.INFO, "Location Handler finished.")

	//var user com.ReqUser
	var location com.ReqLocationForDB

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "400 - Cannot read body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	bodyStr := string(body)
	dbLog.Req.Body = bodyStr

	err = json.Unmarshal(body, &location)

	if err != nil {
		msg := "400 - Cannot unmarshal body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	dbLog.SentAt = location.SentAt
	dbLog.AppName = location.AppName

	//save location
	locationInfo := db.LocationInfo{Name: location.LocationInfo.Name, Languages: location.LocationInfo.Languages, Population: location.LocationInfo.Population, RegionalBlocks: location.LocationInfo.RegionalBlocs}
	err = db.SaveLocation(locationInfo, application.DB)
	if err != nil {
		msg := "500 - Cannot save the location!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("Saved location %s to database", location.LocationInfo.Name))
	//save location name to user

	locationName := db.LocationName{UserId: location.UserId, LocationName: location.LocationInfo.Name}
	err = db.SaveLocationNameToUser(locationName, application.DB)
	if err != nil {
		msg := "500 - Cannot save the location name to user table!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("Saved location %s to user id = %s.", location.LocationInfo.Name, location.UserId))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	dbLog.Resp = "200 - OK!"
}

func (application *Application) imagesHandler(w http.ResponseWriter, r *http.Request) {
	com.TweetyLog(com.INFO, "Images Handler starting...")

	imagesTimer := prometheus.NewTimer(application.Metrics.RequestsDuration.WithLabelValues("user_images"))

	application.Metrics.TotalRequests.WithLabelValues("/user_images").Inc()
	defer imagesTimer.ObserveDuration()

	dbLog := &db.DBLog{AppName: "", Address: r.RemoteAddr, ArrivedAt: time.Now(), SentAt: time.Time{}, Req: db.Request{Method: r.Method, URI: r.RequestURI, Body: ""}, Resp: ""}

	defer db.SaveLog(dbLog, application.DB)
	defer r.Body.Close()
	defer com.TweetyLog(com.INFO, "Images Handler finished.")

	//var user com.ReqUser
	var image com.ReqImagesForDB

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		msg := "400 - Cannot read body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	bodyStr := string(body)
	dbLog.Req.Body = bodyStr

	err = json.Unmarshal(body, &image)

	if err != nil {
		msg := "400 - Cannot unmarshal body!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	dbLog.SentAt = image.SentAt
	dbLog.AppName = image.AppName

	//save images
	err = db.UpdateUserImage(image, application.DB)
	if err != nil {
		msg := "500 - Cannot save the location!"
		dbLog.Resp = msg
		com.TweetyLog(com.ERROR, fmt.Sprintf("%s Error: %s", msg, err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	com.TweetyLog(com.INFO, fmt.Sprintf("Saved images to user id = %s.", image.UserId))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	dbLog.Resp = "200 - OK!"
}

func readConfig() (string, db.ServerConfig) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://172.17.41.118:2379"},
		DialTimeout: 5 * time.Second,
	})

	defer cli.Close()

	kv := clientv3.NewKV(cli)
	data, err := kv.Get(ctx, "DBSaverConfig")

	cancel()
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Etdc client error: %s", err.Error()))
	}

	/*data, err := ioutil.ReadFile(nameOfConfigFile)
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot read config file. Error: %s", err.Error()))
	}*/

	var config db.Configuration

	err = json.Unmarshal(data.Kvs[0].Value, &config)
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot unmarshal config file. Error: %s", err.Error()))
	}

	DBinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", config.Database.Host, config.Database.Port, config.Database.User, config.Database.Password, config.Database.DBname)
	ServerInfo := config.Server
	return DBinfo, ServerInfo
}

func (application *Application) report(done chan int) {
	ticker_hour := time.NewTicker(1 * time.Hour)
	ticker_day := time.NewTicker(24 * time.Hour)
	ticker_week := time.NewTicker(7 * 24 * time.Hour)
	ticker_month := time.NewTicker(30 * 24 * time.Hour)
	var reportType string

	for {
		select {

		// 1 hour
		case <-ticker_hour.C:
			reportType = "HOURLY"
			com.TweetyLog(com.INFO, "Hourly report starting...")
			err := db.SaveLogReport(db.LAST_HOUR, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save log report. Error: %s", err.Error()))
			}
			com.TweetyLog(com.INFO, "Hourly report finished.")

		// 1 day
		case <-ticker_day.C:
			com.TweetyLog(com.INFO, "Daily report starting...")
			reportType = "DAILY"
			err := db.SaveLogReport(db.LAST_DAY, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save log report. Error: %s", err.Error()))
			}
			err = db.SaveTweetReport(db.LAST_DAY, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save tweet report. Error: %s", err.Error()))
			}
			err = db.SaveLocationReport(db.LAST_DAY, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save location report. Error: %s", err.Error()))
			}
			com.TweetyLog(com.INFO, "Daily report finished.")

		// 1 week
		case <-ticker_week.C:
			reportType = "WEEKLY"
			com.TweetyLog(com.INFO, "Weekly report starting...")
			err := db.SaveLogReport(db.LAST_WEEK, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save log report. Error: %s", err.Error()))
			}
			err = db.SaveTweetReport(db.LAST_WEEK, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save tweet report. Error: %s", err.Error()))
			}
			err = db.SaveLocationReport(db.LAST_WEEK, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save location report. Error: %s", err.Error()))
			}
			com.TweetyLog(com.INFO, "Weekly report finished.")

		// 1 month
		case <-ticker_month.C:
			reportType = "MONTHLY"
			com.TweetyLog(com.INFO, "Monthly report starting...")
			err := db.SaveTweetReport(db.LAST_MONTH, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save tweet report. Error: %s", err.Error()))
			}
			err = db.SaveLocationReport(db.LAST_MONTH, reportType, application.DB)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot save location report. Error: %s", err.Error()))
			}
			com.TweetyLog(com.INFO, "Monthly report finished.")
		case <-done:
			return
		}
	}
}

func setUpMetrics() Metrics {
	totalRequests := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "NumberOfRequests",
		Help: "The total number of requests on certain endpoint.",
	}, []string{"endpoint"})

	RequestDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "DurationOfRequests",
		Help:    "Duration of requests on certain endpoint.",
		Buckets: prometheus.LinearBuckets(0, 2, 10),
	}, []string{"endpoint"})

	metrics := Metrics{TotalRequests: totalRequests,
		RequestsDuration: RequestDuration}

	return metrics
}

func setUpAppInfo() Application {
	var application Application

	/*configPath := "/tmp/config.json"
	if isWindowsOs() {
		configPath = "config.json"
	}*/

	DBInfo, ServerInfo := readConfig()

	DB, err := db.ConnectToDB(DBInfo)

	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot connect to database. Error: %s", err.Error()))
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/user_metadata", application.metadataHandler)
	mux.HandleFunc("/user_last_tweet", application.lastTweetHandler)
	mux.HandleFunc("/user_exists", application.existsHandler)
	mux.HandleFunc("/user_tweets", application.tweetsSavingHandler)
	mux.HandleFunc("/location", application.locationHandler)
	mux.HandleFunc("/user_images", application.imagesHandler)
	mux.Handle("/metrics", promhttp.Handler())

	s := &http.Server{
		Addr:    fmt.Sprintf("%s%s", ServerInfo.Host, ServerInfo.Port),
		Handler: mux,
	}

	application.DB = DB
	application.Server = s
	application.Metrics = setUpMetrics()

	return application
}

/*func isWindowsOs() bool {
	boolPtr := flag.Bool("win", false, "os type")
	flag.Parse()
	return *boolPtr
} */

func main() {

	application := setUpAppInfo()
	defer application.DB.Close()

	bye := make(chan os.Signal)
	signal.Notify(bye, os.Interrupt, syscall.SIGTERM)

	go func() {
		com.TweetyLog(com.INFO, fmt.Sprintf("Server starting..."))
		err := application.Server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			com.TweetyLog(com.ERROR, fmt.Sprintf("Server error: %q", err.Error()))
		}
	}()

	ch := make(chan int, 1)
	go application.report(ch)

	// wait for the SIGINT
	sig := <-bye
	com.TweetyLog(com.INFO, fmt.Sprintf("Detected os signal %s.", sig))

	ctx1, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	err := application.Server.Shutdown(ctx1)
	cancel()
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Error %s.", err.Error()))
	}

	ch <- 1

}
