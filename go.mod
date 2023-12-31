module github.com/kong/go-database-reconciler

go 1.21.1

replace github.com/yudai/gojsondiff v1.0.0 => github.com/Kong/gojsondiff v1.3.0

require (
	dario.cat/mergo v1.0.0
	github.com/Kong/gojsondiff v1.3.2
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/alecthomas/jsonschema v0.0.0-20191017121752-4bb6e3fae4f2
	github.com/blang/semver/v4 v4.0.0
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/fatih/color v1.15.0
	github.com/google/go-cmp v0.6.0
	github.com/google/go-querystring v1.1.0
	github.com/google/uuid v1.5.0
	github.com/hashicorp/go-memdb v1.3.4
	github.com/hashicorp/go-retryablehttp v0.7.5
	github.com/hexops/gotextdiff v1.0.3
	github.com/kong/deck v1.29.3-0.20231114000742-27bb40ad93d6
	github.com/kong/go-kong v0.49.0
	github.com/shirou/gopsutil/v3 v3.23.12
	github.com/ssgelm/cookiejarparser v1.0.1
	github.com/stretchr/testify v1.8.4
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/sync v0.6.0
	golang.org/x/term v0.16.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	atomicgo.dev/cursor v0.1.1 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.0.2 // indirect
	github.com/Kong/go-diff v1.2.2 // indirect
	github.com/adrg/strutil v0.2.3 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/daveshanley/vacuum v0.2.7 // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/getkin/kin-openapi v0.108.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gookit/color v1.5.3 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/yaml v0.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kong/go-apiops v0.1.23 // indirect
	github.com/kong/go-slugify v0.0.0-20231027194833-8e212cc29c16 // indirect
	github.com/kong/semver/v4 v4.0.1 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mozillazg/go-unidecode v0.2.0 // indirect
	github.com/pb33f/libopenapi v0.9.6 // indirect
	github.com/pb33f/libopenapi-validator v0.0.10 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/pterm/pterm v0.12.62 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.0 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.16.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/tidwall/gjson v1.17.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/vmware-labs/yaml-jsonpath v0.3.2 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/exp v0.0.0-20220909182711-5c715a9e8561 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
