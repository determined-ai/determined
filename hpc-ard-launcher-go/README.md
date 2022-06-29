# hpc-ard-launcher-go

This repo is the home of the Capsules (hpc-ard-capsules-core) dispatch server Go client.

The code found here is generated automatically using openapi tools from the Capsules REST API specification. It can be build wit the following command line executed in the hpc-ard-capsules-core project:

```
mvn -pl com.cray.analytics.capsules:capsules-dispatch-client clean generate-sources -P go-client
```
To install the package to your Go environment:

If you use ssh to interact with github.hpe.com, add the following to your ~/.gitconfig:
```
[url "ssh://git@github.hpe.com/"]
        insteadOf = https://github.hpe.com/
```
Then:
```
% export GOPRIVATE=github.hpe.com/hpe/hpc-ard-launcher-go
% go get github.hpe.com/hpe/hpc-ard-launcher-go/launcher
```
Import the launcher package to your Go program thus:
```
import (
	<other imports go here>

	"github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
)
```