package mocks

import (
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	dnsrecordv1alpha1 "github.com/dana-team/provider-dns/apis/record/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativev1beta1 "knative.dev/serving/pkg/apis/serving/v1beta1"
)

// CreateDomainMappingObject returns an empty DomainMapping object.
func CreateDomainMappingObject(name string) *knativev1beta1.DomainMapping {
	return &knativev1beta1.DomainMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NSName,
		},
	}
}

// CreateCertificateObject returns an empty DomainMapping object.
func CreateCertificateObject(name string) *cmapi.Certificate {
	return &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NSName,
		},
	}
}

// CreateDNSRecordObject returns an empty ARecordSet object.
func CreateDNSRecordObject(name string) *dnsrecordv1alpha1.CNAMERecord {
	return &dnsrecordv1alpha1.CNAMERecord{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
