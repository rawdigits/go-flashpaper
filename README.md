# Flashpaper

Flashpaper is a simple go-based service for creating one time use links to text data or individual files.


## What is this?

It is a web service that allows you to upload text snippets or files and generates one time use links to share these things with other people. As soon as the sharing link is accessed, the data is deleted from the web service's memory and the link expires. This means that old links are useless, even if shared somewhere insecure. This can be used to share sensitive data with friends or colleagues. Flashpaper has a maximum data retention period of 24 hours. It has no build dependencies outside of the go standard library.

## Changelog

1. Let's Encrypt support for auto-certificate generation. 
2. A new UI.
3. Optional integration with a Canarytoken.
4. Optional multi-user authentication using BasicAuth.

## Installation

1. Get a server - preferably one you run yourself and install Golang.
2. Disable swap on the server so you don't write secrets to disk.
3. Build flashpaper: `go build`
4. Run go-flashpaper: `./flashpaper` specifying all optional parameters.
5. Connect to the web service via `https://yourserver.example.com:8443` (note: 8443 is the default port, so no one has an excuse to run this as root)
6. Share secret things.

For development purposes, the server can be setup to run on your local machine. Simply run the code and navigate to: <br>
`https://localhost:8443`<br>
Browser dependent, you may get a security warning. Simply click on `Advanced` and proceed. 

## Usage

As mentioned in the Installation section, optional parameters can be specified to customise the server. The flags are as follows:
1. `-auth {"filename.txt}`: specify this flag to enable authentication. Leave it out to remove authentication.
2. `-token {canary token link}`: specify this to be notified when a link has been accessed more than once. Leave it out to not enable it.
3. `-autocert {true}`: specify this to make use of autocert so that you don't need to specify your own cert files.

#### Example
1. Specify an auth file, a token and autocert. <br>
`./flashpaper -auth auth.txt -autocert true -token http://canarytokens.com/tags/feedback/terms/...`
2. Use no auth nor a token. <br>
`./flashpaper`


## Canarytoken integration

The basis of this tool is that a link is only accessed once. However, valuable information may be gathered if a user clicks on a link that has already been used. Subsequently, integration with `canarytokens.org` has been provided. If a user clicks on a link that has already been used, their browser will be fingerprinted and a notification sent. The setup process is as follows:
1. Visit [canarytokens.org/generate](http://canarytokens.org/generate).
2. Click on the dropdown box and select either a `Fast Redirect` or `Slow Redirect` token. The trade-off is as follows:
	- A `Fast Redirect` will not be visible in the redirect process. This means that the user will not know that they have been caught. However, less information can be gathered.
	- A `Slow Redirect` will be visible to the user giving away your position. However, you will be able to gather more information about their browser.<br>
Both work, it is simply a matter of priorities.
3. Specify the email address where you would like to be notified.
4. Provide a note.
5. Provide a redirect link. As mentioned in the code, it is advisable to use `https://yourserver.example.com:8443/usedlink` so that the user is still met with a Flashpaper screen after the redirect. Note that you will not be able to test this on your localhost as Canarytokens does not allow for generation of tokens with localhost as a URL.
6. Generate the token. You will be provided with a link that will be used in the Usage section.

## Multi-user Authentication

If required, authentication can be specified for creating and viewing secrets. Validation is done against a text file, specified by you, which stores a hash of a username and password on a line. This way, new credentials can be specified. If the file is not found, a file with the specified name will be created for you with default credentials. This file is read on each Auth attempt meaning that users can be added / removed without having to restart the server. The default credentials specified in the code are as follows:<br>
`Username: user1`<br>
`Password: pass1`


## Let's Encrypt

Functionality has been provided to auto-generate `tls` certificates which means that you don't need to specify certificates manually. 


## Systemd service ##

It is important to note that if the service restarts we will lose all current secrets (as they are stored in memory and pointed to by the process). Running it as a systemd service and auto-restarting will mean that we won't notice if it restarts and old links may fail. This might lead us to think that the link has been used before. We should check logs to see if this is the case or if it was due to a restart:

`sudo journalctl -u go-flashpaper`

If you are happy with this then:

1. Specify your arguments in the `go-flashpaper.service` file. Add them to the end of `ExecStart=...`.
2. Copy `systmd/go-flashpaper.service` to `/lib/systemd/system/`
3. `sudo chmod 755 /lib/systemd/system/go-flashpaper.service`
4. `sudo systemctl enable go-flashpaper.service`
5. `sudo systemctl start go-flashpaper`

## Manually running in background ##

Once up and running, you can run with:

	`nohup ./go-flashpaper &`

This will run in a background process and log to `nohup.out`. The service will not be automatically restarted.

<em> *Please note that this feature has not yet been tested with the new updates. </em>

## FAQ

Q: *How do I configure it?*

A: You don't. As shown in the Usage, after building `flashpaper.go` you simply run it with the required flags.

Q: *Should I run this on (cloud provider x)?*

A: That is up to you and your individual risk appetite. This comes with no warranty or promise of security, but it is probably way better than using gtalk to share your root password.

Q: *Should I put this on the public internet?*

A: Probably not. It is susceptible to Denial of Service by simply uploading a lot of data. 

Q: *Why do you limit uploads to 10mb?*

A: Because everything is kept in memory and we ain't made of money.


## What are some examples of using this?

### Example 1

**Alice**: Hi Bob, I just set your new linux account with a random password. The password is available here: `https://seekret.example.com:8443/de818e28daa568cb433b39da292b589bdcbc1bc771d52cffe453b0e01e93865b` Please log in and change your password immediately.

**Bob**: Thanks, I logged in with that password and changed my password!

### Example 2

**Alice**: Hi Bob, I just set your new linux account with a random password. The password is available here: `https://seekret.example.com:8443/de818e28daa568cb433b39da292b589bdcbc1bc771d52cffe453b0e01e93865b` Please log in and change your password immediately.

**Bob**: Hmmm, that link didn't work.

**Alice**: It hasn't been 24 hours, so someone may have intercepted it. I'll delete that account and create a new one. Then I'll send you another link by carrier pigeon.

**Bob**: Perfect, thanks!

### Example 3

**Alice**: Hi Bob, I have that document on our new security infrastructure plans. Go here to download it: `https://seekret.example.com/de818e28daa568cb433b39da292b589bdcbc1bc771d52cffe453b0e01e93865b`

**Bob**: Yay!

## Have a nice day!
