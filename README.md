kantaloupe
==========

## Description


## Build

### Generate proto files

```bash
make genproto

```

### Build Apiserver

```bash
make apiserver

```


## Run

```bash
./bin/kantaloupe-apiserver --v=4
curl -X GET http://localhost:8000/apis/kantaloupe.dynamia.ai/v1/clusters
```
