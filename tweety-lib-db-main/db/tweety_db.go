package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	pq "github.com/lib/pq"
	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
)

type Configuration struct {
	Server   ServerConfig `json:"server"`
	Database DBConfig     `json:"database"`
}

type DBConfig struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	User     string `json:"User"`
	Password string `json:"Password"`
	DBname   string `json:"DBname"`
}

type ServerConfig struct {
	Host string `json:"server"`
	Port string `json:"port"`
}

type Request struct {
	Method string `json:"method"`
	URI    string `json:"URI"`
	Body   string `json:"body"`
}

type DBLog struct {
	AppName   string    `json:"method"`
	Address   string    `json:"address"`
	ArrivedAt time.Time `json:"arrived_at"`
	SentAt    time.Time `json:"sent_at"`
	Req       Request   `json:"request"`
	Resp      string    `json:"response"`
}

type Tweet struct {
	Id         uint64
	Id_str     string
	UserId     string
	Text       string
	Created_at time.Time
	Url        string
}

type TweetCounts struct {
	counts map[string]uint32
}

type LocationName struct {
	UserId       string
	LocationName string
}

type LocationInfo struct {
	Name           string
	Languages      []com.LanguagesType
	Population     int64
	RegionalBlocks []com.RegionalBlocsType
}

type RegionalBlockCounts struct {
	RegionalBlock  string `json:"regional_block_acronym"`
	NumberOfTweets uint64 `json:"number_of_tweets"`
}

type ReportRequestDuration struct {
	LogId           string `json:"log_id"`
	ApplicationName string `json:"app_name"`
	//	Request         Request `json:"request"`
	Duration float64 `json:"duration"`
}

type ReportErrorRequest struct {
	LogId           string `json:"log_id"`
	ApplicationName string `json:"app_name"`
	//Request         Request `json:"request"`
	Response string `json:"response"`
}

type TopDurationsReport struct {
	Requests []ReportRequestDuration `json:"requests"`
}

type TopErrorsReport struct {
	Request []ReportErrorRequest `json:"requests"`
}

type TweetLenghts struct {
	UserName string `json:"user_name"`
	Lenght   uint64 `json:"tweet_lenght"`
}

const (
	LAST_HOUR  = -1 * time.Hour
	LAST_DAY   = -24 * time.Hour
	LAST_WEEK  = -24 * 7 * time.Hour
	LAST_MONTH = -30 * 24 * time.Hour

	insert_user = `INSERT INTO public.user (
		id, 
		id_str, 
		name, 
		screen_name, 
		location, 
		url, 
		description,
		protected,
		verified,
		followers_count,
		friends_count,
		statuses_count,
		created_at,
		list_of_follower_ids,
		word_counts,
		last_modified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW())
		ON CONFLICT (id_str)
		DO UPDATE SET id = $1, name = $3, screen_name = $4, location = $5, url = $6, description = $7, protected = $8, verified = $9, followers_count = $10, friends_count = $11, statuses_count = $12, created_at = $13, list_of_follower_ids = $14, last_modified = NOW();`

	insert_tweet = `INSERT INTO public.tweet (
		tweet_id, 
		tweet_id_str,
		user_id_str, 
		text,
		created_at,
		last_modified,
		url)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
		ON CONFLICT (tweet_id_str)
		DO UPDATE SET last_modified = NOW();`

	insert_log = `INSERT INTO public.log (
		app_name, 
		addr,
		sent_at,
		arrived_at, 
		request,
		response)
		VALUES ($1, $2, $3, $4, $5, $6)`

	insert_location = `INSERT INTO public.location (
		name, 
		languages,
		regional_blocks,
		population)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name) DO NOTHING;`

	insert_log_report = `INSERT INTO public.log_report(
		app_most_requests,
		top_error_requests,
		top_longest_requests,
		top_shortest_requests,
		type,
		reported_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`

	insert_tweet_report = `INSERT INTO public.tweet_report(
		most_tweets,
		largest_tweets,
		most_used_words,
		type,
		reported_at)
		VALUES ($1, $2, $3, $4, NOW())`

	insert_location_report = `INSERT INTO public.location_report(
		top_tweet_location,
		top_tweet_regional_blocks,
		most_spoken_languages,
		total_population,
		type,
		reported_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`

	//update_user_location_name = `UPDATE public.user SET location_name = $2 WHERE id_str = $1;`

	update_user_location_name = `INSERT INTO public.user (
		id_str, 
		location_name,
		last_modified) 
		VALUES ($1, $2, NOW()) 
		ON CONFLICT (id_str) 
		DO UPDATE SET location_name = $2, last_modified = NOW();`

	update_user_image = `INSERT INTO public.user (
		id_str, 
		images,
		last_modified) 
		VALUES ($1, $2, NOW()) 
		ON CONFLICT (id_str) 
		DO UPDATE SET images = $2, last_modified = NOW();`

	//update_wc = `UPDATE public.user SET word_counts = $2 WHERE id_str = $1;`
	update_wc = `INSERT INTO public.user (
		id_str, 
		word_counts,
		last_modified) 
		VALUES ($1, $2, NOW()) 
		ON CONFLICT (id_str) 
		DO UPDATE SET word_counts = $2, last_modified = NOW();`

	get_last_tweet = `
	SELECT A.tweet_id_str
		FROM PUBLIC.tweet A
		JOIN PUBLIC.user B
	ON A.user_id_str = B.id_str
	WHERE A.user_id_str LIKE $1
	ORDER BY A.created_at DESC
	FETCH FIRST 1 ROWS ONLY
	`
	get_last_modified_by_id = `SELECT last_modified
	FROM PUBLIC.user
	WHERE id_str LIKE $1;`

	check_if_user_exists_by_id = `SELECT EXISTS (SELECT * 
	FROM PUBLIC.user
	WHERE id_str LIKE $1 AND name IS NOT NULL);`

	get_tweet_counts_last_period = `SELECT B.name, COUNT(*) tweet_count
	FROM PUBLIC.tweet A
	JOIN PUBLIC.user B
	ON A.user_id_str = B.id_str
	WHERE A.last_modified >= $1 AND B.name IS NOT NULL
	GROUP BY B.name
	ORDER BY tweet_count DESC
	FETCH FIRST 10 ROWS ONLY`

	get_largest_tweets_last_period = `SELECT B.name, LENGTH(A.text)
	FROM PUBLIC.tweet A
	JOIN PUBLIC.user B
	ON A.user_id_str = B.id_str
	WHERE A.last_modified >= $1 AND B.name IS NOT NULL
	ORDER BY length DESC
	FETCH FIRST 10 ROWS ONLY `

	get_number_of_requests_by_app_last_period = `SELECT app_name, COUNT(*) request_count
	FROM PUBLIC.log
	WHERE arrived_at >= $1
	GROUP BY app_name
	ORDER BY request_count DESC
	FETCH FIRST 1 ROWS ONLY`

	get_error_responses_last_period = `SELECT id, app_name, response   
	FROM PUBLIC.log
	WHERE arrived_at >= $1 AND (response LIKE '400%' OR response LIKE '500%')
	ORDER BY sent_at DESC
	FETCH FIRST 10 ROWS ONLY`

	get_longest_requests_last_period = `SELECT id, app_name, ABS(EXTRACT(EPOCH FROM (arrived_at - sent_at))) time_diff
	FROM PUBLIC.log
	WHERE (arrived_at >= $1) AND EXTRACT (YEAR FROM sent_at) != 1
	ORDER BY time_diff DESC
	FETCH FIRST 10 ROWS ONLY`

	get_shortest_requests_last_period = `SELECT id, app_name, ABS(EXTRACT(EPOCH FROM (arrived_at - sent_at))) time_diff
	FROM PUBLIC.log
	WHERE (arrived_at >= $1) AND EXTRACT (YEAR FROM sent_at) != 1
	ORDER BY time_diff 
	FETCH FIRST 10 ROWS ONLY`

	get_top_tweet_locations = `SELECT A.name, COUNT(*) num_of_tweets, A.population, A.languages
	FROM PUBLIC.location A
	JOIN PUBLIC.user B 
	ON A.name = B.location_name
	JOIN PUBLIC.tweet C
	ON B.id_str LIKE C.user_id_str
	WHERE (C.last_modified >= $1)
	GROUP BY A.name
	ORDER BY num_of_tweets DESC
	FETCH FIRST 10 ROWS ONLY`

	get_tweets_last_period = `SELECT text
	FROM PUBLIC.tweet
	WHERE last_modified >= $1`

	get_tweets_regional_blocks_last_period = `SELECT A.regional_blocks
	FROM PUBLIC.location A
	JOIN PUBLIC.user B 
	ON A.name = B.location_name
	JOIN PUBLIC.tweet C
	ON B.id_str LIKE C.user_id_str
	WHERE C.last_modified >= $1`
)

func ConnectToDB(DBinfo string) (*sql.DB, error) {

	db, err := sql.Open("postgres", DBinfo)

	if err != nil {
		log.Fatalln("Error while opening database, err: ", err, time.Now())
	}

	err = db.Ping()

	return db, err
}

func SaveLog(logInfo *DBLog, db *sql.DB) {

	jsonRequest, err := json.Marshal(logInfo.Req)
	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Error while logging, err: %s", err.Error()))
		com.TweetyLog(com.INFO, "Continuing without logging")
		return
	}

	_, err = db.Exec(insert_log, logInfo.AppName, logInfo.Address, logInfo.SentAt, logInfo.ArrivedAt, jsonRequest, logInfo.Resp)

	if err != nil {
		com.TweetyLog(com.ERROR, fmt.Sprintf("Error while logging, err: %s", err.Error()))
		com.TweetyLog(com.INFO, "Continuing without logging")
	}
}

func SaveTweet(t Tweet, db *sql.DB) error {
	_, err := db.Exec(insert_tweet, t.Id, t.Id_str, t.UserId, t.Text, t.Created_at, t.Url)
	return err
}

func GetLastTweet(userId string, db *sql.DB) (tweetId com.RespTweetId, err error) {

	err = db.QueryRow(get_last_tweet, userId).Scan(&tweetId.Id)

	if err == sql.ErrNoRows {
		//log.Println("Last tweet doesn't exist for user id ", userId, ", returning empty tweet id.")
		tweetId.Id = ""
		return tweetId, nil
	}

	return tweetId, err
}

func UserExists(userId string, db *sql.DB) (com.RespUserExists, error) {

	var existsResponse com.RespUserExists

	err := db.QueryRow(check_if_user_exists_by_id, userId).Scan(&existsResponse.Exists)

	if err != nil {
		return existsResponse, err
	}

	if existsResponse.Exists {
		err = db.QueryRow(get_last_modified_by_id, userId).Scan(&existsResponse.Last_modified)
		if err != nil {
			return existsResponse, err
		}
	}

	return existsResponse, nil
}

func SaveUserMetadata(user com.ReqUser, db *sql.DB) error {
	_, err := db.Exec(insert_user, user.Id, user.Id_str, user.Name, user.Screen_name, user.Location, user.URL, user.Description,
		user.Protected, user.Verified, user.Followers_count, user.Friends_count, user.Statuses_count, user.Created_at, pq.Array(user.Followers_id), nil)

	return err
}

func UpdateWordCount(userId string, fWordCount []com.KvPair, db *sql.DB) error {

	wcJson, err := json.Marshal(fWordCount)
	if err != nil {
		return err
	}

	_, err = db.Exec(update_wc, userId, wcJson)
	return err
}

func UpdateUserImage(images com.ReqImagesForDB, db *sql.DB) error {

	_, err := db.Exec(update_user_image, images.UserId, images.UserImages)
	return err
}

func GetTweetCountsInLastPeriod(period time.Duration, db *sql.DB) (map[string]uint64, error) {

	counts := make(map[string]uint64)

	rows, err := db.Query(get_tweet_counts_last_period, time.Now().Add(period))

	if err != nil {
		return counts, err
	}

	defer rows.Close()

	for rows.Next() {
		var userName string
		var count uint64
		if err := rows.Scan(&userName, &count); err != nil {
			return counts, err
		}
		counts[userName] = count
	}

	return counts, nil
}

func GetLargestTweetsInLastPeriod(period time.Duration, db *sql.DB) ([]TweetLenghts, error) {

	//lengths := make(map[string]uint64)
	var lengths []TweetLenghts

	rows, err := db.Query(get_largest_tweets_last_period, time.Now().Add(period))

	if err != nil {
		return lengths, err
	}

	defer rows.Close()

	for rows.Next() {
		var userName string
		var len uint64
		if err := rows.Scan(&userName, &len); err != nil {
			return lengths, err
		}
		//lengths[userName] = len
		lengths = append(lengths, TweetLenghts{UserName: userName, Lenght: len})
	}

	return lengths, nil
}

func GetTopWordCountsInLastPeriod(period time.Duration, db *sql.DB) ([]com.KvPair, error) {

	var kvPairs []com.KvPair

	rows, err := db.Query(get_tweets_last_period, time.Now().Add(period))

	if err != nil {
		return kvPairs, err
	}

	defer rows.Close()

	wordCounts := make(map[string]uint64)

	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return kvPairs, err
		}
		words := strings.Fields(text)
		for _, word := range words {
			wordCounts[word]++
		}
	}

	for k, v := range wordCounts {
		kvPairs = append(kvPairs, com.KvPair{Word: k, Count: v})
	}
	sort.Slice(kvPairs, func(i, j int) bool {
		return kvPairs[i].Count > kvPairs[j].Count
	})

	if len(kvPairs) >= 10 {
		kvPairs = kvPairs[:10]
	}

	return kvPairs, nil
}

func GetNumberOfRequestsByAppInLastPeriod(period time.Duration, db *sql.DB) (string, error) {

	var appName string
	var count uint64

	err := db.QueryRow(get_number_of_requests_by_app_last_period, time.Now().Add(period)).Scan(&appName, &count)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return fmt.Sprintf("Application: %s, number of requests: %d", appName, count), nil
}

func GetTopErrorRequestsByAppInLastPeriod(period time.Duration, db *sql.DB) (TopErrorsReport, error) {

	var errorReports TopErrorsReport

	rows, err := db.Query(get_error_responses_last_period, time.Now().Add(period))

	if err != nil {
		if err == sql.ErrNoRows {
			return errorReports, nil
		}
		return errorReports, err
	}

	defer rows.Close()

	for rows.Next() {
		var errReport ReportErrorRequest
		//var jsonRequest []byte
		if err := rows.Scan(&errReport.LogId, &errReport.ApplicationName /* &jsonRequest, */, &errReport.Response); err != nil {
			return errorReports, err
		}
		//	err = json.Unmarshal(jsonRequest, &errReport.Request)
		//	if err != nil {
		//		return errorReports, err
		//	}
		errorReports.Request = append(errorReports.Request, errReport)
	}

	return errorReports, nil
}

func GetTopLongestRequestsInLastPeriod(period time.Duration, db *sql.DB) (TopDurationsReport, error) {
	var requestDurations TopDurationsReport

	rows, err := db.Query(get_longest_requests_last_period, time.Now().Add(period))

	if err != nil {
		if err == sql.ErrNoRows {
			return requestDurations, nil
		}
		return requestDurations, err
	}

	defer rows.Close()

	for rows.Next() {
		var reqDuration ReportRequestDuration
		//var jsonRequest []byte
		if err := rows.Scan(&reqDuration.LogId, &reqDuration.ApplicationName /* &jsonRequest,*/, &reqDuration.Duration); err != nil {
			return requestDurations, err
		}
		//err = json.Unmarshal(jsonRequest, &reqDuration.Request)
		//if err != nil {
		//	return requestDurations, err
		//}
		requestDurations.Requests = append(requestDurations.Requests, reqDuration)
	}

	return requestDurations, nil
}

func GetTopShortestsRequestsInLastPeriod(period time.Duration, db *sql.DB) (TopDurationsReport, error) {
	var requestDurations TopDurationsReport

	rows, err := db.Query(get_shortest_requests_last_period, time.Now().Add(period))

	if err != nil {
		return requestDurations, err
	}

	defer rows.Close()

	for rows.Next() {
		var reqDuration ReportRequestDuration
		//var jsonRequest []byte
		if err := rows.Scan(&reqDuration.LogId, &reqDuration.ApplicationName /*&jsonRequest, */, &reqDuration.Duration); err != nil {
			return requestDurations, err
		}
		//err = json.Unmarshal(jsonRequest, &reqDuration.Request)
		//if err != nil {
		//	return requestDurations, err
		//}
		requestDurations.Requests = append(requestDurations.Requests, reqDuration)
	}

	return requestDurations, nil
}

func GetTopTweetLocationsData(period time.Duration, db *sql.DB) (map[string]uint64, map[string]uint64, int, error) {
	locCounts := make(map[string]uint64)
	langCounts := make(map[string]uint64)
	totalPopulation := 0

	rows, err := db.Query(get_top_tweet_locations, time.Now().Add(period))

	if err != nil {
		return locCounts, langCounts, totalPopulation, err
	}

	defer rows.Close()

	for rows.Next() {
		var count uint64
		var location LocationInfo
		var jsonLangs []byte
		if err := rows.Scan(&location.Name, &count, &location.Population, &jsonLangs); err != nil {
			return locCounts, langCounts, totalPopulation, err
		}
		locCounts[location.Name] = count
		totalPopulation += int(location.Population)

		err = json.Unmarshal(jsonLangs, &location.Languages)
		if err != nil {
			return locCounts, langCounts, totalPopulation, err
		}

		for _, lang := range location.Languages {
			langCounts[lang.Name]++
		}
	}

	return locCounts, langCounts, totalPopulation, nil
}

func GetTopTweetRegionalBlocks(period time.Duration, db *sql.DB) ([]RegionalBlockCounts, error) {
	var blockCounts []RegionalBlockCounts

	rows, err := db.Query(get_tweets_regional_blocks_last_period, time.Now().Add(period))

	if err != nil {
		return blockCounts, err
	}

	defer rows.Close()

	counts := make(map[string]uint64)
	for rows.Next() {
		var regBlocks []com.RegionalBlocsType
		var jsonBlock []byte
		if err := rows.Scan(&jsonBlock); err != nil {
			return blockCounts, err
		}

		err = json.Unmarshal(jsonBlock, &regBlocks)
		if err != nil {
			return blockCounts, err
		}

		for _, regBlock := range regBlocks {
			counts[regBlock.Acronym]++
		}

	}

	for k, v := range counts {
		blockCounts = append(blockCounts, RegionalBlockCounts{RegionalBlock: k, NumberOfTweets: v})
	}
	sort.Slice(blockCounts, func(i, j int) bool {
		return blockCounts[i].NumberOfTweets > blockCounts[j].NumberOfTweets
	})

	if len(blockCounts) >= 10 {
		blockCounts = blockCounts[:10]
	}

	return blockCounts, nil
}

func SaveLogReport(period time.Duration, reportType string, db *sql.DB) error {
	mostRequests, err := GetNumberOfRequestsByAppInLastPeriod(period, db)
	if err != nil {
		return err
	}

	errorRequests, err := GetTopErrorRequestsByAppInLastPeriod(period, db)
	if err != nil {
		return err
	}

	jsonError, err := json.Marshal(errorRequests)
	if err != nil {
		return err
	}

	longestRequests, err := GetTopLongestRequestsInLastPeriod(period, db)
	if err != nil {
		return err
	}

	jsonLongest, err := json.Marshal(longestRequests)
	if err != nil {
		return err
	}

	shortestsRequests, err := GetTopShortestsRequestsInLastPeriod(period, db)
	if err != nil {
		return err
	}

	jsonShortest, err := json.Marshal(shortestsRequests)
	if err != nil {
		return err
	}

	//save to database
	_, err = db.Exec(insert_log_report, mostRequests, jsonError, jsonLongest, jsonShortest, reportType)
	return err
}

func SaveTweetReport(period time.Duration, reportType string, db *sql.DB) error {
	counts, err := GetTweetCountsInLastPeriod(period, db)
	if err != nil {
		return err
	}

	jsonCounts, err := json.Marshal(counts)
	if err != nil {
		return err
	}

	lengths, err := GetLargestTweetsInLastPeriod(period, db)
	if err != nil {
		return err
	}

	jsonLenghts, err := json.Marshal(lengths)
	if err != nil {
		return err
	}

	words, err := GetTopWordCountsInLastPeriod(period, db)

	jsonWords, err := json.Marshal(words)
	if err != nil {
		return err
	}

	//save to database
	_, err = db.Exec(insert_tweet_report, jsonCounts, jsonLenghts, jsonWords, reportType)

	return err
}

func SaveLocationReport(period time.Duration, reportType string, db *sql.DB) error {
	locCounts, langCounts, totalPopulation, err := GetTopTweetLocationsData(period, db)
	if err != nil {
		return err
	}

	jsonLocations, err := json.Marshal(locCounts)
	if err != nil {
		return err
	}

	jsonLanguages, err := json.Marshal(langCounts)
	if err != nil {
		return err
	}

	blockCounts, err := GetTopTweetRegionalBlocks(period, db)

	jsonBlocks, err := json.Marshal(blockCounts)
	if err != nil {
		return err
	}

	//save to database
	_, err = db.Exec(insert_location_report, jsonLocations, jsonBlocks, jsonLanguages, totalPopulation, reportType)

	return err
}

func SaveLocation(locationInfo LocationInfo, db *sql.DB) error {

	languagesJson, err := json.Marshal(locationInfo.Languages)
	if err != nil {
		return err
	}

	regBlocksJson, err := json.Marshal(locationInfo.RegionalBlocks)
	if err != nil {
		return err
	}

	_, err = db.Exec(insert_location, locationInfo.Name, languagesJson, regBlocksJson, locationInfo.Population)

	return err
}

func SaveLocationNameToUser(locationName LocationName, db *sql.DB) error {
	_, err := db.Exec(update_user_location_name, locationName.UserId, locationName.LocationName)
	return err
}
