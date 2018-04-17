/*
Copyright (c) 2018 Red Hat, Inc.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha2 "github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeHealingRules implements HealingRuleInterface
type FakeHealingRules struct {
	Fake *FakeAutohealV1alpha2
	ns   string
}

var healingrulesResource = schema.GroupVersionResource{Group: "autoheal.openshift.io", Version: "v1alpha2", Resource: "healingrules"}

var healingrulesKind = schema.GroupVersionKind{Group: "autoheal.openshift.io", Version: "v1alpha2", Kind: "HealingRule"}

// Get takes name of the healingRule, and returns the corresponding healingRule object, and an error if there is any.
func (c *FakeHealingRules) Get(name string, options v1.GetOptions) (result *v1alpha2.HealingRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(healingrulesResource, c.ns, name), &v1alpha2.HealingRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealingRule), err
}

// List takes label and field selectors, and returns the list of HealingRules that match those selectors.
func (c *FakeHealingRules) List(opts v1.ListOptions) (result *v1alpha2.HealingRuleList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(healingrulesResource, healingrulesKind, c.ns, opts), &v1alpha2.HealingRuleList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.HealingRuleList{}
	for _, item := range obj.(*v1alpha2.HealingRuleList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested healingRules.
func (c *FakeHealingRules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(healingrulesResource, c.ns, opts))

}

// Create takes the representation of a healingRule and creates it.  Returns the server's representation of the healingRule, and an error, if there is any.
func (c *FakeHealingRules) Create(healingRule *v1alpha2.HealingRule) (result *v1alpha2.HealingRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(healingrulesResource, c.ns, healingRule), &v1alpha2.HealingRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealingRule), err
}

// Update takes the representation of a healingRule and updates it. Returns the server's representation of the healingRule, and an error, if there is any.
func (c *FakeHealingRules) Update(healingRule *v1alpha2.HealingRule) (result *v1alpha2.HealingRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(healingrulesResource, c.ns, healingRule), &v1alpha2.HealingRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealingRule), err
}

// Delete takes name of the healingRule and deletes it. Returns an error if one occurs.
func (c *FakeHealingRules) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(healingrulesResource, c.ns, name), &v1alpha2.HealingRule{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeHealingRules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(healingrulesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha2.HealingRuleList{})
	return err
}

// Patch applies the patch and returns the patched healingRule.
func (c *FakeHealingRules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha2.HealingRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(healingrulesResource, c.ns, name, data, subresources...), &v1alpha2.HealingRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.HealingRule), err
}
