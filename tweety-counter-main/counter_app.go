package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	clientv3 "go.etcd.io/etcd/client/v3"

	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
	tw "gitlab.com/leapbit-practice/tweety-lib-twitter/twitter"
)

const (
	Version = "1.0.0"
	Author  = "Josip Srzic"
	AppName = "Tweety-Counter"

	httpRequestTemplate = "http://%s/%s"
	lastTweetEndpoint   = "user_last_tweet"
	sendTweetEndpoint   = "user_tweets"
	sendImagesEndpoint  = "user_images"
	etcdEndpoint        = "tweety-database-tck-test.demobet.lan:2379"
)

type App struct {
	Ctw     HttpClientTW `json:"client_twitter"`
	Cdb     HttpClientDB `json:"client_database"`
	Metrics Metrics      `json:"metrics"`
}

type Metrics struct {
	UserIdsTotalRequests    prometheus.Counter
	UserIdsRequestsDuration prometheus.Histogram
	TotalSentRequests       *prometheus.CounterVec
	SentRequestsDuration    *prometheus.HistogramVec
}

type HttpRequestClient struct {
	Client http.Client
}

type HttpClientTW struct {
	RequestClient HttpRequestClient
	TweetNo       uint64 `json:"tweet_no"`
	Bearer        string `json:"bearer_token"`
}

type HttpClientDB struct {
	RequestClient HttpRequestClient
	DbIpAndPort   string `json:"db_ip_port"`
}

type Config struct {
	TweetNo     uint64 `json:"tweet_no"`
	Bearer      string `json:"bearer_token"`
	DbIpAndPort string `json:"db_ip_port"`
}

func setUpMetrics() Metrics {
	UserIdsTotalRequests := promauto.NewCounter(prometheus.CounterOpts{
		Name: "UserIdsTotalRequests",
		Help: "The total number of processed requests on user_ids endpoint",
	})

	UserIdsRequestsDuration := promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "DurationOfUserIdsRequests",
		Help:    "Duration of processed requests on user_ids endpoint.",
		Buckets: prometheus.LinearBuckets(0, 2, 10),
	})

	TotalSentRequests := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "TotalSentRequests",
		Help: "The total number of sent requests with certain method.",
	}, []string{"method"})

	SentRequestsDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "DurationOfSentRequests",
		Help:    "Duration of requests with certain method.",
		Buckets: prometheus.LinearBuckets(0, 2, 10),
	}, []string{"method"})

	metrics := Metrics{
		UserIdsTotalRequests:    UserIdsTotalRequests,
		UserIdsRequestsDuration: UserIdsRequestsDuration,
		TotalSentRequests:       TotalSentRequests,
		SentRequestsDuration:    SentRequestsDuration,
	}

	return metrics
}

func NewHttpClientTW(tweetNo uint64, bearer string) HttpClientTW {
	var ctw HttpClientTW
	ctw.RequestClient = HttpRequestClient{Client: http.Client{Timeout: time.Duration(15) * time.Second}}
	ctw.TweetNo = tweetNo
	ctw.Bearer = bearer
	return ctw
}

func NewHttpClientDB(dbIpAndPort string) HttpClientDB {
	var cdb HttpClientDB
	cdb.RequestClient = HttpRequestClient{Client: http.Client{Timeout: time.Duration(15) * time.Second}}
	cdb.DbIpAndPort = dbIpAndPort
	return cdb
}

func (rc *HttpRequestClient) DownloadFile(url string) ([]byte, error) {
	img, err, errMsg := rc.performRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error while downloading user image from url: %s Error: %s", url, err.Error())
	}
	if errMsg != nil {
		return nil, fmt.Errorf("error while downloading user image from url: %s Error: %s", url, errMsg.Error())
	}

	return img, nil
}

func ZipFiles(imgNames []string, imgData [][]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	for i := 0; i < len(imgNames); i++ {
		zipEntry, err := zipWriter.Create(imgNames[i])
		if err != nil {
			return nil, fmt.Errorf("failed to zip %s Error: %s", imgNames[i], err.Error())
		}

		_, err = zipEntry.Write(imgData[i])
		if err != nil {
			return nil, fmt.Errorf("failed to zip %s Error: %s", imgNames[i], err.Error())
		}
	}
	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zipWriter. Error: %s", err.Error())
	}

	return buf.Bytes(), nil
}

func (rc *HttpRequestClient) performRequest(httpMethod string, path string, data interface{}) (body []byte, err error, errMsg error) {
	var jsonRequestData []byte
	if data != nil {
		jsonRequestData, errMsg = json.Marshal(data)
		if errMsg != nil {
			return nil, nil, fmt.Errorf("cannot marshal request data. Error: %s", errMsg.Error())
		}
	}
	req, errMsg := http.NewRequest(httpMethod, path, bytes.NewBuffer(jsonRequestData))
	if errMsg != nil {
		return nil, nil, fmt.Errorf("cannot create a request. Error: %s", errMsg.Error())
	}

	resp, err := rc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do the given request. Error: %s", err.Error()), nil
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if errMsg != nil {
		return nil, nil, fmt.Errorf("cannot read request body. Error: %s", errMsg.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status code is not 200 OK"), nil
	}

	return body, nil, nil
}

func rankMostUsedWords(tweets []tw.RespTwitterApiTweet) []com.KvPair {
	wordCount := make(map[string]uint64)

	var words []string
	for i := range tweets {
		words = strings.Fields(tweets[i].Text)
		for j := range words {
			wordCount[words[j]]++
		}
	}

	var kvPairs []com.KvPair
	for k, v := range wordCount {
		kvPairs = append(kvPairs, com.KvPair{Word: k, Count: v})
	}
	sort.Slice(kvPairs, func(i, j int) bool {
		return kvPairs[i].Count > kvPairs[j].Count
	})

	upperBound := 10
	if len(kvPairs) < 10 {
		upperBound = len(kvPairs)
	}
	kvPairs = kvPairs[:upperBound]

	return kvPairs
}

func (app *App) getImageUrlsFromTwitter(userId string) (string, string, error) {
	var respImages tw.RespTwitterApiImages
	var err error
	var errMsg error

	app.Metrics.TotalSentRequests.WithLabelValues("getImageUrlsFromTwitter").Inc()
	methodTimer := prometheus.NewTimer(app.Metrics.SentRequestsDuration.WithLabelValues("getImageUrlsFromTwitter"))
	defer methodTimer.ObserveDuration()

	com.TweetyLog(com.INFO, fmt.Sprintf("Getting image urls from twitter for user %s...", userId))
	respImages, err, errMsg = tw.UserGetImageUrls(userId, &app.Ctw.RequestClient.Client, app.Ctw.Bearer)
	if errMsg != nil {
		return "", "", fmt.Errorf("internal error occurred while communicating with Twitter. Error: %s", errMsg.Error())
	}
	if err != nil {
		return "", "", fmt.Errorf("error occurred while communicating with Twitter. Error: %s", err.Error())
	}
	com.TweetyLog(com.INFO, fmt.Sprintf("Getting image urls from twitter for user %s DONE.", userId))

	return respImages.UrlProfileImage, respImages.UrlBanner, nil
}

func (app *App) sendImagesToDB(userId string, zippedData []byte) (err error, errMsg error) {
	app.Metrics.TotalSentRequests.WithLabelValues("sendImagesToDB").Inc()
	methodTimer := prometheus.NewTimer(app.Metrics.SentRequestsDuration.WithLabelValues("sendImagesToDB"))
	defer methodTimer.ObserveDuration()

	images := com.ReqImagesForDB{
		UserId:     userId,
		UserImages: zippedData,
		AppName:    AppName,
		SentAt:     time.Now(),
	}

	_, err, errMsg = app.Cdb.RequestClient.performRequest(http.MethodPost, fmt.Sprintf(httpRequestTemplate, app.Cdb.DbIpAndPort, sendImagesEndpoint), images)

	return err, errMsg
}

func (app *App) sendTweetsToDB(userTweets []tw.RespTwitterApiTweet, rankedWordCount []com.KvPair) (err error, errMsg error) {
	app.Metrics.TotalSentRequests.WithLabelValues("sendTweetsToDB").Inc()
	methodTimer := prometheus.NewTimer(app.Metrics.SentRequestsDuration.WithLabelValues("sendTweetsToDB"))
	defer methodTimer.ObserveDuration()

	reqTweetsForDB := com.ReqTweetsForDB{
		UserId:    strconv.FormatUint(userTweets[0].User.Id, 10),
		Tweets:    userTweets,
		WordCount: rankedWordCount,
		AppName:   AppName,
		SentAt:    time.Now(),
	}

	_, err, errMsg = app.Cdb.RequestClient.performRequest(http.MethodPost, fmt.Sprintf(httpRequestTemplate, app.Cdb.DbIpAndPort, sendTweetEndpoint), reqTweetsForDB)

	return err, errMsg
}

func (app *App) hasNewTweets(lastTweet tw.RespTwitterApiTweet) (hasNew bool, err error, errMsg error) {
	app.Metrics.TotalSentRequests.WithLabelValues("hasNewTweets").Inc()
	methodTimer := prometheus.NewTimer(app.Metrics.SentRequestsDuration.WithLabelValues("hasNewTweets"))
	defer methodTimer.ObserveDuration()

	userId := com.ReqUserId{
		UserId:  strconv.FormatUint(lastTweet.User.Id, 10),
		AppName: AppName,
		SentAt:  time.Now(),
	}
	var respTweetId com.RespTweetId

	body, err, errMsg := app.Cdb.RequestClient.performRequest(http.MethodGet, fmt.Sprintf(httpRequestTemplate, app.Cdb.DbIpAndPort, lastTweetEndpoint), userId)
	if err != nil {
		return false, fmt.Errorf("cannot perform request. Error: %s", err.Error()), nil
	}

	if errMsg != nil {
		return false, nil, fmt.Errorf("cannot perform request. Error: %s", errMsg.Error())
	}

	errMsg = json.Unmarshal(body, &respTweetId)
	if errMsg != nil {
		return false, nil, fmt.Errorf("cannot unmarshal body. Error: %s", errMsg.Error())
	}

	if respTweetId.Id == "" {
		return true, nil, nil
	}

	if respTweetId.Id == lastTweet.Id_str {
		return false, nil, nil
	}

	return true, nil, nil
}

func readConfigEtcd(config *Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		com.TweetyLog(com.ERROR, "Cannot create etcd client. Error: %s", err.Error())
	}
	defer cli.Close()
	kv := clientv3.NewKV(cli)
	gr, _ := kv.Get(ctx, "CounterConfig")
	cancel()
	configData := gr.Kvs[0].Value
	json.Unmarshal(configData, config)
}

func (app *App) getTweetsFromTwitter(userId string) ([]tw.RespTwitterApiTweet, error) {
	app.Metrics.TotalSentRequests.WithLabelValues("getTweetsFromTwitter").Inc()
	methodTimer := prometheus.NewTimer(app.Metrics.SentRequestsDuration.WithLabelValues("getTweetsFromTwitter"))
	defer methodTimer.ObserveDuration()

	var tweets []tw.RespTwitterApiTweet
	var err error
	var errMsg error

	com.TweetyLog(com.INFO, fmt.Sprintf("Getting tweets from Twitter for user %s...", userId))
	tweets, err, errMsg = tw.UserGetTweets(userId, app.Ctw.TweetNo, &app.Ctw.RequestClient.Client, app.Ctw.Bearer)
	if errMsg != nil {
		return nil, fmt.Errorf("internal error occurred while communicating with Twitter. Error: %s", errMsg.Error())
	}
	if err != nil {
		return nil, fmt.Errorf("error occurred while communicating with Twitter. Error: %s", err.Error())
	}
	com.TweetyLog(com.INFO, fmt.Sprintf("Getting tweets from Twitter for user %s DONE.", userId))

	return tweets, nil
}

func (app *App) checkAndSendTweetsToDB(userId string, tweets []tw.RespTwitterApiTweet) error {
	var err error
	var errMsg error
	var hasNew bool

	com.TweetyLog(com.INFO, fmt.Sprintf("Checking if user %s has any new tweets...", userId))
	if len(tweets) > 0 {
		hasNew, err, errMsg = app.hasNewTweets(tweets[0])
		if errMsg != nil {
			return fmt.Errorf("internal error occurred while communicating with database. Error: %s", errMsg.Error())
		}
		if err != nil {
			return fmt.Errorf("error occurred while communicating with database. Error: %s", err.Error())
		}
	} else {
		hasNew = false
	}
	com.TweetyLog(com.INFO, fmt.Sprintf("Checking if user %s has any new tweets DONE.", userId))

	if hasNew {
		com.TweetyLog(com.INFO, fmt.Sprintf("Ranking most used words from user %s...", userId))
		rankedWordCount := rankMostUsedWords(tweets)
		com.TweetyLog(com.INFO, fmt.Sprintf("Ranking most used words from user %s DONE.", userId))

		com.TweetyLog(com.INFO, fmt.Sprintf("Sending tweets from user %s to database...", userId))
		err, errMsg := app.sendTweetsToDB(tweets, rankedWordCount)
		if errMsg != nil {
			return fmt.Errorf("internal error occurred while communicating with database. Error: %s", errMsg.Error())
		}
		if err != nil {
			return fmt.Errorf("error occurred while communicating with database. Error: %s", err.Error())
		}
		com.TweetyLog(com.INFO, fmt.Sprintf("Sending tweets from user %s to database DONE.", userId))
	} else {
		com.TweetyLog(com.INFO, fmt.Sprintf("User %s doesn't have any new tweets.", userId))
	}

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (app *App) processUserIds(w http.ResponseWriter, req *http.Request) {
	app.Metrics.UserIdsTotalRequests.Inc()
	userIdsTimer := prometheus.NewTimer(app.Metrics.UserIdsRequestsDuration)
	defer userIdsTimer.ObserveDuration()

	com.TweetyLog(com.INFO, "New request received on /user_ids")
	var reqFriends tw.ReqFriends
	var respDoneFriends tw.RespDoneFriends

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot read body. Error: %s Sending response with code %v", err.Error(), http.StatusBadRequest))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &reqFriends)
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot unmarshal body. Error: %s Sending error response with code %v", err.Error(), http.StatusBadRequest))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	com.TweetyLog(com.INFO, fmt.Sprintf("Unpacked user_ids from request: %v", reqFriends.Friends_ids))
	w.Header().Set("Content-Type", "application/json")

	inputChannel := make(chan string, len(reqFriends.Friends_ids))
	outputChannel := make(chan string, len(reqFriends.Friends_ids))
	allDoneChannel := make(chan bool, 1)

	for _, id := range reqFriends.Friends_ids {
		inputChannel <- id
	}
	close(inputChannel)

	doneCounter := len(reqFriends.Friends_ids)
	mutex := &sync.Mutex{}
	com.TweetyLog(com.INFO, "Starting workers...")
	for i := 0; i < 10; i++ {
		go app.userIdWorker(inputChannel, outputChannel, allDoneChannel, mutex, &doneCounter)
	}

	select {
	case <-allDoneChannel:
		com.TweetyLog(com.INFO, fmt.Sprintf("All user ids sucessfully processed. Sending unsuccessful user ids in response: %v", respDoneFriends.Friends_ids))
		resp, err := json.Marshal(respDoneFriends)
		if err != nil {
			com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot create response body. Error: %s Sending error response with code %v", err.Error(), http.StatusInternalServerError))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(resp))

	case <-time.After(20 * time.Second):
		var doneIds []string
		close(outputChannel)
		for doneId := range outputChannel {
			doneIds = append(doneIds, doneId)
		}
		for _, id := range reqFriends.Friends_ids {
			if !contains(doneIds, id) {
				respDoneFriends.Friends_ids = append(respDoneFriends.Friends_ids, id)
			}
		}
		com.TweetyLog(com.INFO, fmt.Sprintf("20 seconds have passed since the request came. Sending unsuccessful user ids in response: %v", respDoneFriends.Friends_ids))
		resp, err := json.Marshal(respDoneFriends)
		if err != nil {
			com.TweetyLog(com.ERROR, fmt.Sprintf("Cannot create response body. Error: %s Sending error response with code %v", err.Error(), http.StatusInternalServerError))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(resp))

	}
}

func (app *App) userIdWorker(inputChannel chan string, outputChannel chan string, allDoneChannel chan bool, mutex *sync.Mutex, doneCounter *int) {
	for userId := range inputChannel {
		urlProfileImage, urlBanner, err := app.getImageUrlsFromTwitter(userId)
		if err != nil {
			com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
			continue
		}

		imgNames := make([]string, 0)
		dataToZip := make([][]byte, 0)

		if urlProfileImage != "" {
			com.TweetyLog(com.INFO, fmt.Sprintf("Downloading profile image of user %s...", userId))
			img1, err := app.Ctw.RequestClient.DownloadFile(urlProfileImage)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
				continue
			}
			com.TweetyLog(com.INFO, fmt.Sprintf("Downloading profile image of user %s DONE.", userId))
			imgNames = append(imgNames, "profile_image.png")
			dataToZip = append(dataToZip, img1)
		}

		if urlBanner != "" {
			com.TweetyLog(com.INFO, fmt.Sprintf("Downloading profile banner of user %s...", userId))
			img2, err := app.Ctw.RequestClient.DownloadFile(urlBanner)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
				continue
			}
			com.TweetyLog(com.INFO, fmt.Sprintf("Downloading profile banner of user %s DONE.", userId))
			imgNames = append(imgNames, "banner.png")
			dataToZip = append(dataToZip, img2)
		}

		if len(dataToZip) > 0 {
			com.TweetyLog(com.INFO, fmt.Sprintf("Zipping images of user %s...", userId))
			zippedData, err := ZipFiles(imgNames, dataToZip)
			if err != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
				continue
			}
			com.TweetyLog(com.INFO, fmt.Sprintf("Zipping images of user %s DONE.", userId))

			com.TweetyLog(com.INFO, fmt.Sprintf("Sending images of user %s to database...", userId))
			err, errMsg := app.sendImagesToDB(userId, zippedData)
			if err != nil || errMsg != nil {
				com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
				continue
			}
			com.TweetyLog(com.INFO, fmt.Sprintf("Sending images of user %s to database DONE.", userId))
		}

		tweets, err := app.getTweetsFromTwitter(userId)
		if err != nil {
			com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
			continue
		}
		err = app.checkAndSendTweetsToDB(userId, tweets)
		if err != nil {
			com.TweetyLog(com.ERROR, fmt.Sprintf("Worker failed to process user id: %s. Error: %s", userId, err.Error()))
			continue
		}
		outputChannel <- userId

		mutex.Lock()
		*doneCounter = *doneCounter - 1
		if *doneCounter == 0 {
			allDoneChannel <- true
		}
		mutex.Unlock()
	}
}

func main() {
	com.TweetyLog(com.INFO, fmt.Sprintf("%s initialized.", AppName))
	com.TweetyLog(com.INFO, fmt.Sprintf("Version: %s\nAuthor: %s\n", Version, Author))

	com.TweetyLog(com.INFO, "Creating clients and loading configuration...")
	var config Config
	readConfigEtcd(&config)
	ctw := NewHttpClientTW(config.TweetNo, config.Bearer)
	cdb := NewHttpClientDB(config.DbIpAndPort)
	metrics := setUpMetrics()
	app := &App{
		Ctw:     ctw,
		Cdb:     cdb,
		Metrics: metrics,
	}
	com.TweetyLog(com.INFO, "Clients created and configuration loaded.")
	fmt.Printf("\n\n")

	http.HandleFunc("/user_ids", app.processUserIds)
	http.Handle("/metrics", promhttp.Handler())

	log.Fatal(http.ListenAndServe(":8090", nil))

}
