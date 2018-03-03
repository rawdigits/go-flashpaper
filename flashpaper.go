//Ryan Huber rhuber@gmail.com - 2016

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const MAXUPLOADSIZE = 104857600 //10mb
const MAXHOURSTOKEEP = 24.0     //colloquially: "a day"
const SECRETSTOREKEY = 234      //if you don't know what this is, don't worry about it

//Because typing this over and over is silly
type smap map[string]*secret

//Secrets are this
type secret struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Data []byte `json:"data"`
	time time.Time
	Name string `json:"name"`
}

//Clear the actual bytes in memory .. hopefully.
func (s *secret) Wipe() {
	for i, _ := range s.Data {
		s.Data[i] = 0
	}
}

func NewSecret() *secret {
	id, err := randPathString()
	if err != nil {
		log.Fatal(err)
	}
	return &secret{Id: id, time: time.Now()}
}

func secretHandler(w http.ResponseWriter, r *http.Request) {

	secrets := r.Context().Value(SECRETSTOREKEY).(smap)

	path := r.URL.Path[1:]
	r.ParseForm()

	isShare, _ := regexp.MatchString("^share/", path)
	isRoot, _ := regexp.MatchString("^$", path)
	isAdd, _ := regexp.MatchString("^add$", path)
	isAddfile, _ := regexp.MatchString("^addfile$", path)
	isFavicon, _ := regexp.MatchString("^favicon.ico$", path)

	//prevent slackbot from exploding links when posted to a channel
	if strings.Contains(r.UserAgent(), "Slack") {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case "GET":
		switch {
		case isShare:
			secretID := path[6:]
			sec, ok := popSecret(secrets, secretID)
			if ok {
				log.Print(fmt.Sprintf("%s ->> %s\n", secretID, r.Header))
				defer sec.Wipe()
				//If this is a file, set to octet-stream to force download
				//Otherwise just print the data
				if sec.Type == "file" {
					w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", sec.Name))
					w.Header().Set("Content-Type", "application/octet-stream")
					w.Write(sec.Data)
				} else {
					fmt.Fprintf(w, "%s", sec.Data)
				}
			} else {
				http.NotFound(w, r)
				fmt.Fprintf(w, "The requested paper has expired or viewed by someone else.")
			}
		case isFavicon:
			return
		case isRoot:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, lackofstyle+index+endofstyle)
		case isAdd:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, lackofstyle+inputtextform+endofstyle)
		case isAddfile:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, lackofstyle+inputfileform+endofstyle)
		default:
			{
				http.NotFound(w, r)
				fmt.Fprintf(w, "The requested paper has expired or viewed by someone else.")
			}
		}

	case "POST":
		//I could lock on adding things to the map, but i'm not gonna.
		//If randomString() collides, sorry...?
		switch {
		case isAdd:
			for k, v := range r.Form {
				if k == "secret" {
					v := strings.Join(v, "")
					secret := NewSecret()
					secrets[secret.Id] = secret
					secrets[secret.Id].Data = []byte(v)

					shareable(secret.Id, w, r)
				} else {
					fmt.Fprintf(w, "no secret provided.")
				}
			}
		case isAddfile:
			f, h, _ := r.FormFile("file")
			defer f.Close()
			d := new(bytes.Buffer)

			//Limit the size of uploads. We aren't made of money.
			mb := http.MaxBytesReader(w, f, MAXUPLOADSIZE)
			_, err := io.Copy(d, mb)
			if err != nil {
				fmt.Fprintf(w, lackofstyle+uploaderror+endofstyle)
				return
			}
			secret := NewSecret()
			secrets[secret.Id] = secret
			secrets[secret.Id].Type = "file"
			secrets[secret.Id].Data = d.Bytes()
			secrets[secret.Id].Name = h.Filename

			shareable(secret.Id, w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func shareable(id string, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	proto := "https"
	if r.TLS == nil {
		proto = "http"
	}
	ret := fmt.Sprintf(lackofstyle+shareform+endofstyle, proto, r.Host, id, MAXHOURSTOKEEP)
	fmt.Fprintf(w, ret)
}

//This generates a (crypto) random 32 byte string for the path
func randPathString() (string, error) {
	rb := make([]byte, 32)
	_, err := rand.Read(rb)
	s := fmt.Sprintf("%x", rb)
	if err == nil {
		return s, nil
	}
	return "", errors.New("could not generate random string")
}

func popSecret(secrets smap, sec string) (secret, bool) {
	//It is important to use the mutex here, otherwise a race condition
	//could lead to the ability to read the secret twice.
	//This would defeat the whole purpose of flashpaper...
	mu.Lock()
	defer mu.Unlock()
	val, ok := secrets[sec]
	if ok {
		delete(secrets, sec)
	} else {
		return secret{}, false
	}
	return *val, ok
}

//Runs every 1 second(s) to remove things that haven't been read and are expired
func janitor(secrets smap) {
	for {
		for k, v := range secrets {
			duration := time.Since(v.time)
			if duration.Hours() > MAXHOURSTOKEEP {
				sec, _ := popSecret(secrets, k)
				sec.Wipe()
			}
		}
		//Sleep one second
		time.Sleep(1000000000)
	}
}

func contextify(fn func(w http.ResponseWriter, r *http.Request), secrets smap) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := context.WithValue(r.Context(), SECRETSTOREKEY, secrets)
		fn(w, r.WithContext(c))
	}
}

//ya ya ya, globals are bad, but this is a lock.
var mu = &sync.Mutex{}

func main() {

	//set up the map that stores secrets
	secrets := smap{}

	//launch the janitor to remove secrets that haven't been retrieved
	go janitor(secrets)

	//this handles all http requests. secretHandler uses a big stupid case statement.
	http.HandleFunc("/", contextify(secretHandler, secrets))

	//You can uncomment the non TLS version of ListenAndServe and
	//run this without TLS if you have taken leave of your senses.
	//err := http.ListenAndServe(":8080", nil)
	err := http.ListenAndServeTLS(":8443", "server.crt", "server.key", nil)
	if err != nil {
		fmt.Printf("main(): %s\n", err)
		fmt.Printf("Errors usually mean you don't have the required server.crt or server.key files.\n")
	}
}

//That's right friend, all the terrible HTML is right here in the source.
const lackofstyle = `
<html><head></head>
<style>
* { font-family: "Raleway", "HelveticaNeue", "Helvetica Neue", Helvetica, Arial, sans-serif; }
</style>
<body><br>
<div style="text-align:center; top: 25px;">
`
const endofstyle = `
</div></body></html>
`

const index = `
<a href=/add>Share a TEXT secret.</add><br><br>
<a href=/addfile>Share a secret FILE.</add><br>
`

const inputtextform = `
<form action="/add" method="POST">
<textarea name="secret" rows="20" cols="80"></textarea>
<br>
<input type=submit>
</form>
`

const inputfileform = `
<form action="/addfile" method="POST" enctype="multipart/form-data">
<!-- <label for="file">Filename: </label><br> -->
<input type="file" name="file" id="file">
<input type=submit>
</form>
`

const shareform = `
share this link (do not click!):<br><br>
<h2>%s://%s/share/%s</h2>
<br><br>THIS LINK WILL EXPIRE IN %f HOURS<br><br>
<a href="/">Share Another Secret</a>
`

const uploaderror = `
<h2>Upload too large.</h2>
<a href="/">Share Another Secret</a>
`
