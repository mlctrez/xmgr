# xmgr

### Xonotic server manager. 

> [!CAUTION]
> This is a work in progress and may not be complete or fully tested.

### Rationale

A service responsible for starting/restarting a xonotic dedicated server process and providing 
http endpoints for serving TOS and xonotic assets.

### Requirements

* Linux server with xonotic user
* Xonotic repository and build artifacts under ~/xonotic
  * Follow source build steps at https://gitlab.com/xonotic/xonotic/-/wikis/Repository_Access
  * After building the dedicated binary with `./all compile dedicated`, the output binary
  `~/xonotic/darkplaces/darkplaces-dedicated` should be copied 
  to `~xonotic/xonotic-local-dedicated`  
* Reverse proxy with [Let's Encrypt](https://letsencrypt.org/)
* DNS entries

### TODO list / Wish List
* Add API for rebuilding the dedicated server.
* Add start / stop / restart of server in the api.
  * Right now the xonotic server only starts and stops when the service starts and stops.
* Add API to download new maps from map distribution servers.
* Add API for uploading maps / assets.

* Create a web interface for adjusting console variables.
  * Persist console variables to DB?
   

[![Go Report Card](https://goreportcard.com/badge/github.com/mlctrez/xmgr)](https://goreportcard.com/report/github.com/mlctrez/xmgr)

created by [tigwen](https://github.com/mlctrez/tigwen)
