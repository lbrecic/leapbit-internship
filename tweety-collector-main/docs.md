# Tweety-Collector Documentation

version 1.10.0 - stable

package main - Package main initializes and run Tweety-Collector application and its methods.

## main.go

### func main();
Main function. Prints start message, initializes Tweety-Collector application and runs it.

## app.go

### type App struct;
Tweety-Collector application structure. Contains clients for communications with Twitter and other microservices.

### type Session struct;
Structure Session application internal queue, how many friends to search and how many friends to download.

### func appInit(\*Config) \*App;
Function initializes Tweety-Collector application based on loaded configuration parameters.

### func NewCollectorApp(\*Config, \*HTTPClientTwitter, \*HTTPClientCounter, \*HTTPClientDBSaver, \*com.TweetyLogger) \*App;
Tweety-Collector application constructor.

### func (\*App) start(string);
Method marks starting point of Tweety-Collector application. Once run, method can only be interrupted by internal error or SIGINT.

### func (\*App) session();
Method processes users and sends their data to Tweety-Counter and Tweety-DBSaver applications.

### func (\*App) process(string, bool) (string, error);
Method scrapes metadata and friends ids for given usrename/user id.

## config.go
 
### type Config struct;
Structure represents configuration data which is stored in config.json file

### func configurationLoader(\*Config) error;
Function loads configuration data into variable of type Config from config.json file.

## clients.go

### Type Cache struct;
Cache struct represents Tweety-Counter client internal cache for ids that failed to sent.

### type HTTPClientTwitter struct;
Twitter API client structure.

### type HTTPClientCounter struct;
Tweety-Counter client structure.

### type HTTPClientDBSaver struct;
Tweety-DBSaver client structure.

### func NewTwitterClient(string) HTTPClientTwitter;
Twitter client constructor.

### func NewCounterClient(string, string) HTTPClientCounter;
Tweety-Counter client constructor.

### func NewDBSaverClient(string, string) HTTPClientDBSaver;
Tweety-DBSaver client constructor.

### func (*HTTPClientCounter) clearCache();
Method clears Tweety-Counter clients cache memory.

## workers.go

### func (*App) apiIdsGetterWorker();
Twitter API v1.1 client worker method listens for user response on application built-in channel.

### func (\*HTTPClientCounter) counterIdsSenderWorker(); 
Tweety-Counter client worker method listens for user ids on clients built-in channel.

### func (\*HTTPClientDBSaver) databaseUserSenderWorker();
Tweety-DBSaver client worker method listens for users on clients built-in channel.

### func (\*HTTPClientDBSaver) databaseLocationSenderWorker();
Tweety-DBSaver client worker method listens for locations on clients built-in channel.

### func (\*HTTPClientCounter) counterIdsSender([]string) ([]string, error);
Method handles user ids data sending to Tweety-Counter server.

### func (\*HTTPClientDBSaver) databaseUserSender(com.ReqUser) error;
Method handles user data sending to Tweety-DBSaver server.

### func (\*HTTPClientDBSaver) databaseLocationSender(com.RespLocation, string) error;
Method handles location data sending to Tweety-DBSaver server.

### func (\*HTTPClientDBSaver) userExists(string) (*http.Response, com.RespUserExists, error);
Method for handling response from database while checking if user exists.

## location.go

### func init();
Function init starts at build; Creates array of endpoints for REST Countries API v2.

### func (\*HTTPClientDBSaver) location(loc string) (com.RespLocation, \*http.Response, error);
Method tries to collect location data for given input at any of endpoints for REST Countries API v2.

### func (\*HTTPClientDBSaver) requestSending(string, string) (com.RespLocation, \*http.Response, error);
Method forms location data request for given endpoint and location name and sends it to getLocationInfo method.

### func (\*HTTPClientDBSaver) getLocationInfo(string) ([]com.RespLocation, \*http.Response, error);
Method for request processing and scraping location data from REST Countries API v2.

### func splitLocation(string) []string;
Method parses locations that are separated by comma or whitespace.

## phases.go

### (app \*App) func startMessage();
Function prints out start message to user.

### func (\*App) shutdownAwait();
Secure clients shutdown method.

### func (app \*App) shutdownMessage();
Function prints out shutdown message to user.

## tools.go

### type UserLocationPair struct;
Structure UserLocationPair holds location name for given user id.

### func createUser(tw.RespTwitterApiUser, []string) com.ReqUser;
Function for creating User struct variable.

### func createUserLocationPair(string, string) UserLocationPair;
Function for creating UserLocationPair struct variable.

## metrics.go

### type Metric struct;
Metric structure contains all required counters for data representation.

### func NewMetric() Metric;
Collector metrics constructor.

### func startMetrics();
Functions starts server that handles metrics in real time using prometheus package.
