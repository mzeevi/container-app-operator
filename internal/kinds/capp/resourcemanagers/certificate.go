package resourcemanagers

import (
	"context"
	"fmt"
	"reflect"

	certv1alpha1 "github.com/dana-team/certificate-operator/api/v1alpha1"
	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	rclient "github.com/dana-team/container-app-operator/internal/kinds/capp/resourceclient"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Certificate                        = "certificate"
	eventCappCertificateCreationFailed = "CertificateCreationFailed"
	eventCappCertificateCreated        = "CertificateCreated"
	certificateForm                    = "pfx"
	certificateConfig                  = "certificateconfig-capp"
)

type CertificateManager struct {
	Ctx           context.Context
	K8sclient     client.Client
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// prepareResource prepares a Certificate resource based on the provided Capp.
func (c CertificateManager) prepareResource(capp cappv1alpha1.Capp) certv1alpha1.Certificate {
	return certv1alpha1.Certificate{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      capp.Spec.RouteSpec.Hostname,
			Namespace: capp.Namespace,
			Labels: map[string]string{
				CappResourceKey: capp.Name,
			},
		},
		Spec: certv1alpha1.CertificateSpec{
			CertificateData: certv1alpha1.CertificateData{
				Subject: certv1alpha1.Subject{
					CommonName: capp.Spec.RouteSpec.Hostname,
				},
				San: certv1alpha1.San{
					DNS: []string{capp.Spec.RouteSpec.Hostname},
				},
				Form: certificateForm,
			},
			SecretName: capp.Spec.RouteSpec.TlsSecret,
			ConfigRef: certv1alpha1.ConfigReference{
				Name: certificateConfig,
			},
		},
	}
}

// CleanUp attempts to delete the associated Certificate for a given Capp resource.
func (c CertificateManager) CleanUp(capp cappv1alpha1.Capp) error {
	resourceManager := rclient.ResourceManagerClient{Ctx: c.Ctx, K8sclient: c.K8sclient, Log: c.Log}

	if capp.Status.RouteStatus.DomainMappingObjectStatus.URL != nil {
		certificate := rclient.PrepareCertificate(capp.Status.RouteStatus.DomainMappingObjectStatus.URL.Host, capp.Namespace)
		if err := resourceManager.DeleteResource(&certificate); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}
	return nil
}

// IsRequired is responsible to determine if resource Certificate is required.
func (c CertificateManager) IsRequired(capp cappv1alpha1.Capp) bool {
	return capp.Spec.RouteSpec.Hostname != "" && capp.Spec.RouteSpec.TlsEnabled
}

// Manage creates or updates a Certificate resource based on the provided Capp if it's required.
// If it's not, then it cleans up the resource if it exists.
func (c CertificateManager) Manage(capp cappv1alpha1.Capp) error {
	if c.IsRequired(capp) {
		return c.createOrUpdate(capp)
	}

	return c.CleanUp(capp)
}

// createOrUpdate creates or updates a Certificate resource.
func (c CertificateManager) createOrUpdate(capp cappv1alpha1.Capp) error {
	certificateFromCapp := c.prepareResource(capp)
	certificate := certv1alpha1.Certificate{}
	resourceManager := rclient.ResourceManagerClient{Ctx: c.Ctx, K8sclient: c.K8sclient, Log: c.Log}

	if err := c.deletePreviousCertificates(capp, resourceManager); err != nil {
		return fmt.Errorf("failed to delete previous Certificates: %w", err)
	}

	if err := c.K8sclient.Get(c.Ctx, types.NamespacedName{Namespace: capp.Namespace, Name: certificateFromCapp.Name}, &certificate); err != nil {
		if errors.IsNotFound(err) {
			return c.createCertificate(capp, certificateFromCapp, resourceManager)
		} else {
			return fmt.Errorf("failed to get Certificate %q: %w", certificateFromCapp.Name, err)
		}
	}

	return c.updateCertificate(certificate, certificateFromCapp, resourceManager)
}

// createCertificate creates a new Certificate and emits an event.
func (c CertificateManager) createCertificate(capp cappv1alpha1.Capp, certificateFromCapp certv1alpha1.Certificate, resourceManager rclient.ResourceManagerClient) error {
	if err := resourceManager.CreateResource(&certificateFromCapp); err != nil {
		c.EventRecorder.Event(&capp, corev1.EventTypeWarning, eventCappCertificateCreationFailed,
			fmt.Sprintf("Failed to create Certificate %s", certificateFromCapp.Name))

		return err
	}

	c.EventRecorder.Event(&capp, corev1.EventTypeNormal, eventCappCertificateCreated,
		fmt.Sprintf("Created Certificate %s", certificateFromCapp.Name))

	return nil
}

// updateCertificate checks if an update to the Certificate is necessary and performs the update to match desired state.
func (c CertificateManager) updateCertificate(certificate, certificateFromCapp certv1alpha1.Certificate, resourceManager rclient.ResourceManagerClient) error {
	if !reflect.DeepEqual(certificate.Spec, certificateFromCapp.Spec) {
		certificate.Spec = certificateFromCapp.Spec
		return resourceManager.UpdateResource(&certificate)
	}

	return nil
}

// deletePreviousCertificates deletes all previous Certificates associated with a Capp.
func (c CertificateManager) deletePreviousCertificates(capp cappv1alpha1.Capp, resourceManager rclient.ResourceManagerClient) error {
	requirement, err := labels.NewRequirement(CappResourceKey, selection.Equals, []string{capp.Name})
	if err != nil {
		return fmt.Errorf("unable to create label requirement for Capp: %w", err)
	}

	labelSelector := labels.NewSelector().Add(*requirement)
	listOptions := client.ListOptions{
		LabelSelector: labelSelector,
	}

	certificates := certv1alpha1.CertificateList{}
	if err := c.K8sclient.List(c.Ctx, &certificates, &listOptions); err != nil {
		return fmt.Errorf("unable to list Certificates of Capp %q: %w", capp.Name, err)
	}

	for _, certificate := range certificates.Items {
		if certificate.Name != capp.Spec.RouteSpec.Hostname {
			cert := rclient.PrepareCertificate(certificate.Name, certificate.Namespace)
			if err := resourceManager.DeleteResource(&cert); err != nil {
				return err
			}
		}
	}
	return nil
}
