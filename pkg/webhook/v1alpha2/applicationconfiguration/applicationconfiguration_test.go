package applicationconfiguration

import (
	"context"
	"fmt"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	json "github.com/json-iterator/go"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestApplicationConfigurationValidation(t *testing.T) {
	handler := &ValidatingHandler{}

	var scheme = runtime.NewScheme()
	_ = core.AddToScheme(scheme)
	dec, _ := admission.NewDecoder(scheme)
	handler.InjectDecoder(dec)

	resource := metav1.GroupVersionResource{Group: "core.oam.dev", Version: "v1alpha2", Resource: "applicationconfigurations"}
	ctx := context.Background()
	namespace := "ns"
	workloadName := "NonEmptyWorkloadName"
	workloadNameFiledPath := "metadata.name"
	paramName := "AssignName"

	cwRaw, _ := json.Marshal(v1alpha2.ContainerizedWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Name: "",
		},
	})
	cwRawWithWorkloadName, _ := json.Marshal(v1alpha2.ContainerizedWorkload{
		ObjectMeta: metav1.ObjectMeta{
			Name: workloadName,
		},
	})
	mstRaw, _ := json.Marshal(v1alpha2.ManualScalerTrait{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManualScalerTrait",
			APIVersion: "core.oam.dev",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "",
		}})
	comp := v1alpha2.Component{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "",
		},
		Spec: v1alpha2.ComponentSpec{
			Workload: runtime.RawExtension{
				Raw: cwRaw,
			},
			Parameters: nil,
		},
	}

	cr := appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: namespace},
		Data: runtime.RawExtension{Object: &v1alpha2.Component{
			TypeMeta: metav1.TypeMeta{
				Kind:       "",
				APIVersion: "",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "r1",
				Namespace: namespace,
			},
			Spec: v1alpha2.ComponentSpec{
				Workload: runtime.RawExtension{
					Raw: cwRaw,
				},
			}}}}
	crWithWorkloadName := appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: namespace},
		Data: runtime.RawExtension{Object: &v1alpha2.Component{
			TypeMeta: metav1.TypeMeta{
				Kind:       "",
				APIVersion: "",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "r1",
				Namespace: namespace,
			},
			Spec: v1alpha2.ComponentSpec{
				Workload: runtime.RawExtension{
					Raw: cwRawWithWorkloadName,
				},
			}}}}
	crWithParam := appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: namespace},
		Data: runtime.RawExtension{Object: &v1alpha2.Component{
			TypeMeta: metav1.TypeMeta{
				Kind:       "",
				APIVersion: "",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "r1",
				Namespace: namespace,
			},
			Spec: v1alpha2.ComponentSpec{
				Workload: runtime.RawExtension{
					Raw: cwRaw,
				},
				Parameters: []v1alpha2.ComponentParameter{
					{
						Name:       paramName,
						FieldPaths: []string{workloadNameFiledPath},
					},
				},
			}}}}

	appConfigRevisionNameConflict, _ := json.Marshal(v1alpha2.ApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: v1alpha2.ApplicationConfigurationSpec{Components: []v1alpha2.ApplicationConfigurationComponent{
			{
				RevisionName:  "r1",
				ComponentName: "c1",
			},
		}}})
	appConfigCompName, _ := json.Marshal(v1alpha2.ApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: v1alpha2.ApplicationConfigurationSpec{Components: []v1alpha2.ApplicationConfigurationComponent{
			{
				ComponentName: "c1",
				Traits: []v1alpha2.ComponentTrait{
					{
						Trait: runtime.RawExtension{
							Raw: mstRaw,
						},
					},
				},
			},
		}}})
	appConfigRevisionName, _ := json.Marshal(v1alpha2.ApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: v1alpha2.ApplicationConfigurationSpec{Components: []v1alpha2.ApplicationConfigurationComponent{
			{
				RevisionName: "r1",
			},
		}}})
	appConfigWithParam, _ := json.Marshal(v1alpha2.ApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: v1alpha2.ApplicationConfigurationSpec{Components: []v1alpha2.ApplicationConfigurationComponent{
			{
				RevisionName: "r1",
				ParameterValues: []v1alpha2.ComponentParameterValue{
					{
						Name:  paramName,
						Value: intstr.FromString(workloadName),
					},
				},
			},
		}}})

	tests := []struct {
		caseName   string
		req        admission.Request
		mockGetFun test.MockGetFn
		mockCR     *appsv1.ControllerRevision
		pass       bool
		reason     string
	}{
		{
			caseName: "Test conflicts on revisionName and componentName",
			req: admission.Request{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Resource: resource,
					Object:   runtime.RawExtension{Raw: appConfigRevisionNameConflict},
				},
			},
			mockGetFun: func(_ context.Context, _ client.ObjectKey, obj runtime.Object) error {
				return nil
			},
			mockCR: &cr,
			pass:   false,
			reason: "componentName and revisionName are mutually exclusive, you can only specify one of them",
		},
		{
			caseName: "Test no conflicts on revisionName and componentName",
			req: admission.Request{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Resource: resource,
					Object:   runtime.RawExtension{Raw: appConfigRevisionName},
				},
			},
			mockCR: &cr,
			mockGetFun: func(_ context.Context, _ client.ObjectKey, obj runtime.Object) error {
				if o, ok := obj.(*appsv1.ControllerRevision); ok {
					*o = cr
					return nil
				}
				return nil
			},
			pass: true,
		},
		{
			caseName: "Test validation fails for workload name fixed in component",
			req: admission.Request{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Resource: resource,
					Object:   runtime.RawExtension{Raw: appConfigRevisionName},
				},
			},
			mockCR: &crWithWorkloadName,
			mockGetFun: func(_ context.Context, _ client.ObjectKey, obj runtime.Object) error {
				if o, ok := obj.(*appsv1.ControllerRevision); ok {
					*o = crWithWorkloadName
					return nil
				}
				return nil
			},
			pass:   false,
			reason: fmt.Sprintf(reasonFmtWorkloadNameNotEmpty, workloadName),
		},
		{
			caseName: "Test validation fails for workload name assigned by parameter",
			req: admission.Request{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Resource: resource,
					Object:   runtime.RawExtension{Raw: appConfigWithParam},
				},
			},
			mockCR: &crWithParam,
			mockGetFun: func(_ context.Context, _ client.ObjectKey, obj runtime.Object) error {
				if o, ok := obj.(*appsv1.ControllerRevision); ok {
					*o = crWithParam
					return nil
				}
				return nil
			},
			pass:   false,
			reason: fmt.Sprintf(reasonFmtWorkloadNameNotEmpty, workloadName),
		},
		{
			caseName: "Test validation workload name success",
			req: admission.Request{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Resource: resource,
					Object:   runtime.RawExtension{Raw: appConfigCompName},
				},
			},
			mockCR: &cr,
			mockGetFun: func(_ context.Context, _ client.ObjectKey, obj runtime.Object) error {
				if o, ok := obj.(*v1alpha2.TraitDefinition); ok {
					*o = v1alpha2.TraitDefinition{
						Spec: v1alpha2.TraitDefinitionSpec{
							RevisionEnabled: true,
						},
					}
				}
				if o, ok := obj.(*v1alpha2.Component); ok {
					*o = comp
				}
				return nil
			},
			pass: true,
		},
	}
	for _, tv := range tests {
		handler.Client = &test.MockClient{
			MockGet: tv.mockGetFun,
		}
		handler.AppsClient = fake.NewSimpleClientset().AppsV1()
		handler.AppsClient.ControllerRevisions(namespace).Create(ctx, tv.mockCR, metav1.CreateOptions{})
		resp := handler.Handle(ctx, tv.req)
		if tv.pass != resp.Allowed {
			t.Logf("Running Test Case: %v", tv.caseName)
			t.Errorf("expect %v but got %v from validation", tv.pass, resp.Allowed)
		}
		if tv.reason != "" {
			if tv.reason != string(resp.Result.Reason) {
				t.Errorf("\nvalidation should fail by reason: %v \ninstead of by reason: %v ", tv.reason, resp.Result.Reason)
			}
		}
	}
}
