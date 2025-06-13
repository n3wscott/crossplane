/*
Copyright 2023 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package revision

import (
	"context"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	pkgmetav1 "github.com/crossplane/crossplane/apis/pkg/meta/v1"
	v1 "github.com/crossplane/crossplane/apis/pkg/v1"
	"github.com/crossplane/crossplane/internal/initializer"
	"github.com/crossplane/crossplane/internal/xpkg"
)

const (
	errListFunctionRevisions         = "cannot list revisions"
	errParseFunctionRevisionEndpoint = "cannot parse function revision endpoint url"
)

// ExternalFunctionHooks performs runtime operations for function packages.
type ExternalFunctionHooks struct {
	client resource.ClientApplicator
}

// NewExternalFunctionHooks returns a new ExternalFunctionHooks.
func NewExternalFunctionHooks(client client.Client) *ExternalFunctionHooks {
	return &ExternalFunctionHooks{
		client: resource.ClientApplicator{
			Client:     client,
			Applicator: resource.NewAPIPatchingApplicator(client),
		},
	}
}

// Pre performs operations meant to happen before establishing objects.
func (h *ExternalFunctionHooks) Pre(ctx context.Context, pkg runtime.Object, pr v1.PackageRevisionWithRuntime, build ManifestBuilder) error {
	fo, _ := xpkg.TryConvert(pkg, &pkgmetav1.Function{})
	functionMeta, ok := fo.(*pkgmetav1.Function)
	if !ok {
		return errors.New(errNotFunction)
	}

	if pr.GetDesiredState() != v1.PackageRevisionActive {
		return nil
	}

	secServer := build.TLSServerSecret()
	if err := h.client.Apply(ctx, secServer); err != nil {
		return errors.Wrap(err, errApplyFunctionSecret)
	}

	revs := v1.FunctionRevisionList{}
	if err := h.client.List(ctx, &revs, client.MatchingLabels{v1.LabelParentPackage: functionMeta.Name}); err != nil {
		return errors.Wrap(err, errListFunctionRevisions)
	}

	var dnsNames []string
	dnsNames = append(dnsNames, functionMeta.Name) // TODO: there is a race condition here where the cert is needed to make the knative service but we can't make the cert until we know what the url will be for the knative service.
	for _, rev := range revs.Items {
		if rev.Status.Endpoint == "" {
			continue
		}
		endpoint, err := url.Parse(rev.Status.Endpoint)
		if err != nil {
			return errors.Wrap(err, errParseFunctionRevisionEndpoint)
		}
		dnsNames = append(dnsNames, endpoint.Host)
	}

	if err := initializer.NewTLSCertificateGenerator(secServer.Namespace, initializer.RootCACertSecretName,
		initializer.TLSCertificateGeneratorWithServerSecretName(secServer.GetName(), dnsNames),
		initializer.TLSCertificateGeneratorWithOwner([]metav1.OwnerReference{meta.AsController(meta.TypedReferenceTo(pr, pr.GetObjectKind().GroupVersionKind()))})).Run(ctx, h.client); err != nil {
		return errors.Wrapf(err, "cannot generate TLS certificates for %q", pr.GetLabels()[v1.LabelParentPackage])
	}

	return nil
}

// Post performs operations meant to happen after establishing objects.
func (h *ExternalFunctionHooks) Post(ctx context.Context, pkg runtime.Object, pr v1.PackageRevisionWithRuntime, build ManifestBuilder) error {
	return nil
}

// Deactivate performs operations meant to happen before deactivating a revision.
func (h *ExternalFunctionHooks) Deactivate(ctx context.Context, _ v1.PackageRevisionWithRuntime, build ManifestBuilder) error {
	return nil
}
