module github.com/infralight/k8s-collector

go 1.16

require (
	github.com/ido50/requests v1.2.0
	github.com/jgroeneveld/schema v1.0.0 // indirect
	github.com/jgroeneveld/trial v2.0.0+incompatible
	github.com/rs/zerolog v1.22.0
	github.com/thoas/go-funk v0.9.1
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	helm.sh/helm/v3 v3.6.0
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	rsc.io/letsencrypt v0.0.3 // indirect
)
