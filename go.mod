module github.com/kart-io/sentinel-x

go 1.25.0

require (
	github.com/casbin/casbin/v2 v2.134.0
	github.com/gin-gonic/gin v1.11.0
	github.com/go-playground/locales v0.14.1
	github.com/go-playground/universal-translator v0.18.1
	github.com/go-playground/validator/v10 v10.28.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/kart-io/logger v0.2.2
	github.com/kart-io/version v1.1.0
	github.com/labstack/echo/v4 v4.13.4
	github.com/redis/go-redis/v9 v9.17.2
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	go.uber.org/automaxprocs v1.5.3
	golang.org/x/crypto v0.46.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.6.1 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.14.2 // indirect
	github.com/bytedance/sonic/loader v0.4.0 // indirect
	github.com/casbin/govaluate v1.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.10 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.uber.org/mock v0.5.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251020155222-88f65dc88635 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251111163417-95abcf5c77ba // indirect
)

replace github.com/kart-io/goagent => ./staging/src/github.com/kart-io/goagent

replace gin-reload-example => ./staging/src/github.com/kart-io/logger/example/gin-reload

replace github.com/kart-io/logger/example/integrations => ./staging/src/github.com/kart-io/logger/example/integrations

replace github.com/kart-io/logger/example/integrations/gin => ./staging/src/github.com/kart-io/logger/example/integrations/gin

replace github.com/kart-io/logger/example/slog => ./staging/src/github.com/kart-io/logger/example/slog

replace github.com/kart-io/logger/example/performance => ./staging/src/github.com/kart-io/logger/example/performance

replace rotation-example => ./staging/src/github.com/kart-io/logger/example/rotation

replace github.com/kart-io/logger/example/zap => ./staging/src/github.com/kart-io/logger/example/zap

replace github.com/kart-io/logger/example/echo => ./staging/src/github.com/kart-io/logger/example/echo

replace github.com/kart-io/logger/example/reload => ./staging/src/github.com/kart-io/logger/example/reload

replace github.com/kart-io/logger/example/otlp => ./staging/src/github.com/kart-io/logger/example/otlp

replace github.com/kart-io/logger/example/configuration => ./staging/src/github.com/kart-io/logger/example/configuration

replace github.com/kart-io/logger/example/gin => ./staging/src/github.com/kart-io/logger/example/gin

replace github.com/kart-io/logger/example/comprehensive => ./staging/src/github.com/kart-io/logger/example/comprehensive

replace github.com/kart-io/logger/example/web-frameworks => ./staging/src/github.com/kart-io/logger/example/web-frameworks

replace github.com/kart-io/logger/e2e => ./staging/src/github.com/kart-io/logger/e2e

replace github.com/kart-io/logger => ./staging/src/github.com/kart-io/logger
