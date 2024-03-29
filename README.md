# GBB

A simple bulletin board writed in Go for the  terminal to run in a shared *nix
server. It has a simple board with threads or conversations sorted by the last
message date. It hasn't a users database and it uses the system common
users. For admin tasks you must be logged as root (or sudo). 

It's written in Go entirely using [tcell](https://github.com/gdamore/tcell)
library for the user interface. Use a SQLite file to store threads and messages.

The keys to use it are:

- `a`: Add a new thread or new message
- `d`: Delete a thread or a message. Only for admin or for the author 
- `e`: Edit a message. Only for admin or for the author 
- `b`: Search thread by a filter
- `f`: Fix a thread in the header of the board. Only for the admin
- `c`: Close a thread. Only for the admin
- `↑↓`: With arrows keys you can navegate into threads or the replies
- `AvPg/RePg`: To navigate inside the pages of a reply, If it is too long to show it in a screen
- `?`: Show the help
- `ESC`: Return to the previous panel or quit the application


All the help messages are in spanish but if you want to use `gbb` in your server
and you want change them, send PR or ask me.


![](stuff/gbb-snapshot.png)



## Status

- [X] User Interface
- [X] API REST server
- [X] Auth users
- [X] Scripts to manage the database
- [ ] Emailing users to notify mentions
- [ ] Add key to go to the end of a thread


## Build

To compile `gbb` you must be installed Go17 or newest. Only type:

```
go build
```
