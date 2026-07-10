module github.com/mkongthong-work/bms-be

go 1.22.2

require (
	github.com/signintech/gopdf v0.32.0
	golang.org/x/crypto v0.24.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/xuri/efp v0.0.0-20231025114914-d1ff6096ae53 // indirect
	github.com/xuri/nfp v0.0.0-20230919160717-d98342af3f05 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/phpdave11/gofpdi v1.0.14-0.20211212211723-1f10f9844311 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/xuri/excelize/v2 v2.8.1
)

replace golang.org/x/crypto => github.com/golang/crypto v0.24.0

replace golang.org/x/net => github.com/golang/net v0.26.0

replace golang.org/x/text => github.com/golang/text v0.16.0

replace golang.org/x/sync => github.com/golang/sync v0.7.0

replace golang.org/x/sys => github.com/golang/sys v0.21.0

replace gopkg.in/yaml.v3 => ./third_party/yaml

replace golang.org/x/image => github.com/golang/image v0.18.0
