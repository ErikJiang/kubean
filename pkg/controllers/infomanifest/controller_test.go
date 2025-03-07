// Copyright 2023 Authors of kubean-io
// SPDX-License-Identifier: Apache-2.0

package infomanifest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	localartifactsetv1alpha1 "github.com/kubean-io/kubean-api/apis/localartifactset/v1alpha1"
	manifestv1alpha1 "github.com/kubean-io/kubean-api/apis/manifest/v1alpha1"
	kubeanclientsetfake "github.com/kubean-io/kubean-api/client/clientset/versioned/fake"
	"github.com/kubean-io/kubean-api/constants"
	"github.com/kubean-io/kubean/pkg/util"
)

func newFakeClient() client.Client {
	sch := scheme.Scheme
	if err := manifestv1alpha1.AddToScheme(sch); err != nil {
		panic(err)
	}
	if err := localartifactsetv1alpha1.AddToScheme(sch); err != nil {
		panic(err)
	}
	client := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(&manifestv1alpha1.Manifest{}).WithRuntimeObjects(&localartifactsetv1alpha1.LocalArtifactSet{}).Build()
	return client
}

func fetchTestingFake(obj interface{ RESTClient() rest.Interface }) *k8stesting.Fake {
	// https://stackoverflow.com/questions/69740891/mocking-errors-with-client-go-fake-client
	return reflect.Indirect(reflect.ValueOf(obj)).FieldByName("Fake").Interface().(*k8stesting.Fake)
}

func removeReactorFromTestingTake(obj interface{ RESTClient() rest.Interface }, verb, resource string) {
	if fakeObj := fetchTestingFake(obj); fakeObj != nil {
		newReactionChain := make([]k8stesting.Reactor, 0)
		fakeObj.Lock()
		defer fakeObj.Unlock()
		for i := range fakeObj.ReactionChain {
			reaction := fakeObj.ReactionChain[i]
			if simpleReaction, ok := reaction.(*k8stesting.SimpleReactor); ok && simpleReaction.Verb == verb && simpleReaction.Resource == resource {
				continue // ignore
			}
			newReactionChain = append(newReactionChain, reaction)
		}
		fakeObj.ReactionChain = newReactionChain
	}
}

func Test_ParseConfigMapToLocalService(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}
	localServiceData := `
      imageRepo: 
        kubeImageRepo: "temp-registry.daocloud.io:5000/registry.k8s.io"
        gcrImageRepo: "temp-registry.daocloud.io:5000/gcr.io"
        githubImageRepo: "a"
        dockerImageRepo: "b"
        quayImageRepo: "c"
      imageRepoAuth:
        - imageRepoAddress: temp-registry.daocloud.io:5000
          userName: admin
          passwordBase64: SGFyYm9yMTIzNDUK
      filesRepo: 'http://temp-registry.daocloud.io:9000'
      yumRepos:
        aRepo: 
          - 'aaa1'
          - 'aaa2'
        bRepo: 
          - 'bbb1'
          - 'bbb2'
      hostsMap:
        - domain: temp-registry.daocloud.io
          address: 'a.b.c.d'
`
	tests := []struct {
		name string
		arg  *corev1.ConfigMap
		want *manifestv1alpha1.LocalService
	}{
		{
			name: "zero data",
			arg:  &corev1.ConfigMap{},
			want: &manifestv1alpha1.LocalService{},
		},
		{
			name: "empty string",
			arg:  &corev1.ConfigMap{Data: map[string]string{"localService": ""}},
			want: &manifestv1alpha1.LocalService{},
		},
		{
			name: "good string data",
			arg:  &corev1.ConfigMap{Data: map[string]string{"localService": localServiceData}},
			want: &manifestv1alpha1.LocalService{
				ImageRepo: map[manifestv1alpha1.ImageRepoType]string{
					"kubeImageRepo":   "temp-registry.daocloud.io:5000/registry.k8s.io",
					"gcrImageRepo":    "temp-registry.daocloud.io:5000/gcr.io",
					"githubImageRepo": "a",
					"dockerImageRepo": "b",
					"quayImageRepo":   "c",
				},
				ImageRepoAuth: []manifestv1alpha1.ImageRepoPasswordAuth{
					{
						ImageRepoAddress: "temp-registry.daocloud.io:5000",
						UserName:         "admin",
						PasswordBase64:   "SGFyYm9yMTIzNDUK",
					},
				},
				FilesRepo: "http://temp-registry.daocloud.io:9000",
				YumRepos: map[string][]string{
					"aRepo": {"aaa1", "aaa2"},
					"bRepo": {"bbb1", "bbb2"},
				},
				HostsMap: []*manifestv1alpha1.HostsMap{
					{Domain: "temp-registry.daocloud.io", Address: "a.b.c.d"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, _ := controller.ParseConfigMapToLocalService(test.arg)
			if !reflect.DeepEqual(result, test.want) {
				t.Fatal()
			}
		})
	}
}

func Test_Test_UpdateLocalAvailableImage2(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}

	manifestName := "manifest"

	manifest := &manifestv1alpha1.Manifest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Manifest",
			APIVersion: "kubean.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: manifestName,
		},
		Spec: manifestv1alpha1.Spec{
			KubeanVersion: "123",
		},
	}
	controller.Client.Create(context.Background(), manifest)
	controller.KubeanClientSet.ManifestV1alpha1().Manifests().Create(context.Background(), manifest, metav1.CreateOptions{})
	// controller.InfoManifestClientSet.KubeanV1alpha1().Manifests().Create(context.Background(), manifest, metav1.CreateOptions{})

	tests := []struct {
		name string
		args func() string
		want string
	}{
		{
			name: "FetchGlobalInfoManifest with error",
			args: func() string {
				fetchTestingFake(controller.KubeanClientSet.ManifestV1alpha1()).PrependReactor("get", "manifests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					// fetchTestingFake(controller.InfoManifestClientSet.KubeanV1alpha1()).PrependReactor("get", "manifests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("this is error")
				})
				controller.UpdateLocalAvailableImage([]manifestv1alpha1.Manifest{*manifest})
				removeReactorFromTestingTake(controller.KubeanClientSet.ManifestV1alpha1(), "get", "manifests")
				// removeReactorFromTestingTake(controller.InfoManifestClientSet.KubeanV1alpha1(), "get", "manifests")

				manifest := &manifestv1alpha1.Manifest{}
				err := controller.Client.Get(context.Background(), types.NamespacedName{Name: manifestName}, manifest)
				if err != nil {
					t.Error(err)
				}
				return manifest.Status.LocalAvailable.KubesprayImage
			},
			want: "",
		},
		{
			name: "UpdateStatus with error",
			args: func() string {
				fetchTestingFake(controller.KubeanClientSet.ManifestV1alpha1()).PrependReactor("update", "manifests/status", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					// fetchTestingFake(controller.InfoManifestClientSet.KubeanV1alpha1()).PrependReactor("update", "manifests/status", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("this is error when updateStatus")
				})
				controller.UpdateLocalAvailableImage([]manifestv1alpha1.Manifest{*manifest})
				removeReactorFromTestingTake(controller.KubeanClientSet.ManifestV1alpha1(), "update", "manifests/status")
				// removeReactorFromTestingTake(controller.InfoManifestClientSet.KubeanV1alpha1(), "update", "manifests/status")
				manifest := &manifestv1alpha1.Manifest{}
				err := controller.Client.Get(context.Background(), types.NamespacedName{Name: manifestName}, manifest)
				if err != nil {
					t.Error(err)
				}
				return manifest.Status.LocalAvailable.KubesprayImage
			},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.args() != test.want {
				t.Fatal()
			}
		})
	}
}

func Test_UpdateLocalAvailableImage(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}
	manifestName := "manifest1"
	manifest, err := controller.KubeanClientSet.ManifestV1alpha1().Manifests().Create(context.Background(), &manifestv1alpha1.Manifest{
		// manifest, err := controller.InfoManifestClientSet.KubeanV1alpha1().Manifests().Create(context.Background(), &manifestv1alpha1.Manifest{
		ObjectMeta: metav1.ObjectMeta{
			Name: manifestName,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name string
		arg  func()
		want string
	}{
		{
			name: "update local kubespray image with ghcr.m.daocloud.io",
			arg: func() {
				manifest.Spec = manifestv1alpha1.Spec{
					KubeanVersion: "123",
				}
				manifest, _ := controller.KubeanClientSet.ManifestV1alpha1().Manifests().Update(context.Background(), manifest, metav1.UpdateOptions{})
				os.Setenv("POD_NAMESPACE", "")
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: util.GetCurrentNSOrDefault(),
						Name:      "kubean-config",
					},
					Data: map[string]string{
						"CLUSTER_OPERATIONS_BACKEND_LIMIT": "10000",
						"SPRAY_JOB_IMAGE_REGISTRY":         "ghcr.m.daocloud.io",
					},
				}
				controller.ClientSet.CoreV1().ConfigMaps(util.GetCurrentNSOrDefault()).Create(context.Background(), configMap, metav1.CreateOptions{})
				controller.UpdateLocalAvailableImage([]manifestv1alpha1.Manifest{*manifest})
			},
			want: "ghcr.m.daocloud.io/kubean-io/spray-job:123",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.arg()
			manifest, err := controller.KubeanClientSet.ManifestV1alpha1().Manifests().Get(context.Background(), manifestName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if manifest.Status.LocalAvailable.KubesprayImage != test.want {
				t.Fatalf("got %s, want %s", manifest.Status.LocalAvailable.KubesprayImage, test.want)
			}
		})
	}
}

func addLocalArtifactSet(controller *Controller) {
	set := &localartifactsetv1alpha1.LocalArtifactSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "set-1",
		},
		Spec: localartifactsetv1alpha1.Spec{
			Items: []*localartifactsetv1alpha1.SoftwareInfo{
				{
					Name:         "etcd-1",
					VersionRange: []string{"1.1", "1.2"},
				},
			},
		},
	}
	controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Create(context.Background(), set, metav1.CreateOptions{})
}

func TestIsOnlineENV(t *testing.T) {
	genController := func() *Controller {
		return &Controller{
			Client:          newFakeClient(),
			ClientSet:       clientsetfake.NewSimpleClientset(),
			KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
		}
	}
	controller := genController()

	tests := []struct {
		name string
		args func() bool
		want bool
	}{
		{
			name: "list but error",
			args: func() bool {
				// use plural: localartifactsets
				fetchTestingFake(controller.KubeanClientSet.LocalArtifactSetV1alpha1()).PrependReactor("list", "localartifactsets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("this is error")
				})
				defer removeReactorFromTestingTake(controller.KubeanClientSet.LocalArtifactSetV1alpha1(), "list", "localartifactsets")
				return controller.IsOnlineENV()
			},
			want: true,
		},
		{
			name: "list nothing",
			args: func() bool {
				return controller.IsOnlineENV()
			},
			want: true,
		},
		{
			name: "airgap env",
			args: func() bool {
				addLocalArtifactSet(controller)
				return controller.IsOnlineENV()
			},
			want: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.args() != test.want {
				t.Fatal()
			}
		})
	}
}

func TestFetchLocalServiceCM(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}
	controller.FetchLocalServiceCM("")
	tests := []struct {
		name string
		args func() bool
		want bool
	}{
		{
			name: "get localService from default namespace",
			args: func() bool {
				configMap := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubean-localservice",
						Namespace: "default",
					},
				}
				controller.ClientSet.CoreV1().ConfigMaps("default").Create(context.Background(), configMap, metav1.CreateOptions{})
				result, err := controller.FetchLocalServiceCM("no-exist-namespace")
				return err == nil && result != nil
			},
			want: true,
		},
		{
			name: "get localService from no-default namespace",
			args: func() bool {
				configMap := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubean-localservice",
						Namespace: "kubean-system",
					},
				}
				controller.ClientSet.CoreV1().ConfigMaps("kubean-system").Create(context.Background(), configMap, metav1.CreateOptions{})
				result, err := controller.FetchLocalServiceCM("kubean-system")
				return err == nil && result != nil
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.args() != test.want {
				t.Fatal()
			}
		})
	}
}

func TestUpdateLocalService(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}

	tests := []struct {
		name        string
		prequisites func() error
		cleaner     func()
		args        []manifestv1alpha1.Manifest
		want        bool
	}{
		{
			name: "air-gap, but no localservice configmap",
			prequisites: func() error {
				_, err := controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Create(context.Background(), &localartifactsetv1alpha1.LocalArtifactSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "localartifactset-1",
					},
				}, metav1.CreateOptions{})
				if err != nil {
					return err
				}
				return nil
			},
			cleaner: func() {
				controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Delete(context.Background(), "localartifactset-1", metav1.DeleteOptions{})
			},
			args: []manifestv1alpha1.Manifest{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
					},
				},
			},
			want: false,
		},
		{
			name: "air-gap, localservice configmap exists and correct data structure",
			prequisites: func() error {
				if _, err := controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Create(context.Background(), &localartifactsetv1alpha1.LocalArtifactSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "localartifactset-2",
					},
				}, metav1.CreateOptions{}); err != nil {
					return err
				}
				if _, err := controller.ClientSet.CoreV1().ConfigMaps("default").Create(context.Background(), &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: LocalServiceConfigMap,
					},
					Data: map[string]string{"localService": "imageRepoScheme: 'https'"},
				}, metav1.CreateOptions{}); err != nil {
					return err
				}
				return nil
			},
			cleaner: func() {
				controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Delete(context.Background(), "localartifactset-2", metav1.DeleteOptions{})
				controller.ClientSet.CoreV1().ConfigMaps("default").Delete(context.Background(), LocalServiceConfigMap, metav1.DeleteOptions{})
			},
			args: []manifestv1alpha1.Manifest{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-2",
					},
				},
			},
			want: true,
		},
		{
			name: "air-gap, localservice configmap exists and but incorrect data structure",
			prequisites: func() error {
				if _, err := controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Create(context.Background(), &localartifactsetv1alpha1.LocalArtifactSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "localartifactset-3",
					},
				}, metav1.CreateOptions{}); err != nil {
					return err
				}
				if _, err := controller.ClientSet.CoreV1().ConfigMaps("default").Create(context.Background(), &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: LocalServiceConfigMap,
					},
					Data: map[string]string{"localService": "test"},
				}, metav1.CreateOptions{}); err != nil {
					return err
				}
				return nil
			},
			cleaner: func() {
				controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Delete(context.Background(), "localartifactset-3", metav1.DeleteOptions{})
				controller.ClientSet.CoreV1().ConfigMaps("default").Delete(context.Background(), LocalServiceConfigMap, metav1.DeleteOptions{})
			},
			args: []manifestv1alpha1.Manifest{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-3",
					},
				},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.prequisites(); err != nil {
				t.Fatal(err)
			}
			defer test.cleaner()
			if controller.UpdateLocalService(test.args) != test.want {
				t.Fatal()
			}
		})
	}
}

func TestGetVersionedManifest(t *testing.T) {
	t.Run("get versioned manifest", func(t *testing.T) {
		if !reflect.DeepEqual(GetVersionedManifest(), versionedManifest) {
			t.Fatal()
		}
	})
}

func TestOp(t *testing.T) {
	var versionedManifest VersionedManifest
	type args struct {
		op string
		m1 *manifestv1alpha1.Manifest
		m2 *manifestv1alpha1.Manifest
	}
	tests := []struct {
		name          string
		args          args
		prerequisites func(versionedManifest *VersionedManifest)
		want          func() map[string][]*manifestv1alpha1.Manifest
	}{
		{
			name: "add",
			args: args{
				op: "add",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.23",
						},
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
		},
		{
			name: "add with no release label",
			args: args{
				op: "add",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{}
			},
		},
		{
			name: "update",
			args: args{
				op: "update",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.23",
						},
					},
				},
				m2: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.22",
						},
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {
				versionedManifest.Manifests = map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{
					"2.22": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.22",
								},
							},
						},
					},
				}
			},
		},
		{
			name: "update with no release label",
			args: args{
				op: "update",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
					},
				},
				m2: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							"label": "value",
						},
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{}
			},
		},
		{
			name: "update with label removing",
			args: args{
				op: "update",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.23",
						},
					},
				},
				m2: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {
				versionedManifest.Manifests = map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{}
			},
		},
		{
			name: "update with label adding",
			args: args{
				op: "update",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
					},
				},
				m2: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.23",
						},
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
		},
		{
			name: "delete",
			args: args{
				op: "delete",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.23",
						},
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {
				versionedManifest.Manifests = map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{}
			},
		},
		{
			name: "delete with no release label",
			args: args{
				op: "delete",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-2",
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {
				versionedManifest.Manifests = map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{
					"2.23": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "manifest-1",
								Labels: map[string]string{
									constants.KeySprayRelease: "2.23",
								},
							},
						},
					},
				}
			},
		},
		{
			name: "delete but no manifest cached",
			args: args{
				op: "delete",
				m1: &manifestv1alpha1.Manifest{
					ObjectMeta: metav1.ObjectMeta{
						Name: "manifest-1",
						Labels: map[string]string{
							constants.KeySprayRelease: "2.23",
						},
					},
				},
			},
			prerequisites: func(versionedManifest *VersionedManifest) {},
			want: func() map[string][]*manifestv1alpha1.Manifest {
				return map[string][]*manifestv1alpha1.Manifest{}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			versionedManifest = VersionedManifest{
				Manifests: make(map[string][]*manifestv1alpha1.Manifest, 0),
			}
			test.prerequisites(&versionedManifest)
			versionedManifest.Op(test.args.op, test.args.m1, test.args.m2)
			if !reflect.DeepEqual(test.want(), versionedManifest.Manifests) {
				t.Fatal()
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}

	manifestName := "manifest1"
	manifest := &manifestv1alpha1.Manifest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Manifest",
			APIVersion: "kubean.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: manifestName,
		},
		Spec: manifestv1alpha1.Spec{
			KubeanVersion: "123",
		},
	}

	tests := []struct {
		name string
		args func() bool
		want bool
	}{
		{
			name: "not for global-manifest",
			args: func() bool {
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: constants.InfoManifestGlobal}})
				return err == nil && result.Requeue == false
			},
			want: true,
		},
		{
			name: "fetch infomanifest but error",
			args: func() bool {
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: "abc-infomanifest"}})
				return err == nil && result.RequeueAfter == Loop
			},
			want: true,
		},
		{
			name: "fetch local service cm error in offline env",
			args: func() bool {
				controller.Client.Create(context.Background(), manifest)
				controller.KubeanClientSet.ManifestV1alpha1().Manifests().Create(context.Background(), manifest, metav1.CreateOptions{})
				controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Create(context.Background(), &localartifactsetv1alpha1.LocalArtifactSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "localartifactset-3",
					},
				}, metav1.CreateOptions{})
				defer func() {
					controller.Client.Delete(context.Background(), manifest)
					controller.KubeanClientSet.ManifestV1alpha1().Manifests().Delete(context.Background(), manifestName, metav1.DeleteOptions{})
					controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Delete(context.Background(), "localartifactset-3", metav1.DeleteOptions{})
				}()
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: manifestName}})
				return err == nil && result.Requeue == false
			},
			want: true,
		},
		{
			name: "update local service success and requeue",
			args: func() bool {
				controller.Client.Create(context.Background(), manifest)
				controller.KubeanClientSet.ManifestV1alpha1().Manifests().Create(context.Background(), manifest, metav1.CreateOptions{})
				controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Create(context.Background(), &localartifactsetv1alpha1.LocalArtifactSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "localartifactset-2",
					},
				}, metav1.CreateOptions{})
				controller.ClientSet.CoreV1().ConfigMaps("default").Create(context.Background(), &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: LocalServiceConfigMap,
					},
					Data: map[string]string{"localService": "imageRepoScheme: 'https'"},
				}, metav1.CreateOptions{})
				defer func() {
					controller.KubeanClientSet.LocalArtifactSetV1alpha1().LocalArtifactSets().Delete(context.Background(), "localartifactset-3", metav1.DeleteOptions{})
				}()
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: manifestName}})
				return err == nil && result.RequeueAfter == Loop
			},
			want: true,
		},
		{
			name: "update local available image",
			args: func() bool {
				controller.KubeanClientSet.ManifestV1alpha1().Manifests().Create(context.Background(), manifest, metav1.CreateOptions{})
				manifest.Spec = manifestv1alpha1.Spec{
					KubeanVersion: "123",
				}
				controller.KubeanClientSet.ManifestV1alpha1().Manifests().Update(context.Background(), manifest, metav1.UpdateOptions{})
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: manifestName}})
				return err == nil && result.RequeueAfter == Loop
			},
			want: true,
		},
		{
			name: "list manifests error",
			args: func() bool {
				fetchTestingFake(controller.KubeanClientSet.ManifestV1alpha1()).PrependReactor("list", "manifests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("this is error when list manifests")
				})
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: manifestName}})
				return err == nil && result.RequeueAfter == Loop
			},
			want: true,
		},
		{
			name: "list manifests error and isNotFound",
			args: func() bool {
				fetchTestingFake(controller.KubeanClientSet.ManifestV1alpha1()).PrependReactor("list", "manifests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.NewNotFound(manifestv1alpha1.Resource("manifests"), "manifest-name")
				})
				result, err := controller.Reconcile(context.Background(), controllerruntime.Request{NamespacedName: types.NamespacedName{Name: manifestName}})
				return err == nil && result.Requeue == false
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.args() != test.want {
				t.Fatal()
			}
		})
	}
}

func TestStart(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	controller.Start(ctx)
}

func TestSetupWithManager(t *testing.T) {
	controller := &Controller{
		Client:          newFakeClient(),
		ClientSet:       clientsetfake.NewSimpleClientset(),
		KubeanClientSet: kubeanclientsetfake.NewSimpleClientset(),
	}
	if controller.SetupWithManager(MockManager{}) != nil {
		t.Fatal()
	}
}

type MockClusterForManager struct {
	_ string
}

func (MockClusterForManager) SetFields(interface{}) error { return nil }

func (MockClusterForManager) GetConfig() *rest.Config { return &rest.Config{} }

func (MockClusterForManager) GetScheme() *runtime.Scheme {
	sch := scheme.Scheme
	if err := manifestv1alpha1.AddToScheme(sch); err != nil {
		panic(err)
	}
	if err := localartifactsetv1alpha1.AddToScheme(sch); err != nil {
		panic(err)
	}
	return sch
}

func (MockClusterForManager) GetClient() client.Client { return nil }

func (MockClusterForManager) GetFieldIndexer() client.FieldIndexer { return nil }

func (MockClusterForManager) GetCache() cache.Cache { return nil }

func (MockClusterForManager) GetEventRecorderFor(name string) record.EventRecorder { return nil }

func (MockClusterForManager) GetRESTMapper() meta.RESTMapper { return nil }

func (MockClusterForManager) GetAPIReader() client.Reader { return nil }

func (MockClusterForManager) Start(ctx context.Context) error { return nil }

type MockManager struct {
	MockClusterForManager
}

func (MockManager) Add(manager.Runnable) error { return nil }

func (MockManager) Elected() <-chan struct{} { return nil }

func (MockManager) AddMetricsExtraHandler(path string, handler http.Handler) error { return nil }

func (MockManager) AddHealthzCheck(name string, check healthz.Checker) error { return nil }

func (MockManager) AddReadyzCheck(name string, check healthz.Checker) error { return nil }

func (MockManager) Start(ctx context.Context) error { return nil }

func (MockManager) GetWebhookServer() webhook.Server { return nil }

func (MockManager) GetLogger() logr.Logger { return logr.Logger{} }

func (MockManager) GetControllerOptions() config.Controller { return config.Controller{} }

func (MockManager) AddMetricsServerExtraHandler(path string, handler http.Handler) error { return nil }

func (MockManager) GetHTTPClient() *http.Client { return &http.Client{} }
