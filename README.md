# go-flashpaper
Go-flashpaper is a simple go-based service for creating one time use links to text data or individual files.

## What even is this?

It is a web service that allows you to upload text snippets or files and generates one time use links to share these things with other people. As soon as the sharing link is accessed, the data is deleted from the web service's memory and the link expires. This means that old links are useless, even if shared somewhere insecure. This can be used to share sensitive data with friends or colleagues. Flashpaper has a maximum data retention period of 24 hours. It has no build dependencies outside of the go standard library.

## Installation

0. Get a server. Preferably one you run yourself. Preferably patched. Install go (the language).
1. Disable swap on the server so you don't write secrets to disk.
2. Clone this here repo: `git clone git@github.com:rawdigits/go-flashpaper.git`
3. Build go-flashpaper: `cd go-flashpaper; go build`
4. Get ahold of a TLS certificate. go-flashpaper requires TLS, because we are not savages.
5. Save the certificate and private key in the same directory as go-flashpaper. Name them `server.crt` and `server.key`, respectively
6. Run go-flashpaper: `./go-flashpaper`
7. Connect to the web service via `https://yourserver.example.com:8443` (note: 8443 is the default port, so no one has an excuse to run this as root)
8. Share secret things.
 
## FAQ

Q: *How do I configure it?*

A: You don't. Everything is hardcoded because configuration is the devil. The service needs the files `server.crt` and `server.key` in its working directory to start. The port it runs on is 8443.

Q: *Should I run this on (cloud provider x)?*

A: That is up to you and your individual risk appetite. This comes with no warranty or promise of security, but it is probably way better than using gtalk to share your root password.

Q: *Should I put this on the public internet?*

A: Probably not. It is susceptible to Denial of Service by simply uploading a lot of data. Put it behind some kind of auth or something.

Q: *Why do you limit uploads to 10mb?*

A: Because everything is kept in memory and we ain't made of money.

Q: *Why doesn't it have (feature x)?*

A: Because. Also it never will.

Q: *Why is mail not being delivered?*

A: You might have outbound firewall rules on the host where this is deployed that prevent `sendmail` from doing its job. You could have deployed this on a host that somehow got really bad sender reputation. Or your `sendmail` is broken somehow (unlikely.)

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

### Example 4

**Alice**: Hi team, here is the new shared Photobucket password. Check your inboxes for the link to see it!

**Bob**: Yay!

**Carol**: Yay!

**Dave**: I hope we don't get banned again.
## Have a nice day!
