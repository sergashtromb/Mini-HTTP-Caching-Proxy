Mini HTTP Caching Proxy 


To get the config file, run the proxy with the parameter. When running with the parameter, a configuration file will be generated.

```
proxy -g nameConfig.yaml
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
	- [ ] Number of requests
	- [ ] Number of processed requests
	- [ ] CacheStore load
