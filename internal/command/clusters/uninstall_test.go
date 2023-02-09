package clusters

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclock "k8s.io/utils/clock/testing"
	"k8s.io/utils/pointer"

	"github.com/jetstack/jsctl/internal/kubernetes/clients"
)

func Test_findIssues(t *testing.T) {
	fakeClock := fakeclock.FakeClock{}
	fooPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "foo",
				},
			},
		},
	}
	fooSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}
	readyCert := cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       cmapi.CertificateKind,
			APIVersion: "v1",
		},
		Status: cmapi.CertificateStatus{
			Conditions: []cmapi.CertificateCondition{
				{
					Type:   cmapi.CertificateConditionReady,
					Status: cmmeta.ConditionTrue,
				},
			},
		},
	}
	tests := map[string]struct {
		want            []notification
		wantErr         bool
		podList         *corev1.PodList
		secretList      *corev1.SecretList
		certificateList *cmapi.CertificateList
	}{
		"cluster that has no cert-manager related resources does not produce any notifications": {
			podList:         &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList:      &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: nil},
			want:            []notification{},
		},
		"cluster that has some ready certs should not produce any notifications": {
			podList:         &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList:      &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{readyCert}},
			want:            []notification{},
		},
		"cluster that has a cert with false ready condition should produce a notification": {
			podList:    &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       cmapi.CertificateKind,
					APIVersion: "v1",
				},
				Status: cmapi.CertificateStatus{
					Conditions: []cmapi.CertificateCondition{{
						Type:   cmapi.CertificateConditionReady,
						Status: cmmeta.ConditionFalse,
					},
					},
				},
			}}},
			want: []notification{{
				header:        unreadyHeader,
				resourceInfos: []string{fmt.Sprintf(unreadyInfoTemplate, "foo", "foo")},
			}},
		},
		"cluster that has a cert without a ready condition should produce a notification": {
			podList:    &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       cmapi.CertificateKind,
					APIVersion: "v1",
				},
				Status: cmapi.CertificateStatus{},
			}}},
			want: []notification{{
				header:        unreadyHeader,
				resourceInfos: []string{fmt.Sprintf(unreadyInfoTemplate, "foo", "foo")},
			}},
		},
		"cluster that has a cert that is currently being issued should produce a notification": {
			podList:    &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       cmapi.CertificateKind,
					APIVersion: "v1",
				},
				Status: cmapi.CertificateStatus{
					Conditions: []cmapi.CertificateCondition{{
						Type:   cmapi.CertificateConditionIssuing,
						Status: cmmeta.ConditionTrue,
					},
						{
							Type:   cmapi.CertificateConditionReady,
							Status: cmmeta.ConditionTrue,
						},
					},
				},
			}}},
			want: []notification{{
				header:        currentIssuancesHeader,
				resourceInfos: []string{fmt.Sprintf(currentIssuancesInfoTemplate, "foo", "foo")},
			}},
		},
		"cluster that has a cert that failed issuance for latest renewal cycle should produce a notification": {
			podList:    &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       cmapi.CertificateKind,
					APIVersion: "v1",
				},
				Status: cmapi.CertificateStatus{
					FailedIssuanceAttempts: pointer.Int(2),
					Conditions: []cmapi.CertificateCondition{{
						Type:   cmapi.CertificateConditionIssuing,
						Status: cmmeta.ConditionFalse,
					},
						{
							Type:   cmapi.CertificateConditionReady,
							Status: cmmeta.ConditionTrue,
						},
					},
				},
			}}},
			want: []notification{{
				header:        failedInfoHeader,
				resourceInfos: []string{fmt.Sprintf(failedInfoTemplate, "foo", "foo", 2)},
			}},
		},
		"cluster that has a cert that is about to be renewed should produce a notification": {
			podList:    &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       cmapi.CertificateKind,
					APIVersion: "v1",
				},
				Status: cmapi.CertificateStatus{
					RenewalTime: &metav1.Time{Time: fakeClock.Now().Add(time.Minute)},
					Conditions: []cmapi.CertificateCondition{
						{
							Type:   cmapi.CertificateConditionReady,
							Status: cmmeta.ConditionTrue,
						},
					},
				},
			}}},
			want: []notification{{
				header:        upcomingRenewalInfoHeader,
				resourceInfos: []string{fmt.Sprintf(upcomingRenewalInfoTemplate, "foo", "foo", fakeClock.Now().Add(time.Minute))},
			}},
		},
		"cluster that has a cert that is about to expire should produce a notification": {
			podList:    &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{fooSecret}},
			certificateList: &cmapi.CertificateList{Items: []cmapi.Certificate{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       cmapi.CertificateKind,
					APIVersion: "v1",
				},
				Status: cmapi.CertificateStatus{
					NotAfter: &metav1.Time{Time: fakeClock.Now().Add(time.Minute)},
					Conditions: []cmapi.CertificateCondition{
						{
							Type:   cmapi.CertificateConditionReady,
							Status: cmmeta.ConditionTrue,
						},
					},
				},
			}}},
			want: []notification{{
				header:        upcomingExpiriesHeader,
				resourceInfos: []string{fmt.Sprintf(upcomingExpiriesInfoTemplate, "foo", "foo", fakeClock.Now().Add(time.Minute))},
			}},
		},
		"cluster that has an issued cert that would get garbage collected if cert-manager is uninstalled should produce a notification": {
			podList: &corev1.PodList{Items: []corev1.Pod{fooPod}},
			secretList: &corev1.SecretList{Items: []corev1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "Certificate",
					}},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
			}}},
			certificateList: &cmapi.CertificateList{Items: nil},
			want: []notification{{
				header:        hasOwnerRefHeader,
				resourceInfos: []string{fmt.Sprintf(hasOwnerRefInfoTemplate, "foo", "foo")},
			}},
		},
		"cluster that appears to have cert-manager-csi-driver installed should produce a warning": {
			podList: &corev1.PodList{Items: []corev1.Pod{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "quay.io/cert-manager-csi-driver:v0.0.5",
						}},
					},
				},
			}},
			secretList:      &corev1.SecretList{Items: nil},
			certificateList: &cmapi.CertificateList{Items: nil},
			want: []notification{{
				header:        integrationHeader,
				resourceInfos: []string{fmt.Sprintf(integrationInfoTemplate, "cert-manager-csi-driver")},
			}},
		},
		"cluster that appears to have cert-manager-csi-driver-spiffe installed should produce a warning": {
			podList: &corev1.PodList{Items: []corev1.Pod{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "quay.io/cert-manager-csi-driver-spiffe:v0.0.5",
						}},
					},
				},
			}},
			secretList:      &corev1.SecretList{Items: nil},
			certificateList: &cmapi.CertificateList{Items: nil},
			want: []notification{{
				header:        integrationHeader,
				resourceInfos: []string{fmt.Sprintf(integrationInfoTemplate, "cert-manager-csi-driver-spiffe")},
			}},
		},
		"cluster that appears to have cert-manager-istio-csr installed should produce a warning": {
			podList: &corev1.PodList{Items: []corev1.Pod{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "quay.io/cert-manager-istio-csr:v0.0.5",
						}},
					},
				},
			}},
			secretList:      &corev1.SecretList{Items: nil},
			certificateList: &cmapi.CertificateList{Items: nil},
			want: []notification{{
				header:        integrationHeader,
				resourceInfos: []string{fmt.Sprintf(integrationInfoTemplate, "cert-manager-istio-csr")},
			}},
		},
	}
	for name, scenario := range tests {
		t.Run(name, func(t *testing.T) {
			clientset := allClients{
				secrets: &clients.FakeGeneric[*corev1.Secret, *corev1.SecretList]{
					FakeList: func(_ context.Context, _ *clients.GenericRequestOptions, result *corev1.SecretList) error {
						result.Items = scenario.secretList.Items
						return nil
					},
				},
				pods: &clients.FakeGeneric[*corev1.Pod, *corev1.PodList]{
					FakeList: func(_ context.Context, _ *clients.GenericRequestOptions, result *corev1.PodList) error {
						result.Items = scenario.podList.Items
						return nil
					},
				},
				certificates: &clients.FakeGeneric[*cmapi.Certificate, *cmapi.CertificateList]{
					FakeList: func(_ context.Context, _ *clients.GenericRequestOptions, result *cmapi.CertificateList) error {
						result.Items = scenario.certificateList.Items
						return nil
					},
				},
			}
			got, err := findIssues(context.Background(), clientset, &fakeClock)
			if (err != nil) != scenario.wantErr {
				t.Errorf("findIssues() error = %v, wantErr %v", err, scenario.wantErr)
				return
			}
			if !reflect.DeepEqual(got, scenario.want) {
				t.Errorf("findIssues() = %v, want %v", got, scenario.want)
			}
		})
	}
}
