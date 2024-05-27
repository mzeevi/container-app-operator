package utils

import (
	certv1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	mock "github.com/dana-team/container-app-operator/test/e2e_tests/mocks"
	knativev1beta1 "knative.dev/serving/pkg/apis/serving/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	dnsv1alpha1 "sigs.k8s.io/external-dns/endpoint"
)

// CreateCappWithHTTPHostname creates a Capp with a Hostname.
func CreateCappWithHTTPHostname(k8sClient client.Client) (*cappv1alpha1.Capp, string) {
	httpsCapp := mock.CreateBaseCapp()
	routeHostname := GenerateRouteHostname()

	httpsCapp.Spec.RouteSpec.Hostname = routeHostname

	return CreateCapp(k8sClient, httpsCapp), routeHostname
}

// CreateHTTPSCapp creates a Capp with a Hostname, TLS Enabled and TLSSecret.
func CreateHTTPSCapp(k8sClient client.Client) (*cappv1alpha1.Capp, string, string) {
	httpsCapp := mock.CreateBaseCapp()
	routeHostname := GenerateRouteHostname()
	secretName := GenerateSecretName()

	httpsCapp.Spec.RouteSpec.Hostname = routeHostname
	httpsCapp.Spec.RouteSpec.TlsSecret = secretName
	httpsCapp.Spec.RouteSpec.TlsEnabled = true

	return CreateCapp(k8sClient, httpsCapp), routeHostname, secretName
}

// GetDomainMapping fetches and returns an existing instance of a DomainMapping.
func GetDomainMapping(k8sClient client.Client, name string, namespace string) *knativev1beta1.DomainMapping {
	domainMapping := &knativev1beta1.DomainMapping{}
	GetResource(k8sClient, domainMapping, name, namespace)
	return domainMapping
}

// GetDNSEndpoint fetches and returns an existing instance of a DNSEndpoint.
func GetDNSEndpoint(k8sClient client.Client, name string, namespace string) *dnsv1alpha1.DNSEndpoint {
	dnsEndpoint := &dnsv1alpha1.DNSEndpoint{}
	GetResource(k8sClient, dnsEndpoint, name, namespace)
	return dnsEndpoint
}

// GetCertificate fetches and returns an existing instance of a Certificate.
func GetCertificate(k8sClient client.Client, name string, namespace string) *certv1alpha1.Certificate {
	certificate := &certv1alpha1.Certificate{}
	GetResource(k8sClient, certificate, name, namespace)
	return certificate
}
