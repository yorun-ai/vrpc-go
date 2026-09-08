package vrpc

import (
	"fmt"
	"sync"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

type MethodSpec struct {
	Name                        string
	SkelName                    string
	ArgumentsSensitive          bool
	ResultSensitive             bool
	ArgumentsContainsBinaryType bool
	ResultContainsBinaryType    bool
}

type ServiceSpec struct {
	Name     string
	SkelName string
	Methods  []MethodSpec
}

type MethodInfo struct {
	serviceName string
	service     string
	path        string
	spec        MethodSpec
}

func (m MethodInfo) ServiceName() string {
	return m.serviceName
}

func (m MethodInfo) Name() string {
	return m.spec.Name
}

func (m MethodInfo) ServiceSkelName() string {
	return m.service
}

func (m MethodInfo) SkelName() string {
	return m.spec.SkelName
}

func (m MethodInfo) FullURLPath() string {
	return m.path
}

func (m MethodInfo) ArgumentsSensitive() bool {
	return m.spec.ArgumentsSensitive
}

func (m MethodInfo) ResultSensitive() bool {
	return m.spec.ResultSensitive
}

func (m MethodInfo) ArgumentsContainsBinaryType() bool {
	return m.spec.ArgumentsContainsBinaryType
}

func (m MethodInfo) ResultContainsBinaryType() bool {
	return m.spec.ResultContainsBinaryType
}

type _ServiceInfo struct {
	methods map[string]MethodInfo
}

// Its zero value is ready to use. Registration and lookups are concurrency-safe.
// A service is registered atomically and cannot be replaced.
type Registry struct {
	mu       sync.RWMutex
	services map[string]_ServiceInfo
}

func NewRegistry() *Registry {
	return new(Registry)
}

var defaultRegistry = NewRegistry()

func Register(spec *ServiceSpec) {
	defaultRegistry.Register(spec)
}

func GetMethodInfo(serviceSkelName, methodSkelName string) (MethodInfo, bool) {
	return defaultRegistry.GetMethodInfo(serviceSkelName, methodSkelName)
}

// Register validates and snapshots a service before making it visible to callers.
// Later edits to the input struct or method slice do not alter registered metadata.
func (r *Registry) Register(spec *ServiceSpec) {
	if spec == nil {
		panic(fmt.Errorf("vrpc: nil service spec"))
	}
	if _, _, err := rpchttp.ParseServiceAndMethodFromPath("/" + spec.SkelName + "/method"); err != nil || spec.SkelName == "." || spec.SkelName == ".." {
		panic(fmt.Errorf("vrpc: invalid service name %q", spec.SkelName))
	}

	methods := make(map[string]MethodInfo, len(spec.Methods))
	for _, method := range spec.Methods {
		if _, _, err := rpchttp.ParseServiceAndMethodFromPath("/" + spec.SkelName + "/" + method.SkelName); err != nil {
			panic(fmt.Errorf("vrpc: invalid method name %q", method.SkelName))
		}
		if _, ok := methods[method.SkelName]; ok {
			panic(fmt.Errorf("vrpc: duplicate method %s/%s", spec.SkelName, method.SkelName))
		}

		methods[method.SkelName] = MethodInfo{
			serviceName: spec.Name,
			service:     spec.SkelName,
			path:        "/" + spec.SkelName + "/" + method.SkelName,
			spec:        method,
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.services[spec.SkelName]; ok {
		panic(fmt.Errorf("vrpc: service %s already registered", spec.SkelName))
	}
	if r.services == nil {
		r.services = make(map[string]_ServiceInfo)
	}
	r.services[spec.SkelName] = _ServiceInfo{
		methods: methods,
	}
}

func (r *Registry) GetMethodInfo(serviceSkelName, methodSkelName string) (MethodInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.services[serviceSkelName].methods[methodSkelName]
	return info, ok
}

type _InvokeEncoding struct {
	request _Codec
	binary  bool
	accept  string
}

func (m MethodInfo) invokeEncoding() _InvokeEncoding {
	_, accept := rpchttp.RequestContentTypes(m.ArgumentsContainsBinaryType(), m.ResultContainsBinaryType())
	return _InvokeEncoding{
		request: _Codec(m.ArgumentsContainsBinaryType()),
		binary:  m.ResultContainsBinaryType(),
		accept:  accept,
	}
}
