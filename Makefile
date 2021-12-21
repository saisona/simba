# File              : Makefile
# Author            : Alexandre Saison <alexandre.saison@inarix.com>
# Date              : 21.12.2021
# Last Modified Date: 21.12.2021
# Last Modified By  : Alexandre Saison <alexandre.saison@inarix.com>


init:
	go build cmd/*.go  

test: 
	go test -test.v -test.shuffle on -test.failfast -timeout 10s -cover -coverprofile=prof.out

coverage: test
	go tool cover -html=prof.out

run:
	go run cmd/*.go
