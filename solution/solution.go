/**
 * Go Challenge - Travel Audience
 *
 * Implementation of an API service by Yorman (@cixtor) for a code challenge
 * designed by Travel Audience to recruit Go Developers. The following para-
 * graphs contain a list of random thoughts about the specifications of the
 * problem as well as things that I considered before implementing the code.
 *
 * Check if the custom port number is usable, eg. 127.0.0.1:80
 *
 * If the user accesses the root page, display usage information.
 *
 * Build a custom HTTP request instead of the basic http.Get, this will gives
 * me the ability to control which headers will be sent to the external URLs,
 * including the "Accept" header which technically speaking should expect a
 * "application/json" response.
 *
 * Using the parameter `u=FOOBAR` to gather the URLs is ambiguous because in
 * most production ready solutions like Apache, Nginx, etc the duplication of
 * parameters forces the server to take the value of the latest entry, this is,
 * if someone sends `u=FOO&u=BAR` the server will only read `BAR` and discard
 * `FOO` because the parameter has the same name. To support multiple entries
 * per parameter the user must send the request like `u[]=FOO&u[]=BAR`. In the
 * case of Go, if someone sends a request with multiple values for the same
 * parameter without defining the URL as an array then the value associated to
 * the first position of the entry list will be returned using this piece of
 * code `(*http.Request).URL.Query().Get("param")`.
 *
 * I don't want to modify the specifications of the challenge so I will just
 * read from `(*http.Request).URL.Query()["u"]` which returns a `[]string`.
 *
 * For an Internet connection like mine, having a hard limit of 500ms makes
 * things difficult, hopefully the _"testserver.go"_ was provided for testing
 * purposes. In an ideal scenario I would divide the maximum number of milli-
 * seconds among the number of URLs submitted by the user, then use the result
 * as the timeout; for example, if the user sends ten URLs we would have
 * `500ms / 10` which gives 50ms as the timeout per URL. However, since this
 * is real-life implementing this would end up in a waste of time, what if
 * each URL takes 51ms to respond? All of them would fail. A better approach
 * would be to use the maximum number of milliseconds for the first request,
 * if it takes 50ms then we subtract from the pool and use the result as the
 * timeout for the second request, which in this case would be 450ms, and so
 * on until the maximum amount of time is exceeded no matter how many URLs
 * are contacted, this way I guarantee that at least the first URL will not
 * be limited by an irrational amount of time.
 *
 * The deletion of duplicated numbers from the merge can be accomplished using
 * brute-force by creating a second empty slice, then iterating through the
 * entire list and sending the numbers to the slice if they do not exist there.
 * However, I think there is a better way to do this, I can sort the list and
 * then iterate, for each position I will check if the number is equal to the
 * number in the previous position, if yes then I skip, otherwise I copy the
 * number to the second slice. This seems like the same solution but in this
 * case I am skipping the _"is-number-in-2nd-slice?"_ check.
 *
 * CloudFlare - Go net/http Timeouts [1] blog post has useful information that
 * helps to determine where to define the request timeout depending on the
 * objective. In the _"Client Timeouts"_ section it is explained that the
 * amount of time that takes to build the HTTP headers and write the response
 * can be controlled via `http.Client.Timeout`.
 *
 * It is not clear if the 500ms hard limit is for the execution of the code
 * that calls the API endpoints or the entire application. If the API services
 * return lists with too many numbers the sorting and merge actions will take
 * more time than expected, but considering the impossibility to determine how
 * much data will be returned by the API endpoints I can just hope that the
 * limit for the execution time is only for the iterator that calls the URLs,
 * this is, using `time curl` the _"real"_ time will always be bigger than
 * 500ms because the program has to process and transform the data, and it is
 * difficult to predict how much time it will take to do that.
 *
 * [1] https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
 */

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"
)

// MaxTimeout defines the maximum number of milliseconds that the whole program
// execution should take with a standard deviation of ~0.20 secs. This is, if
// the program will run against ten different API endpoints the program should
// not take more than ~500ms to collect all the numbers provided by these
// services, timeouts will be fired for every request that takes more than this
// definition.
const MaxTimeout = 500

// DefaultPort defines the port number to run the local server.
const DefaultPort = "8686"

// Result holds the merged list of numbers.
type Result struct {
	Numbers []int `json:"numbers"`
}

// ValidAPIEndpoints a list of parameters sent by the client via a GET request.
// The function assumes that the user will send a list of strings via a GET
// parameter named "u" which is not defined as an array, because of this the
// function will need to take the raw information from the URL to extract all
// the entries. Notice that some of the strings are not valid URI, the function
// will try to parse the string, if the scheme is using the HTTP or HTTPS
// protocols then it will consider it valid.
func ValidAPIEndpoints(params []string) []string {
	var endpoints []string

	for _, param := range params {
		u, err := url.Parse(param)

		if err != nil {
			log.Printf("Invalid URL: %s; %s", param, err)
			continue
		}

		if u.Scheme == "http" || u.Scheme == "https" {
			endpoints = append(endpoints, u.Scheme+"://"+u.Host+u.Path)
		}
	}

	return endpoints
}

/**
 * Request APIs, collect numbers, merge, respond.
 *
 * We have a hardlimit of N milliseconds to execute this operation, with M
 * number of URLs which may or may not be valid or responsive. For each possible
 * API endpoint we will send a HTTP GET request with a timeout of (N-P) where P
 * is the amount of time it took to execute a previous request, for example, if
 * N=500ms and M contains ten valid URLs the function will send the first
 * request with a 500ms timeout, suppose the API responds in 150ms, we will
 * subtract this from N so the second request will timeout after 350ms per this
 * operation (500ms - 150ms = 350ms). If N is lower than zero and there are more
 * URLs in the list then the function will timeout all of them to keep the
 * execution time on limit.
 *
 * @param  []string endpoints List of valid API endpoints
 * @return []int              All collected numbers from the API
 */
func collectAllNumbers(endpoints []string) []int {
	var result Result
	var numbers []int
	var start time.Time
	var elapse time.Duration
	var maximum time.Duration = (MaxTimeout * 1000000)
	var wg sync.WaitGroup

	wg.Add(len(endpoints))

	for _, url := range endpoints {
		go func(wg *sync.WaitGroup, url string) {
			defer wg.Done()

			result = Result{}
			start = time.Now()

			client := &http.Client{Timeout: maximum}
			req, err := http.NewRequest("GET", url, nil)

			req.Header.Set("Accept", "application/json")
			req.Header.Set("User-Agent", "Mozilla/5.0 (KHTML, like Gecko) Safari/537.36")

			if err != nil {
				log.Printf("NewRequest; %s\n", err)
				return
			}

			resp, err := client.Do(req)

			if err != nil {
				elapse = time.Since(start)
				log.Printf("TIMEOUT (%s); %s\n", elapse, err)
				return
			}

			defer resp.Body.Close()

			json.NewDecoder(resp.Body).Decode(&result)

			elapse = time.Since(start)
			log.Printf("RESPONSE (%s) %#v\n", elapse, result.Numbers)

			if result.Numbers != nil {
				numbers = append(numbers, result.Numbers...)
			}
		}(&wg, url)
	}

	wg.Wait()

	return numbers
}

/**
 * Returns a list of unique integers.
 *
 * For a list of integers where a number appears more than once, the function
 * will check if the list if not empty, then sort the list and select the first
 * number. If there are more entries in the list the function will iterate
 * through them and check if the number in position K is equal to the number in
 * position K-1, if this is true then the iterator will skip to the next
 * position and execute the same comparison, if the numbers are different then
 * LIST[K] will be pushed to the end of a second list where all the unique
 * numbers will be collected. This algorithm saves the operation known as _"Is
 * Item in Array"_ which usually take smore time when the list is too big.
 *
 * @param  []int numbers List of unordered numbers
 * @return []int         List of ordered unique numbers
 */
func simpleUniqueNumbers(numbers []int) []int {
	var unique []int

	total := len(numbers)

	if total > 0 {
		sort.Ints(numbers)
		unique = append(unique, numbers[0])

		if total >= 1 {
			for key := 1; key < total; key++ {
				if numbers[key] != numbers[key-1] {
					unique = append(unique, numbers[key])
				}
			}
		}
	}

	return unique
}

/**
 * Process an HTTP request to /numbers
 *
 * The function will validate the strings passed by the client via the URL using
 * a parameter named "u", for every string representing a valid URL the function
 * will send a HTTP GET request and collect a JSON-encoded object which is
 * expected to contain a property named "numbers" which is an array of integers.
 * Then it will merge all these numbers, delete the duplicated entries, and
 * return a JSON-encoded object with the new list of unique integers.
 *
 * @param  http.ResponseWriter w HTTP response writer
 * @param  *http.Request       r HTTP request interface
 * @return void
 */
func solution(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()["u"]
	endpoints := ValidAPIEndpoints(params)
	numbers := collectAllNumbers(endpoints)
	unique := simpleUniqueNumbers(numbers)

	log.Printf("MERGE: %#v\n", numbers)
	log.Printf("UNIQUE: %#v\n", unique)

	json.NewEncoder(w).Encode(Result{Numbers: unique})
}

/**
 * Display instructions of how to use the API.
 *
 * @param  http.ResponseWriter w HTTP response writer
 * @param  *http.Request       r HTTP request interface
 * @return void
 */
func homepage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Send a GET request to /numbers\n"))
}

/**
 * Program entry point.
 *
 * @return void
 */
func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = DefaultPort
	}

	log.Printf("Running server on http://127.0.0.1:%s", port)

	http.HandleFunc("/numbers", solution)

	/* Put the root at the end to prevent conflicts */
	http.HandleFunc("/", homepage)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Cannot use port; %s", err)
	}
}
