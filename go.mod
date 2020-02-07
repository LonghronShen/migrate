module github.com/shaoding/migrate

require (
	cloud.google.com/go v0.36.0
	github.com/aws/aws-sdk-go v1.19.7
	github.com/bitly/go-hostpool v0.0.0-20171023180738-a3a6125de932 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/cockroachdb/cockroach-go v0.0.0-20181001143604-e0a95dfd547c
	github.com/cznic/ql v1.2.0
	github.com/denisenkom/go-mssqldb v0.0.0-20190315220205-a8ed825ac853
	github.com/dhui/dktest v0.3.0
	github.com/docker/docker v0.7.3-0.20190108045446-77df18c24acf
	github.com/fsouza/fake-gcs-server v1.5.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gocql/gocql v0.0.0-20190301043612-f6df8288f9b4
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/protobuf v1.3.0 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/hashicorp/go-multierror v1.0.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.2.0+incompatible // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/kshvakov/clickhouse v1.3.5
	github.com/lib/pq v1.0.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/mongodb/mongo-go-driver v0.3.0
	github.com/nakagami/firebirdsql v0.0.0-20190310045651-3c02a58cfed8
	github.com/pkg/errors v0.8.1 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24 // indirect
	github.com/sirupsen/logrus v1.3.0 // indirect
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/tidwall/pretty v0.0.0-20180105212114-65a9db5fad51 // indirect
	github.com/xanzy/go-gitlab v0.15.0
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v1.0.0 // indirect
	gitlab.com/nyarla/go-crypt v0.0.0-20160106005555-d9a5dc2b789b // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
	golang.org/x/tools v0.0.0-20190312170243-e65039ee4138
	google.golang.org/api v0.3.0
	google.golang.org/genproto v0.0.0-20190307195333-5fe7a883aa19
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/goracle.v2 v2.8.1
)

replace (
	cloud.google.com/go => github.com/GoogleCloudPlatform/google-cloud-go v0.37.2
	golang.org/x/build => github.com/golang/build v0.0.0-20190401232323-335aaf83e1b5
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20190325154230-a5d413f7728c
	golang.org/x/exp => github.com/golang/exp v0.0.0-20190321205749-f0864edee7f3
	golang.org/x/image => github.com/golang/image v0.0.0-20190321063152-3fc05d484e9f
	golang.org/x/lint => github.com/golang/lint v0.0.0-20190313153728-d0100b6bd8b3
	golang.org/x/mobile => github.com/golang/mobile v0.0.0-20190327163128-167ebed0ec6d
	golang.org/x/net => github.com/golang/net v0.0.0-20190328230028-74de082e2cca
	golang.org/x/oauth2 => github.com/golang/oauth2 v0.0.0-20190319182350-c85d3e98c914
	golang.org/x/perf => github.com/golang/perf v0.0.0-20190312170614-0655857e383f
	golang.org/x/sync => github.com/golang/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/sys => github.com/golang/sys v0.0.0-20190329044733-9eb1bfa1ce65
	golang.org/x/text => github.com/golang/text v0.3.0
	golang.org/x/time => github.com/golang/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools => github.com/golang/tools v0.0.0-20190401205534-4c644d7e323d
	google.golang.org/api => github.com/googleapis/google-api-go-client v0.3.0
	google.golang.org/appengine => github.com/golang/appengine v1.5.0
	google.golang.org/genproto => github.com/google/go-genproto v0.0.0-20190401181712-f467c93bbac2
	google.golang.org/grpc => github.com/grpc/grpc-go v1.19.1
)

go 1.13
