package kubernetes

const (
	Namespace        string = "namespace"
	PodLabels               = "pod_labels"
	LabelSelector           = "label_selector"
	PortName                = "port_name"
	RemotingPortName        = "remoting_port_name"
)

// Option represents the kubernetes provider option
type Option struct {
	// KubeConfig represents the kubernetes configuration
	KubeConfig string
	// NameSpace specifies the namespace
	NameSpace string
	// PodLabels defines the pod labels
	PodLabels map[string]string
	// Label Selector
	LabelSelector string
	// Specifies the port name
	PortName string
	// Specifies the remoting port name
	// This port is necessary to send remote messages to node
	RemotingPortName string
}
