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

package apiserver

import (
	"net"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"

	"github.com/openshift/autoheal/pkg/apis/autoheal/install"
	"github.com/openshift/autoheal/pkg/apis/autoheal/v1alpha2"
	"github.com/openshift/autoheal/pkg/apiserver/registry/healingrule"
	"github.com/openshift/autoheal/pkg/signals"
)

var (
	groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
	registry             = registered.NewOrDie("")
	scheme               = runtime.NewScheme()
	codecs               = serializer.NewCodecFactory(scheme)

	// Options for the API server:
	opts = options.NewRecommendedOptions("/registry", codecs.LegacyCodec(v1alpha2.SchemeGroupVersion))
)

var Cmd = &cobra.Command{
	Use:   "apiserver",
	Short: "Starts the auto-heal API server",
	Long:  "Starts the auto-heal API server.",
	Run:   run,
}

func init() {
	// Add our types:
	install.Install(groupFactoryRegistry, registry, scheme)

	// Add the V1 version:
	meta.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})

	// Add the default types:
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	scheme.AddUnversionedTypes(
		unversioned,
		&meta.Status{},
		&meta.APIVersions{},
		&meta.APIGroupList{},
		&meta.APIGroup{},
		&meta.APIResourceList{},
	)

	// Add to the set of command line flags the general and admision flags of the API server:
	flags := Cmd.Flags()
	opts.AddFlags(flags)
}

func run(cmd *cobra.Command, args []string) {
	var err error

	// Set up signals so we handle the first shutdown signal gracefully:
	stopCh := signals.SetupSignalHandler()

	err = opts.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")})
	if err != nil {
		glog.Fatalf("Cant set default secure serving options: %s", err.Error())
	}

	// Create a default configuration:
	config := server.NewRecommendedConfig(codecs)

	// Apply the general command line options to the configuration:
	err = opts.ApplyTo(config, scheme)
	if err != nil {
		glog.Fatalf("Can't apply general options: %s", err.Error())
	}

	// Create the server:
	srv, err := config.Complete().New("autoheal-apiserver", server.EmptyDelegate)
	if err != nil {
		glog.Fatalf("Can't create server: %s", err.Error())
	}

	// Install the versions of the API objects in the server:
	groupInfo := server.NewDefaultAPIGroupInfo(v1alpha2.GroupName, registry, scheme, meta.ParameterCodec, codecs)
	groupInfo.GroupMeta.GroupVersion = v1alpha2.SchemeGroupVersion
	groupStorage := map[string]rest.Storage{
		"healingrules": checkStorage(healingrule.NewStore(scheme, config.RESTOptionsGetter)),
	}
	groupInfo.VersionedResourcesStorageMap["v1alpha2"] = groupStorage
	err = srv.InstallAPIGroup(&groupInfo)
	if err != nil {
		glog.Fatalf("Can't install api versions: %s", err.Error())
	}

	// Run the server:
	srv.PrepareRun().Run(stopCh)
}

func checkStorage(storage rest.StandardStorage, err error) rest.StandardStorage {
	if err != nil {
		glog.Fatal("Can't create REST storage: %s", err)
	}
	return storage
}
