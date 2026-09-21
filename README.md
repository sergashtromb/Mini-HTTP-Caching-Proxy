Mini HTTP Caching Proxy 


To get the config file, run the proxy with the parameter. When running with the parameter, a configuration file will be generated.

```
proxy -g nameConfig.yaml
```

# EVN params

```
APP_PORT - port for app, default 8888
HOST - host your app, default "0.0.0.0" 
LOG_LEVEL - level for debuging (error, debug, info, warn), default info
CACHE_IN_RAM - determines whether the cache will be in RAM or on disk 1 - true, 0 - false
TMP_PATH - path for tmp files cache
LIST_HOSTS - for reversed proxy
```

# Task

### Critical
- [x] Fix a bug with returning a nil result at the start of proxy operation
- [x] Enable file restore from file store (new ones are currently being created)
- [x] Consider the header returned from the Cache-Control service with the cache lifetime
- [x] Set up a reverse or transparent proxy mode

### Future Work
- [x] Add basic authorization
- [x] Add metrics:
	- [x] Number of requests
	- [x] Number of processed requests
	- [ ] CacheStore load
