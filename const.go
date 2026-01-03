package main

const WEBHOOKS_UPGRADE_EVENT = "user.upgraded"

const CONTENT_TYPE_PLAIN_TEXT = "text/plain; charset=utf-8"
const CONTENT_TYPE_HTML = "text/html"
const CONTENT_TYPE_JSON = "application/json"

const METRICS_HTML = `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

const VALID_CHIRP_LENGTH = 140

var PROFANE_WORDS = map[string]bool{
	"kerfuffle": true,
	"sharbert":  true,
	"fornax":    true,
}
