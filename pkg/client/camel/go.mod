module github.com/apache/camel-k/pkg/client/camel

go 1.13

require (
	github.com/apache/camel-k/pkg/apis/camel v1.1.0
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/code-generator v0.18.8 // indirect
)

// Local modules
replace github.com/apache/camel-k/pkg/apis/camel => ../../apis/camel
