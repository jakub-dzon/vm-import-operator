package templates

import (
	"context"
	"fmt"
	"strings"

	templatev1 "github.com/openshift/api/template/v1"
	v1 "github.com/openshift/api/template/v1"
	tempclient "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TemplateOsLabel is a label that specifies the OS of the template
	TemplateOsLabel = "os.template.kubevirt.io/%s"

	// TemplateWorkloadLabel is a label that specifies the workload of the template
	TemplateWorkloadLabel = "workload.template.kubevirt.io/%s"

	// TemplateFlavorLabel is a label that specifies the flavor of the template
	TemplateFlavorLabel = "flavor.template.kubevirt.io/%s"

	// TemplateNameOsAnnotation is an annotation that specifies human readable os name
	TemplateNameOsAnnotation = "name.os.template.kubevirt.io/%s"

	processingURI = "processedTemplates"
	nameParameter = "NAME"
	otherValue    = "other"
)

// Templates is responsible for finding and processing templates
type Templates struct {
	Client *tempclient.TemplateV1Client
}

// TemplateProvider searches for and processes templates in Openshift
type TemplateProvider interface {
	Find(namespace *string, os *string, workload *string, flavor *string) (*templatev1.TemplateList, error)
	Process(namespace string, vmName *string, template *templatev1.Template) (*templatev1.Template, error)
}

// NewTemplateProvider creates new TemplateProvider
func NewTemplateProvider(client *tempclient.TemplateV1Client) *Templates {
	return &Templates{
		Client: client,
	}
}

// Find looks for a template based on given namespace and options
func (t *Templates) Find(namespace *string, os *string, workload *string, flavor *string) (*templatev1.TemplateList, error) {
	labelSelector := osLabelSelectorBuilder(os, workload, flavor)
	options := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	return t.Client.Templates(*namespace).List(context.TODO(), options)
}

// Process calls the openshift api to process parameters
func (t *Templates) Process(namespace string, vmName *string, template *templatev1.Template) (*templatev1.Template, error) {
	temp := template.DeepCopy()
	params := temp.Parameters
	for i, param := range params {
		if param.Name == nameParameter {
			if vmName != nil {
				temp.Parameters[i].Value = *vmName
			}
		} else {
			temp.Parameters[i].Value = otherValue
		}
	}
	result := &v1.Template{}
	err := t.Client.RESTClient().Post().
		Namespace(namespace).
		Resource(processingURI).
		Body(temp).
		Do(context.TODO()).
		Into(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// osLabelSelectorBuilder build the label selector based on template criteria
func osLabelSelectorBuilder(os *string, workload *string, flavor *string) string {
	labels := OSLabelBuilder(os, workload, flavor)
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

// OSLabelBuilder builds template labels based on template criteria
func OSLabelBuilder(os *string, workload *string, flavor *string) map[string]string {
	labels := make(map[string]string)
	if os != nil {
		labels[fmt.Sprintf(TemplateOsLabel, *os)] = "true"
	}
	if workload != nil {
		labels[fmt.Sprintf(TemplateWorkloadLabel, *workload)] = "true"
	}
	if flavor != nil {
		labels[fmt.Sprintf(TemplateFlavorLabel, *flavor)] = "true"
	}
	return labels
}
