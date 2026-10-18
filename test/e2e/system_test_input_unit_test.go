package e2e

import (
	"reflect"
	"testing"
	"time"

	dpuservicev1 "github.com/nvidia/doca-platform/api/dpuservice/v1alpha1"
	dpfe2e "github.com/nvidia/doca-platform/test/e2e"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreateTemplateFromLiveObjectClearsServerMetadata(t *testing.T) {
	deletionTimestamp := metav1.NewTime(time.Unix(1_700_000_000, 0))
	deletionGracePeriodSeconds := int64(30)
	live := &dpuservicev1.DPUServiceIPAM{
		TypeMeta: metav1.TypeMeta{
			APIVersion: dpuservicev1.DPUServiceIPAMGroupVersionKind.GroupVersion().String(),
			Kind:       dpuservicev1.DPUServiceIPAMGroupVersionKind.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "pool1",
			Namespace:                  "dpf-operator-system",
			Labels:                     map[string]string{"user-label": "preserved"},
			Annotations:                map[string]string{"user-annotation": "preserved"},
			ResourceVersion:            "12345",
			UID:                        types.UID("server-assigned-uid"),
			Generation:                 7,
			CreationTimestamp:          metav1.NewTime(time.Unix(1_600_000_000, 0)),
			DeletionTimestamp:          &deletionTimestamp,
			DeletionGracePeriodSeconds: &deletionGracePeriodSeconds,
			ManagedFields:              []metav1.ManagedFieldsEntry{{Manager: "dpu-controller"}},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "svc.dpu.nvidia.com/v1alpha1",
				Kind:       "DPUDeployment",
				Name:       "dpudeployment",
				UID:        types.UID("owner-uid"),
			}},
			Finalizers: []string{"svc.dpu.nvidia.com/finalizer"},
		},
	}

	template := createTemplateFromLiveObject(live)

	if template == live {
		t.Fatal("create template aliases the live object")
	}
	if template.ResourceVersion != "" || template.UID != "" || template.Generation != 0 {
		t.Fatalf("server identity was retained: resourceVersion=%q uid=%q generation=%d",
			template.ResourceVersion, template.UID, template.Generation)
	}
	if !template.CreationTimestamp.IsZero() || template.DeletionTimestamp != nil || template.DeletionGracePeriodSeconds != nil {
		t.Fatalf("server lifecycle metadata was retained: creation=%v deletion=%v grace=%v",
			template.CreationTimestamp, template.DeletionTimestamp, template.DeletionGracePeriodSeconds)
	}
	if template.ManagedFields != nil || template.OwnerReferences != nil || template.Finalizers != nil {
		t.Fatalf("server/controller metadata was retained: managedFields=%v ownerReferences=%v finalizers=%v",
			template.ManagedFields, template.OwnerReferences, template.Finalizers)
	}
	if !reflect.DeepEqual(template.Labels, live.Labels) || !reflect.DeepEqual(template.Annotations, live.Annotations) {
		t.Fatalf("desired metadata changed: labels=%v annotations=%v", template.Labels, template.Annotations)
	}
	if template.APIVersion != live.APIVersion || template.Kind != live.Kind || !reflect.DeepEqual(template.Spec, live.Spec) {
		t.Fatal("the desired object shape changed while removing server metadata")
	}
	if live.ResourceVersion != "12345" || live.UID != types.UID("server-assigned-uid") {
		t.Fatal("sanitizing the template mutated the live source object")
	}
}

func TestAssignDPUServiceIPAMTemplatesClassifiesPoolType(t *testing.T) {
	ipams := []dpuservicev1.DPUServiceIPAM{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "loopback", ResourceVersion: "network-rv"},
			Spec: dpuservicev1.DPUServiceIPAMSpec{
				IPV4Network: &dpuservicev1.IPV4Network{Network: "11.0.0.0/24"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "subnet", ResourceVersion: "subnet-rv"},
			Spec: dpuservicev1.DPUServiceIPAMSpec{
				IPV4Subnet: &dpuservicev1.IPV4Subnet{Subnet: "192.168.0.0/20"},
			},
		},
	}
	input := &dpfe2e.SystemTestInput{}

	assignDPUServiceIPAMTemplates(input, ipams)

	if input.CIDRDPUServiceIPAM == nil || input.CIDRDPUServiceIPAM.Name != "loopback" {
		t.Fatalf("network-mode IPAM was not classified as the CIDR template: %#v", input.CIDRDPUServiceIPAM)
	}
	if input.IPPoolDPUServiceIPAM == nil || input.IPPoolDPUServiceIPAM.Name != "subnet" {
		t.Fatalf("subnet-mode IPAM was not classified as the IPPool template: %#v", input.IPPoolDPUServiceIPAM)
	}
	if input.CIDRDPUServiceIPAM.ResourceVersion != "" || input.IPPoolDPUServiceIPAM.ResourceVersion != "" {
		t.Fatal("classified IPAM templates retained live resource versions")
	}
	if ipams[0].ResourceVersion != "network-rv" || ipams[1].ResourceVersion != "subnet-rv" {
		t.Fatal("classifying IPAM templates mutated the source list")
	}
}
