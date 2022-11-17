package comms

import (
	"time"

	tw "gitlab.com/leapbit-practice/tweety-lib-twitter/twitter"
)

type ReqUser struct {
	Id              uint64            `json:"id"`
	Id_str          string            `json:"id_str"`
	Name            string            `json:"name"`
	Screen_name     string            `json:"screen_name"`
	Location        string            `json:"location"`
	URL             string            `json:"url"`
	Description     string            `json:"description"`
	Protected       bool              `json:"protected"`
	Verified        bool              `json:"verified"`
	Followers_count uint64            `json:"followers_count"`
	Friends_count   uint64            `json:"friends_count"`
	Statuses_count  uint64            `json:"statuses_count"`
	Created_at      time.Time         `json:"created_at"`
	Followers_id    []string          `json:"list_of_follower_ids"`
	Ten_words       map[string]uint64 `json:"word_counts"`
	Last_modified   time.Time         `json:"last_modified"`
	App_name        string            `json:"app_name"`
	Sent_at         time.Time         `json:"timestamp"`
}

type ReqUserId struct {
	UserId  string    `json:"user_id"`
	AppName string    `json:"app_name"`
	SentAt  time.Time `json:"timestamp"`
}

type RespUserExists struct {
	Exists        bool      `json:"exists"`
	Last_modified time.Time `json:"last_modified"`
}

type RespTweetId struct {
	Id string `json:"tweet_id"`
}

type ReqTweetsForDB struct {
	UserId    string                   `json:"user_id"`
	Tweets    []tw.RespTwitterApiTweet `json:"tweets"`
	WordCount []KvPair                 `json:"word_count"`
	AppName   string                   `json:"app_name"`
	SentAt    time.Time                `json:"timestamp"`
}

type ReqImagesForDB struct {
	UserId     string    `json:"user_id"`
	UserImages []byte    `json:"user_images"`
	AppName    string    `json:"app_name"`
	SentAt     time.Time `json:"timestamp"`
}

type KvPair struct {
	Word  string
	Count uint64
}

type ReqLocationForDB struct {
	LocationInfo RespLocation `json:"location_info"`
	UserId       string       `json:"user_id"`
	AppName      string       `json:"app_name"`
	SentAt       time.Time    `json:"sent_at"`
}

type RespLocation struct {
	Name           string              `json:"name"`
	TopLevelDomain []string            `json:"topLevelDomain"`
	Alpha2Code     string              `json:"alpha2Code"`
	Alpha3Code     string              `json:"alpha3Code"`
	CallingCodes   []string            `json:"callingCodes"`
	Capital        string              `json:"capital"`
	AltSpellings   []string            `json:"altSpellings"`
	Region         string              `json:"region"`
	Subregion      string              `json:"subregion"`
	Population     int64               `json:"population"`
	Latlng         []float64           `json:"latlng"`
	Demonym        string              `json:"demonym"`
	Area           float64             `json:"area"`
	Gini           float64             `json:"gini"`
	Timezones      []string            `json:"timezones"`
	Borders        []string            `json:"borders"`
	NativeName     string              `json:"nativeName"`
	NumericCode    string              `json:"numericCode"`
	Currencies     []CurrenciesType    `json:"currencies"`
	Languages      []LanguagesType     `json:"languages"`
	Translations   map[string]string   `json:"translations"`
	Flag           string              `json:"flag"`
	RegionalBlocs  []RegionalBlocsType `json:"regionalBlocs"`
	Cioc           string              `json:"cioc"`
}

type CurrenciesType struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type LanguagesType struct {
	ISO639_1   string `json:"iso639_1"`
	ISO639_2   string `json:"iso639_2"`
	Name       string `json:"name"`
	NativeName string `json:"nativeName"`
}

type RegionalBlocsType struct {
	Acronym       string   `json:"acronym"`
	Name          string   `json:"name"`
	OtherAcronyms []string `json:"otherAcronyms"`
	OtherNames    []string `json:"otherNames"`
}
