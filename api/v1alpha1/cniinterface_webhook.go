/*


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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const ACLGROUP = "icn:nfc:sdewan"

// log is for logging in this package.
var cniinterfacelog = logf.Log.WithName("cniinterface-resource")

func SetupUserACLWebhookWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register(
		"/validate-batch-sdewan-akraino-org-v1alpha1-cniinterface",
		&webhook.Admission{Handler: &userACLValidator{Client: mgr.GetClient()}})
	return nil
}

func SetupMUserACLWebhookWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register(
		"/mutate-batch-sdewan-akraino-org-v1alpha1-cniinterface",
		&webhook.Admission{Handler: &userACLMutate{Client: mgr.GetClient()}})
	return nil
}

func (r *CniInterface) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// userACLValidator validates
type userACLValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// userACLValidator admits the user is in system:cnfoperator.
func (v *userACLValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Kind.Group != "batch.sdewan.akraino.org" {
		return admission.Errored(
			http.StatusBadRequest,
			errors.New("The group is not batch.sdewan.akraino.org"))
	}
	aclgroup := false
	for _, g := range req.UserInfo.Groups {
		// TODO hardcode
		if g == ACLGROUP {
			aclgroup = true
			break
		}
	}

	cniinterfacelog.Info(fmt.Sprintf("Request User Info: %v", req.UserInfo))
	cniinterfacelog.Info(fmt.Sprintf("aclgroup: %v", aclgroup))

	obj := &CniInterface{}

	if req.Operation == "CREATE" || req.Operation == "UPDATE" || req.Operation == "DELETE" {
		if !aclgroup {
			return admission.Denied(fmt.Sprintf("User is not in acl group %s. Operation type: %s is not allowed", ACLGROUP, req.Operation))
		}
		err := v.decoder.Decode(req, obj)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, errors.New("Uknow resource type."))
		}
		return admission.Denied(fmt.Sprintf("We don't support operation type: %s", req.Operation))
	}
	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (v *userACLValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// userACLMutate  mutate
type userACLMutate struct {
	Client  client.Client
	decoder *admission.Decoder
}

// userACLMutate admits
func (v *userACLMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (v *userACLMutate) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-batch-sdewan-akraino-org-v1alpha1-cniinterface,mutating=true,failurePolicy=fail,groups=batch.sdewan.akraino.org,resources=cniinterfaces,verbs=create;update,versions=v1alpha1,name=mcniinterface.kb.io

var _ webhook.Defaulter = &CniInterface{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CniInterface) Default() {
	cniinterfacelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-batch-sdewan-akraino-org-v1alpha1-cniinterface,mutating=false,failurePolicy=fail,groups=batch.sdewan.akraino.org,resources=cniinterfaces,versions=v1alpha1,name=vcniinterface.kb.io

var _ webhook.Validator = &CniInterface{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CniInterface) ValidateCreate() error {
	cniinterfacelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CniInterface) ValidateUpdate(old runtime.Object) error {
	cniinterfacelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CniInterface) ValidateDelete() error {
	cniinterfacelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
