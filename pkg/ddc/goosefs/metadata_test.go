/*
Copyright 2021 The Fluid Authors.

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

package goosefs

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/brahma-adshonor/gohook"
	datav1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/ddc/goosefs/operations"
	"github.com/fluid-cloudnative/fluid/pkg/utils/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSyncMetadata(t *testing.T) {
	QueryMetaDataInfoIntoFileCommon := func(a operations.GooseFSFileUtils, key operations.KeyOfMetaDataFile, filename string) (value string, err error) {
		return "1024", nil
	}

	wrappedUnhookQueryMetaDataInfoIntoFile := func() {
		err := gohook.UnHook(operations.GooseFSFileUtils.QueryMetaDataInfoIntoFile)
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	datasetInputs := []datav1alpha1.Dataset{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Status: datav1alpha1.DatasetStatus{
				UfsTotal: "2Gi",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
			Status: datav1alpha1.DatasetStatus{
				UfsTotal: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hadoop",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{
				DataRestoreLocation: &datav1alpha1.DataRestoreLocation{
					Path:     "local:///host1/erf",
					NodeName: "test-node",
				},
			},
			Status: datav1alpha1.DatasetStatus{
				UfsTotal: "",
			},
		},
	}

	testObjs := []runtime.Object{}
	for _, datasetInput := range datasetInputs {
		testObjs = append(testObjs, datasetInput.DeepCopy())
	}
	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []GooseFSEngine{
		{
			name:      "hbase",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
		{
			name:      "spark",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
	}

	for _, engine := range engines {
		err := engine.SyncMetadata()
		if err != nil {
			t.Errorf("fail to exec the function")
		}
	}

	engine := GooseFSEngine{
		name:      "hadoop",
		namespace: "fluid",
		Client:    client,
		Log:       fake.NullLogger(),
	}

	err := gohook.Hook(operations.GooseFSFileUtils.QueryMetaDataInfoIntoFile, QueryMetaDataInfoIntoFileCommon, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = engine.SyncMetadata()
	if err != nil {
		t.Errorf("fail to exec function RestoreMetadataInternal")
	}
	wrappedUnhookQueryMetaDataInfoIntoFile()
}

func TestShouldSyncMetadata(t *testing.T) {
	datasetInputs := []datav1alpha1.Dataset{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Status: datav1alpha1.DatasetStatus{
				UfsTotal: "2Gi",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
			Status: datav1alpha1.DatasetStatus{
				UfsTotal: "",
			},
		},
	}
	testObjs := []runtime.Object{}
	for _, datasetInput := range datasetInputs {
		testObjs = append(testObjs, datasetInput.DeepCopy())
	}
	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []GooseFSEngine{
		{
			name:      "hbase",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
		{
			name:      "spark",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
	}

	var testCases = []struct {
		engine         GooseFSEngine
		expectedShould bool
	}{
		{
			engine:         engines[0],
			expectedShould: false,
		},
		{
			engine:         engines[1],
			expectedShould: true,
		},
	}

	for _, test := range testCases {
		should, err := test.engine.shouldSyncMetadata()
		if err != nil || should != test.expectedShould {
			t.Errorf("fail to exec the function")
		}
	}
}

func TestShouldRestoreMetadata(t *testing.T) {
	datasetInputs := []datav1alpha1.Dataset{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{
				DataRestoreLocation: &datav1alpha1.DataRestoreLocation{
					Path:     "local:///host1/erf",
					NodeName: "test-node",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
		},
	}
	testObjs := []runtime.Object{}
	for _, datasetInput := range datasetInputs {
		testObjs = append(testObjs, datasetInput.DeepCopy())
	}

	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []GooseFSEngine{
		{
			name:      "hbase",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
		{
			name:      "spark",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
	}

	var testCases = []struct {
		engine         GooseFSEngine
		expectedShould bool
	}{
		{
			engine:         engines[0],
			expectedShould: true,
		},
		{
			engine:         engines[1],
			expectedShould: false,
		},
	}
	for _, test := range testCases {
		should, err := test.engine.shouldRestoreMetadata()
		if err != nil || should != test.expectedShould {
			t.Errorf("fail to exec the function")
		}
	}
}

func TestRestoreMetadataInternal(t *testing.T) {
	QueryMetaDataInfoIntoFileCommon := func(a operations.GooseFSFileUtils, key operations.KeyOfMetaDataFile, filename string) (value string, err error) {
		return "1024", nil
	}
	QueryMetaDataInfoIntoFileErr := func(a operations.GooseFSFileUtils, key operations.KeyOfMetaDataFile, filename string) (value string, err error) {
		return "", errors.New("fail to query MetaDataInfo")
	}

	wrappedUnhookQueryMetaDataInfoIntoFile := func() {
		err := gohook.UnHook(operations.GooseFSFileUtils.QueryMetaDataInfoIntoFile)
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	datasetInputs := []datav1alpha1.Dataset{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{
				DataRestoreLocation: &datav1alpha1.DataRestoreLocation{
					Path:     "local:///host1/erf",
					NodeName: "test-node",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{
				DataRestoreLocation: &datav1alpha1.DataRestoreLocation{
					Path:     "pvc://pvc1/erf",
					NodeName: "test-node",
				},
			},
		},
	}
	testObjs := []runtime.Object{}
	for _, datasetInput := range datasetInputs {
		testObjs = append(testObjs, datasetInput.DeepCopy())
	}

	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []GooseFSEngine{
		{
			name:      "hbase",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
		{
			name:      "hbase",
			namespace: "fluid",
			Client:    client,
			Log:       fake.NullLogger(),
		},
	}

	err := gohook.Hook(operations.GooseFSFileUtils.QueryMetaDataInfoIntoFile, QueryMetaDataInfoIntoFileErr, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = engines[0].RestoreMetadataInternal()
	if err == nil {
		t.Errorf("fail to catch the error")
	}
	wrappedUnhookQueryMetaDataInfoIntoFile()

	err = gohook.Hook(operations.GooseFSFileUtils.QueryMetaDataInfoIntoFile, QueryMetaDataInfoIntoFileCommon, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	var testCases = []struct {
		engine                  GooseFSEngine
		expectedDatasetUfsTotal string
		expectedDatasetFileNum  string
	}{
		{
			engine:                  engines[0],
			expectedDatasetUfsTotal: "1.00KiB",
			expectedDatasetFileNum:  "1024",
		},
		{
			engine:                  engines[1],
			expectedDatasetUfsTotal: "1.00KiB",
			expectedDatasetFileNum:  "1024",
		},
	}

	for _, test := range testCases {
		err = test.engine.RestoreMetadataInternal()
		if err != nil {
			t.Errorf("fail to exec function RestoreMetadataInternal")
		}
	}
	wrappedUnhookQueryMetaDataInfoIntoFile()
}

func TestSyncMetadataInternal(t *testing.T) {
	datasetInputs := []datav1alpha1.Dataset{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{
				Mounts: []datav1alpha1.Mount{
					{
						MountPoint: "cosn://imagenet-1234567/",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hadoop",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{},
		},
	}
	testObjs := []runtime.Object{}
	for _, datasetInput := range datasetInputs {
		testObjs = append(testObjs, datasetInput.DeepCopy())
	}
	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []GooseFSEngine{
		{
			name:               "hbase",
			namespace:          "fluid",
			Client:             client,
			Log:                fake.NullLogger(),
			MetadataSyncDoneCh: make(chan MetadataSyncResult),
		},
		{
			name:               "hadoop",
			namespace:          "fluid",
			Client:             client,
			Log:                fake.NullLogger(),
			MetadataSyncDoneCh: nil,
		},
	}

	result := MetadataSyncResult{
		StartTime: time.Now(),
		UfsTotal:  "2GB",
		Done:      true,
		FileNum:   "5",
	}

	var testCase = []struct {
		engine           GooseFSEngine
		expectedResult   bool
		expectedUfsTotal string
		expectedFileNum  string
	}{
		{
			engine:           engines[0],
			expectedUfsTotal: "2GB",
			expectedFileNum:  "5",
		},
		{
			engine:           engines[1],
			expectedUfsTotal: "2GB",
			expectedFileNum:  "1331167",
		},
	}

	for index, test := range testCase {
		if index == 0 {
			go func() {
				test.engine.MetadataSyncDoneCh <- result
			}()
		}

		if index == 1 {
			var goosefsFileUtils operations.GooseFSFileUtils
			patch1 := ApplyMethod(reflect.TypeOf(goosefsFileUtils), "LoadMetadataWithoutTimeout", func(_ operations.GooseFSFileUtils, path string) error {
				return nil
			})
			defer patch1.Reset()

			patch2 := ApplyMethod(reflect.TypeOf(&test.engine), "TotalStorageBytes", func(_ *GooseFSEngine) (int64, error) {
				return 16, nil
			})
			defer patch2.Reset()

			patch3 := ApplyMethod(reflect.TypeOf(&test.engine), "TotalFileNums", func(_ *GooseFSEngine) (int64, error) {
				return 16, nil
			})
			defer patch3.Reset()

			patch4 := ApplyFunc(retry.RetryOnConflict, func(backoff wait.Backoff, fn func() error) error {
				return nil
			})
			defer patch4.Reset()
		}

		err := test.engine.syncMetadataInternal()
		if err != nil {
			t.Errorf("fail to exec the function with error %v", err)
		}

		key := types.NamespacedName{
			Namespace: test.engine.namespace,
			Name:      test.engine.name,
		}

		dataset := &datav1alpha1.Dataset{}
		err = client.Get(context.TODO(), key, dataset)
		if err != nil {
			t.Errorf("failt to get the dataset with error %v", err)
		}

		if index != 1 {
			if dataset.Status.UfsTotal != test.expectedUfsTotal || dataset.Status.FileNum != test.expectedFileNum {
				t.Errorf("%s expected UfsTotal %s, get UfsTotal %s, expected FileNum %s, get FileNum %s", test.engine.name, test.expectedUfsTotal, dataset.Status.UfsTotal, test.expectedFileNum, dataset.Status.FileNum)
			}
		}
	}
}
