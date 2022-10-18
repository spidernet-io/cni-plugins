package schema

import (
	"github.com/spidernet-io/spiderpool/pkg/constant"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// MultusGroupVersion is group version used to register these objects
	MultusGroupVersion = schema.GroupVersion{Group: "k8s.cni.cncf.io", Version: "v1"}

	// MultusSchemeBuilder is used to add go types to the GroupVersionKind scheme
	MultusSchemeBuilder = &scheme.Builder{GroupVersion: MultusGroupVersion}

	// MultusAddToScheme adds the types in this group-version to the given scheme.
	MultusAddToScheme = MultusSchemeBuilder.AddToScheme
)

var (
	// SpiderPoolGroupVersion is group version used to register these objects
	SpiderPoolGroupVersion = schema.GroupVersion{Group: constant.SpiderpoolAPIGroup, Version: constant.SpiderpoolAPIVersionV1}

	// SpiderPoolSchemeBuilder is used to add go types to the GroupVersionKind scheme
	SpiderPoolSchemeBuilder = &scheme.Builder{GroupVersion: SpiderPoolGroupVersion}

	// SpiderPoolAddToScheme adds the types in this group-version to the given scheme.
	SpiderPoolAddToScheme = SpiderPoolSchemeBuilder.AddToScheme
)
