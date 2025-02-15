module github.com/USA-RedDragon/rtz-server

go 1.24.0

require (
	capnproto.org/go/capnp/v3 v3.0.1-alpha.2
	github.com/aws/aws-sdk-go-v2 v1.36.1
	github.com/aws/aws-sdk-go-v2/config v1.29.6
	github.com/aws/aws-sdk-go-v2/service/s3 v1.77.0
	github.com/gin-contrib/cors v1.7.3
	github.com/gin-contrib/pprof v1.5.2
	github.com/gin-gonic/gin v1.10.0
	github.com/glebarez/sqlite v1.11.0
	github.com/go-errors/errors v1.5.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/klauspost/compress v1.17.11
	github.com/mattn/go-nulltype v0.0.0-20230117041332-6715e831ac05
	github.com/nats-io/nats.go v1.39.0
	github.com/prometheus/client_golang v1.20.5
	github.com/puzpuzpuz/xsync/v3 v3.5.1
	github.com/spf13/cobra v1.9.0
	github.com/spf13/pflag v1.0.6
	github.com/uptrace/opentelemetry-go-extra/otelgorm v0.3.2
	github.com/ztrue/shutdown v0.1.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.59.0
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
	golang.org/x/sync v0.11.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.5.7
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.12
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.9 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.59 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.14 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.12.7 // indirect
	github.com/bytedance/sonic/loader v0.2.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/colega/zeropool v0.0.0-20230505084239-6fb4a4f75381 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.0.0 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.24.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/google/pprof v0.0.0-20221118152302-e6195bd50e26 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/onsi/ginkgo/v2 v2.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/quic-go v0.43.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.3.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/arch v0.13.0 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/exp v0.0.0-20240604190554-fc45aab8b7f8 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	google.golang.org/protobuf v1.36.3 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)

replace github.com/gin-gonic/gin v1.10.0 => github.com/USA-RedDragon/gin v1.10.1-0.20240728192152-87d4db66086b
