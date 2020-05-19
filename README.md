# GOZ phase-1 updater

## Build
`go get github.com/ironman0x7b2/goz-phase-1-updater`

or

`mkdir -p $GOPATH/src/github.com/ironman0x7b2`

`cd $GOPATH/src/github.com/ironman0x7b2`

`git clone https://github.com/ironman0x7b2/goz-phase-1-updater.git`

`cd goz-phase-1-updater`

`go install`

## Configure
It depends on the config file of [relayer](https://github.com/iqlusioninc/relayer) program.

After setting up the relayer configuration you have to pass the specific flags.

For more info try `goz-phase-1-updater --help`
 
## Run
`goz-phase-1-updater --src SOURCE_CHAIN_ID --dst DESTINATION_CHAIN_ID --path PATH_NAME --duration DURATION`

Example: `goz-phase-1-updater --src ibc0 --dst ibc1 --path ibc0ibc1 --duration 2m`