# Changelog

## v1.0.1
BUGFIX
* [#99](https://github.com/bnb-chain/greenfield-cmd/pull/99) fix put bucket policy with object actions and help info

## v1.0.0
FEATURES
* [#94](https://github.com/bnb-chain/greenfield-cmd/pull/94) support multi-account management and set-default command
* [#95](https://github.com/bnb-chain/greenfield-cmd/pull/95) update dependency of go-sdk and greenfield to v1.0.0

## v0.1.1
FEATURES
* [#89](https://github.com/bnb-chain/greenfield-cmd/pull/89)  support resumable download by adding a flag to "object get cmd"
* [#90](https://github.com/bnb-chain/greenfield-cmd/pull/90)  improve cmd of downloading and uploading with printing progress details and speed
* [#90](https://github.com/bnb-chain/greenfield-cmd/pull/92)  update dependency of go-sdk and greenfield to v0.2.6

BUGFIX
* [#90](https://github.com/bnb-chain/greenfield-cmd/pull/90)  solve the problem of parsing object name err when recursive upload folder

## v0.1.0

FEATURES
* [#83](https://github.com/bnb-chain/greenfield-cmd/pull/83)  support uploading multiple files or folder by one command 

## v0.1.0-alpha.2

FEATURES
* [#80](https://github.com/bnb-chain/greenfield-cmd/pull/80) update depenency and support group new API, including "group ls ", "group ls-member", "group ls-belong" and "policy ls"

## v0.1.0-alpha.1

FEATURES
* [#75](https://github.com/bnb-chain/greenfield-cmd/pull/75)  update go-sdk and greenfield version, add expire time to group member and support renew group cmd

REFACTOR
* [#72](https://github.com/bnb-chain/greenfield-cmd/pull/72)  refactor: improve list , delete cmd and the return format

## v0.0.9

FEATURES
* [#50](https://github.com/bnb-chain/greenfield-cmd/pull/50) feat: support resumable upload
* [#56](https://github.com/bnb-chain/greenfield-cmd/pull/56) support version cmd
* [#64](https://github.com/bnb-chain/greenfield-cmd/pull/64) chore: update go-sdk and greenfield version and support both of endpoint or sp operator address to query SP

BUGFIX
* [#61](https://github.com/bnb-chain/greenfield-cmd/pull/61)  fix: remove password for query commands

REFACTOR
* [#54](https://github.com/bnb-chain/greenfield-cmd/pull/54)  refactor: replace the keystore command with account cmd

## v0.0.9-alpha.3

FEATURES
* [#64](https://github.com/bnb-chain/greenfield-cmd/pull/64) chore: update go-sdk and greenfield version and support both of endpoint or sp operator address to query SP 

BUGFIX
* [#61](https://github.com/bnb-chain/greenfield-cmd/pull/61)  fix: remove password for query commands
