//Ryan Huber rhuber@gmail.com - 2016

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// MAXUPLOADSIZE of 10mb
const MAXUPLOADSIZE = 104857600

// MAXHOURSTOKEEP colloquially: "a day"
const MAXHOURSTOKEEP = 24.0

// SECRETSTOREKEY if you don't know what this is, don't worry about it
const SECRETSTOREKEY = 234

// DEFAULTCREDS a Hash of user1 pass1.
const DEFAULTCREDS = "0a041b9462caa4a31bac3567e0b6e6fd9100787db2ab433d96f6d178cabfce90" +
	" e6c3da5b206634d7f3f3586d747ffdb36b5c675757b380c6a5fe5c570c714349"

// CANARYTOKEN The provided canarytoken.
var CANARYTOKEN = ""

// AUTHFILENAME the provided file name.
var AUTHFILENAME = "auth.txt"

//AUTH where auth is enabled or not.
var AUTH = false

//Because typing this over and over is silly
type smap map[string]*secret

//Secrets are this
type secret struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data []byte `json:"data"`
	time time.Time
	Name string `json:"name"`
	Auth bool   `json:"auth"`
}

//Clear the actual bytes in memory .. hopefully.
func (s *secret) Wipe() {
	for i := range s.Data {
		s.Data[i] = 0
	}
}

// NewSecret Generate a new secret.
func NewSecret() *secret {
	id, err := randPathString()
	if err != nil {
		log.Fatal(err)
	}
	return &secret{ID: id, time: time.Now()}
}

func secretHandler(w http.ResponseWriter, r *http.Request) {

	secrets := r.Context().Value(SECRETSTOREKEY).(smap)

	path := r.URL.Path[1:]
	r.ParseForm()

	//prevent slackbot from exploding links when posted to a channel
	if strings.Contains(r.UserAgent(), "Slack") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case "GET":
		switch path {
		case "favicon.ico":
			return
		case "":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, lackofstyle+index+endofstyle)
		case "add":
			w.Header().Set("Content-Type", "text/html")

			var ret = ""
			if AUTH {
				ret = fmt.Sprintf(lackofstyle+inputtextform+endofstyle, checkBox)
			} else {
				ret = fmt.Sprintf(lackofstyle+inputtextform+endofstyle, "")
			}
			fmt.Fprintf(w, ret)
		case "addfile":
			w.Header().Set("Content-Type", "text/html")

			var ret = ""
			if AUTH {
				ret = fmt.Sprintf(lackofstyle+inputfileform+endofstyle, checkBox)
			} else {
				ret = fmt.Sprintf(lackofstyle+inputfileform+endofstyle, "")
			}
			fmt.Fprintf(w, ret)
		case "help":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, lackofstyle+help+endofstyle)
		case "usedlink":
			w.Header().Set("Content-Type", "text/html")
			ret := fmt.Sprintf(lackofstyle+display+endofstyle, "This link has already been used.")
			fmt.Fprintf(w, ret)
		default:
			// Access the secret without popping it.
			val, ok := secrets[path]

			if ok {
				// First check whether we need to perform Auth.
				if val.Auth {
					user, pass, ok := r.BasicAuth()
					// Check if the provided username and password that have been hashed, match the provided ones.
					if !ok || !checkFileAuth(hasher(user), hasher(pass)) {
						realm := "Please enter a username and password."
						w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
						http.Error(w, "Unauthorized.", http.StatusUnauthorized)
						return
					}
				}
				//If this is a file, set to octet-stream to force download
				//Otherwise just print the data
				if val.Type == "file" {
					sec, _ := popSecret(secrets, path)

					defer sec.Wipe()

					w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", sec.Name))
					w.Header().Set("Content-Type", "application/octet-stream")
					w.Write(sec.Data)
				} else {
					w.Header().Set("Content-Type", "text/html")

					sec, _ := popSecret(secrets, path)

					defer sec.Wipe()

					ret := fmt.Sprintf(lackofstyle+display+endofstyle, html.EscapeString(string(sec.Data)))
					fmt.Fprintf(w, ret)
				}
			} else {
				// When setting up the token on a live server, please generate a new token that redirects to /usedlink.
				if CANARYTOKEN != "" {
					http.Redirect(w, r, CANARYTOKEN+"?l="+path, http.StatusSeeOther)
				} else {
					w.Header().Set("Content-Type", "text/html")
					ret := fmt.Sprintf(lackofstyle+display+endofstyle, "This link has already been used.")
					fmt.Fprintf(w, ret)
				}
			}
		}
	case "POST":
		//I could lock on adding things to the map, but i'm not gonna.
		//If randomString() collides, sorry...?
		switch path {
		case "add":

			// Extract the parameters from the POST.
			_, existsAuth := r.Form["auth"]
			secretVal, existsSecret := r.Form["secret"]

			if existsSecret {
				secretVal := strings.Join(secretVal, "")
				secret := NewSecret()
				secrets[secret.ID] = secret
				secrets[secret.ID].Data = []byte(secretVal)

				if existsAuth {
					// We don't want them to Specify auth if the server can't check for it.
					if AUTH {
						secrets[secret.ID].Auth = true
					}
				}
				shareable(secret.ID, w, r)
			} else {
				fmt.Fprintf(w, "no secret provided.")
			}

		case "addfile":
			f, h, _ := r.FormFile("file")
			_, existsAuth := r.Form["auth"]

			defer f.Close()
			d := new(bytes.Buffer)

			//Limit the size of uploads. We aren't made of money.
			mb := http.MaxBytesReader(w, f, MAXUPLOADSIZE)
			_, err := io.Copy(d, mb)
			if err != nil {
				return
			}

			secret := NewSecret()
			secrets[secret.ID] = secret
			secrets[secret.ID].Type = "file"
			secrets[secret.ID].Data = d.Bytes()
			secrets[secret.ID].Name = h.Filename

			if existsAuth {
				// We don't want them to Specify auth if the server can't check for it.
				if AUTH {
					secrets[secret.ID].Auth = true
				}
			}

			shareable(secret.ID, w, r)
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

	ret := fmt.Sprintf(lackofstyle+shareform+endofstyle, int(MAXHOURSTOKEEP), proto, r.Host, id)
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

// Hasher returns a buffer of bytes. This is not very useful when comparing them hashes in the text file.
// After hasher has returned, the buffer is converted to hex so as to make it easier to compare.
func hasher(s string) []byte {
	val := sha256.Sum256([]byte(s))
	var hex []string

	// Iterate through the bytes.
	for i := 0; i < len(val); i++ {
		// We want each number to be represented by 2 chars.
		placeHolder := []string{"0"}
		value := strconv.FormatInt(int64(val[i]), 16)

		if len(value) != 2 {
			placeHolder = append(placeHolder, value)
			hex = append(hex, strings.Join(placeHolder, ""))
		} else {
			hex = append(hex, value)
		}
	}
	return []byte(strings.Join(hex, ""))

}

// This function will verify that the provided username and password are in fact in the text file.
func checkFileAuth(user []byte, pass []byte) bool {

	fi, err := os.Open(AUTHFILENAME)

	// Something has gone wrong whilst opening the file.
	if err != nil {
		fmt.Println("The Specified auth file does not exist.")
		// Create a new auth,txt file.
		f, err2 := os.OpenFile(AUTHFILENAME, os.O_CREATE|os.O_RDWR, 0644)
		if err2 != nil {
			panic(err2)
		}

		// Ensure that the file will be closed.
		defer func() {
			if err2 := f.Close(); err2 != nil {
				panic(err2)
			}
		}()

		// Give it the default credentials.
		_, err2 = f.Write([]byte(DEFAULTCREDS))

		if err2 != nil {
			panic(err2)
		}

		fmt.Println("A file was created for you with a set of default credentials.")
		return false

	}

	// Ensure that the file will be closed.
	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()

	data := make([]byte, 130)

	// Iterate through all entries in the file.
	for {
		count, err := fi.Read(data)
		// Read until we reach the EOF.
		if err != nil && err != io.EOF {
			panic(err)
		}

		if count == 0 {
			break
		}

		// Check whether we have the username/password combination.
		if subtle.ConstantTimeCompare(data[0:64], user) == 1 && subtle.ConstantTimeCompare(data[65:129], pass) == 1 {
			return true
		}
	}

	return false
}

func authHandler(handler http.HandlerFunc, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Don't authenticate for links.
		if len(r.URL.Path) > 10 || r.URL.Path == "/usedlink" {
			handler(w, r)
		} else {
			user, pass, ok := r.BasicAuth()
			// Check if the provided username and password that have been hashed, match the provided ones.
			if !ok || !checkFileAuth(hasher(user), hasher(pass)) {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized.", http.StatusUnauthorized)
				return
			}
			handler(w, r)
		}

	}
}

//ya ya ya, globals are bad, but this is a lock.
var mu = &sync.Mutex{}

func main() {

	fmt.Println(ascii)

	realm := "Please enter a username and password."

	//set up the map that stores secrets
	secrets := smap{}

	//set up cert manager for ssl certs
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(""), //Your domain here
		Cache:      autocert.DirCache("certs"),                             //Folder for storing certificates
	}

	//launch the janitor to remove secrets that haven't been retrieved
	go janitor(secrets)

	// Setup the flags.
	// value = default when not specified.
	var auth = flag.String("auth", "", "The auth filename.")
	var token = flag.String("token", "", "The token link.")
	var autocert = flag.Bool("autocert", false, "Whether to use Autocert or not.")
	flag.Parse()

	// Check whether a token was specified. If not, it will get an empty string which is fine.
	CANARYTOKEN = *token

	if *token == "" {
		fmt.Println("Token: False.")
	} else {
		fmt.Println("Token: True.")
	}

	// Auth needs to be used.
	if *auth != "" {
		AUTH = true
		AUTHFILENAME = *auth
		fmt.Println("Authentication: Enabled.")
		http.HandleFunc("/", authHandler(contextify(secretHandler, secrets), realm))

	} else {
		fmt.Println("Authentication: Disabled.")
		http.HandleFunc("/", contextify(secretHandler, secrets))

	}

	if *autocert {
		fmt.Println("AutoCert: True.")
		server := &http.Server{
			Addr: ":8443",
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}

		go http.ListenAndServe(":http", certManager.HTTPHandler(nil))
		err := server.ListenAndServeTLS("", "") //Key and cert are coming from Let's Encrypt
		if err != nil {
			fmt.Printf("main(): %s\n", err)
		}

	} else {
		fmt.Println("AutoCert: False.")
		//Key and cert are coming from Let's Encrypt.
		err := http.ListenAndServeTLS(":8443", "server.crt", "server.key", nil)
		if err != nil {
			fmt.Printf("main(): %s\n", err)
			fmt.Printf("Errors usually mean you don't have the required server.crt or server.key files.\n")
		}
	}

}

//That's right friend, all the terrible HTML is right here in the source.
const lackofstyle = `
<html><head><title>FlashPaper</title></head>
<style>
* {  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol"; }
body {
	background-color: rgb(237, 244, 239)
}

.buttonLarge {
	background-color: #4CAF50;
	color: white;
	padding: 12px 12px;
	text-align: center;
	text-decoration: none;
	display: inline-block;
	font-size: 14px;
	border-radius:4px;
    word-wrap: break-word;
	cursor:pointer;
}

.buttonLarge:hover {
	box-shadow: 0 3px 10px 0 rgba(0,0,0,0.24),0 3px 10px 0 rgba(0,0,0,0.19);
}

.buttonSmall {
	color:#4CAF50;
	background-color: white;
	border: none;
	text-align: center;
	text-decoration: none;
	font-size: 12px;
	cursor:pointer;
	word-wrap: break-word;
}

.box2 {
    word-wrap: break-word;
    font-family: "menlo";
	cursor:pointer;
}


.container {
	display:grid;
    grid-template-columns: repeat(21, 0.2fr);
    grid-template-rows: repeat(15, 0.2fr);
    justify-content:space-evenly;
	height:100vh;
	text-align:center;
}

.box {
	border-width:thin;
    border-radius:8px;
	background-color:white;
	padding: 16px;
	box-shadow: 0px 3px 14px 3px rgba(0, 0, 0, 0.1);
}

.checkBox {
	color:gray;
	margin:12px;
}

</style>
<body><br>
`
const endofstyle = `
</body></html>
`
const display = `

<script>
function copy() {
	var text = document.getElementById("link").innerText;
	var elem = document.createElement("textarea");
	document.body.appendChild(elem);
	elem.value = text;
	elem.select();
	document.execCommand("copy");
	document.body.removeChild(elem);
}

function finish() {
	document.getElementById("done").style.visibility = 'hidden';
	document.getElementById("copy").style.visibility = 'hidden';
	document.getElementById("heading").style.visibility = 'hidden';
	document.getElementById("link").innerHTML = "Your data has been deleted. You can close the tab.";

}

</script>

<style>

.boxSpec {
    grid-row-start:5;
    grid-column:9/14;

}

.header {
	grid-row-start:4;
    grid-column-start:11;
}


.buttonLargeSpec {
	display: block;
	width:100px;
	margin:20px auto;

}


</style>

<div class="container">
	<div class="header">
		<h1 style="color:dimgray"> <a style="text-decoration: none; color:dimgray;" href=/>Flashpape<span style="color:#4caf50">r</span> </a></h1>
	</div>
	<div class="box boxSpec">
		<h5 id="heading" style="color:gray;"> Here is your data: </h5>
        <div class="box2">
  			<p style="white-space:pre-line;" id="link">%s</p>
        </div>
		<button id="copy" class="buttonSmall" onclick="copy()">Copy</button>
        <a id="done" onclick="finish()" class="buttonLarge buttonLargeSpec">Done</a>
     
    </div>
</div>

`
const help = `

<style>

.boxSpec {
    grid-row-start:4;
    grid-column:7/16;

}

.header {
	grid-row-start:3;
    grid-column-start:11;

}


.buttonLargeSpec {
	width:100px;

}

</style>

<div class="container">
	<div class="header">
		<h1 style="color:dimgray"> <a style="text-decoration: none; color:dimgray;" href=/>Flashpape<span style="color:#4caf50">r</span> </a></h1>
	</div>
	<div class="box boxSpec">
		<h2 style="color:gray;">A better way to share passwords.</h2>
		<p style="color:gray;">Flashpaper provides you with a unique link that you can use to send a password or file to someone else. For security purposes, the link is only active for 24 hours.
			In the main menu, you can either generate a link by entering text or by providing a file. This link can then be opened by anyone and viewed. However, this can only be done
			once, after which the data is destroyed. This way, if the intended party tries to open the link and they can't, it means that someone has already accessed it. </p>
		<h4> Please be careful when opening links from people you do not trust.</h4>
		<a class="buttonLarge buttonLargeSpec" href=/> Done </a>
	</div>
</div>

`
const index = `

<style>
.container {
	display:grid;
    grid-template-columns: repeat(21, 0.2fr);
    grid-template-rows: repeat(15, 0.2fr);
    justify-content:space-evenly;
}
.box {
    grid-row-start:6;
    grid-column:10/13;
}

.header {
	grid-row-start:5;
    grid-column-start:11;
}

.buttonLarge {
	background-color: #4CAF50;
	color: white;
	padding: 15px 15px;
	text-align: center;
	text-decoration: none;
	display: block;
	font-size: 14px;
	width:150px;
	margin:8px auto;
    border-radius:4px;
}

.button3 {
	display: block;
}

</style>

<div class="container">
	<div class="header">
		<h1 style="color:dimgray"> <a style="text-decoration: none; color:dimgray;" href=/>Flashpape<span style="color:#4caf50">r</span> </a></h1>
	</div>
	<div class="box boxSpec">
		<a class="buttonLarge" href=/add>Share text</a>
		<a class="buttonLarge" href=/addfile>Share a file</a>
		<a class="buttonSmall button3"  href=/help>Learn More</a>
	</div>
</div>

`
const inputtextform = `

<style>

.boxSpec {
    grid-row:6;
    grid-column:8/15;
}

.header {
	grid-row-start:5;
    grid-column-start:11;
	font-family: "menlo";
}

.buttonLargeSpec {
	grid-row-start:6;
	grid-column:1/11;
}

.innerContainer {
	display:grid;
    grid-template-columns: repeat(10, 0.1fr);
    grid-template-rows: 1fr 1fr 1fr 1fr 1fr;
    justify-content:space-evenly;
	height:auto;
}

.textarea {
	grid-row:1/5;
	grid-column:1/11;
	border-color:lightgray;
    border-style:solid;
    border-width:thin;
    border-radius:4px;
	font-family:menlo;
	padding:8px 8px;
	margin:8px;
}

.checkBoxSpec {
	grid-row-start:5;
	grid-column:1/11;
}

</style>

<div class="container">
	<div class="header">
		<h1 style="color:dimgray"> <a style="text-decoration: none; color:dimgray;" href=/>Flashpape<span style="color:#4caf50">r</span> </a></h1>
	</div>
	<div class="boxSpec box">
		<form action="/add" method="POST">
			<div class="innerContainer">
				<textarea 	data-gramm="false" style="resize:none" class="textarea" name="secret" placeholder="Enter your text here..."></textarea>
				<div class="checkBox checkBoxSpec">
					%s
				</div>
				<input class="buttonLarge buttonLargeSpec" type="submit" value="Submit">
			<div/>
		</form> 
    </div>
</div>

`
const inputfileform = `

<style>
.boxSpec {
    grid-row-start:6;
    grid-column:9/14;

}

.header {
	grid-row-start:5;
    grid-column-start:11;
	    font-family: "menlo";
}

input[type="submit"] {
	background-color: #4CAF50;
	padding: 8px 8px;
	color:white;
	text-align: center;
	text-decoration: none;
	font-size: 14px;
    border-radius:4px;
	cursor:pointer;

}

input[type="submit"][disabled] {
	color:white;
	cursor:not-allowed;
	opacity:0.7;
}

.checkBoxSpec {
	grid-row-start:5;
	grid-column:1/11;
	display:block;
}

</style>

<div class="container">
	<div class="header">
		<h1 style="color:dimgray"> <a style="text-decoration: none; color:dimgray;" href=/>Flashpape<span style="color:#4caf50">r</span> </a></h1>
	</div>
	<div class="box boxSpec">
		<h5 style="color:#DD2727" id="fileSizeError"></h5>
		<form action="/addfile" method="POST" enctype="multipart/form-data">
			<input onchange="fileChecker(this)" type="file" name="file" id="file" style="font-family:menlo cursor:pointer">
			<div class="checkBox checkBoxSpec">
					%s
				</div>
			<input type="submit" value="Submit" id="submit" disabled>
		</form>
    </div>
</div>

<script>
function fileChecker(file) {

	var FileSize;
	try {

		FileSize = file.files[0].size / 1024 / 1024; //MB

	} catch (err) {
		FileSize = 11;
	}
	
	document.getElementById("submit").value="Submit";
	if (FileSize > 10) {
		document.getElementById("fileSizeError").innerHTML = "Please upload a file smaller than 10 Mb.";
		document.getElementById("submit").disabled = true;
		document.getElementById("submit").style.opacity = "0.7";

	} else {
		document.getElementById("submit").disabled = false;
		document.getElementById("fileSizeError").innerHTML = "";
		document.getElementById("submit").style.opacity = "1.0";

	}
}
</script>

`
const shareform = `

<script>
function copy() {
	var text = document.getElementById("link").innerText;
	var elem = document.createElement("textarea");
	document.body.appendChild(elem);
	elem.value = text;
	elem.select();
	document.execCommand("copy");
	document.body.removeChild(elem);

	var newText = text.fontcolor("gray");
	document.getElementById("link").innerHTML = newText;
	setTimeout(clear, 200, text);

}

function clear(text) {
	document.getElementById("link").innerHTML = text;
	document.getElementById("copied").innerHTML = "Copied";
}

</script>

<style>
.boxSpec {
    grid-row-start:6;
    grid-column:7/16;
}

.header {
	grid-row-start:5;
    grid-column-start:11;
	    font-family: "menlo";
}

.buttonLargeSpec {
	width:200px;
}

.buttonSpec {

	display: inline-block;

}

</style>


<div class="container">
	<div class="header">
		<h1 style="color:dimgray"> <a style="text-decoration: none; color:dimgray;" href=/>Flashpape<span style="color:#4caf50">r</span> </a></h1>
	</div>
	<div class="box boxSpec">
        <p style="color:#A0A0A0;">Give this link, which will expire in %d hours or when opened, to the person you want to share information with.</p>

		<div class="box2">
			<h4 id="link" onclick="copy()">%s://%s/%s</h4>
		</div>
        
        <button class="buttonSmall buttonSpec" id="copied" onclick="copy()">Copy</button>
		<span style="color:gray;">&#183;</span>
		<button class="buttonSmall buttonSpec" onClick="window.location.reload();">Generate another link</button>
        <br>
        <h5 style="color:gray;"></h5>

		<a class="buttonLarge buttonLargeSpec" href="/">Generate a new secret</a>

    </div>
</div>


`
const ascii = `
___________.__                .__                                      
\_   _____/|  | _____    _____|  |__ ___________  ______   ___________ 
 |    __)  |  | \__  \  /  ___/  |  \\____ \__  \ \____ \_/ __ \_  __ \
 |     \   |  |__/ __ \_\___ \|   Y  \  |_> > __ \|  |_> >  ___/|  | \/
 \___  /   |____(____  /____  >___|  /   __(____  /   __/ \___  >__|   
     \/              \/     \/     \/|__|       \/|__|        \/     

Welcome to flashpaper!
Your server has been started and is running...
`
const checkBox = `
<input type="checkbox" name="auth" id="auth" checked>
<label for="auth" style="color:gray; font-size:80%;">Require Auth when viewing?</label>

`
