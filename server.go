package zip_server

import (
	"net/http"
	"net/url"
	"time"
	"encoding/json"

	"fmt"
	"sync"
)

var config *Config

var shared struct {
	sync.Mutex
	openKeys map[string]bool
}

func init() {
	shared.openKeys = make(map[string]bool)
}

func keyBusy(key string) bool {
	shared.Lock()
	defer shared.Unlock()
	return shared.openKeys[key]
}

func lockKey(key string) {
	shared.Lock()
	defer shared.Unlock()
	shared.openKeys[key] = true
}

// release the key later to give the initial requester time to update the
// database
func releaseKeyLater(key string) {
	go func() {
		<-time.After(10 * time.Second)
		shared.Lock()
		defer shared.Unlock()

		shared.openKeys[key] = false
	}()
}

type errorHandler func(http.ResponseWriter, *http.Request) error

func (fn errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w,r); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// get the first value of param or error
func getParam(params url.Values, name string) (string, error) {
	vals := params[name]

	if len(vals) == 0 {
		return "", fmt.Errorf("Missing param %v", name)
	}

	val := vals[0]

	if len(val) == 0 {
		return "", fmt.Errorf("Missing param %v", name)
	}

	return val, nil
}

func zipHandler(w http.ResponseWriter, r *http.Request) error {
	params := r.URL.Query()
	key, err := getParam(params, "key")
	if err != nil {
		return err
	}

	if keyBusy(key) {
		msg, err := json.Marshal(struct{Error string}{"already processing"})
		if err != nil {
			return err
		}
		w.Header()["Content-Type"] = []string{"application/json"}
		w.Write(msg)
		return nil
	}

	lockKey(key)
	defer releaseKeyLater(key)

	// archiver := NewArchiver(config)
	// err := archiver.ExtractZip("test.zip", "zips/test")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	return nil
}

func StartZipServer(listenTo string, _config *Config) {
	config = _config
	http.Handle("/", errorHandler(zipHandler))
	fmt.Println("Listening on: " + listenTo)
	http.ListenAndServe(listenTo, nil)
}


