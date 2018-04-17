/*
Copyright 2018 Red Hat, Inc.

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

package healingrule

import (
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	"github.com/openshift/autoheal/pkg/apis/autoheal"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func NewStrategy(typer runtime.ObjectTyper) healingRuleStrategy {
	return healingRuleStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	apiserver, ok := obj.(*autoheal.HealingRule)
	if !ok {
		return nil, nil, false, fmt.Errorf("Given object is not a healing rule")
	}
	return labels.Set(apiserver.ObjectMeta.Labels), healingRuleToSelectableFields(apiserver), apiserver.Initializers != nil, nil
}

// MatchHealingRuile is the filter used by the generic etcd backend to watch events from etcd to
// clients of the apiserver only interested in specific labels/fields.
//
func MatchHealingRule(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// healingRuleToSelectableFields returns a field set that represents the object.
//
func healingRuleToSelectableFields(obj *autoheal.HealingRule) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}

type healingRuleStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (healingRuleStrategy) NamespaceScoped() bool {
	return false
}

func (healingRuleStrategy) PrepareForCreate(ctx request.Context, obj runtime.Object) {
}

func (healingRuleStrategy) PrepareForUpdate(ctx request.Context, obj, old runtime.Object) {
}

func (healingRuleStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (healingRuleStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (healingRuleStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (healingRuleStrategy) Canonicalize(obj runtime.Object) {
}

func (healingRuleStrategy) ValidateUpdate(ctx request.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
