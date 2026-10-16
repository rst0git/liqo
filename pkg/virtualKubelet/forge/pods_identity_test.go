// Copyright 2019-2026 The Liqo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package forge_test

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	offloadingv1beta1 "github.com/liqotech/liqo/apis/offloading/v1beta1"
	"github.com/liqotech/liqo/pkg/virtualKubelet/forge"
)

func TestRemoteShadowPodOriginPodUID(t *testing.T) {
	for _, tt := range []struct {
		name     string
		localUID types.UID
		remote   *offloadingv1beta1.ShadowPod
		wantUID  string
	}{
		{name: "creation binds source UID", localUID: "source-uid", wantUID: "source-uid"},
		{name: "missing source UID stays unbound"},
		{
			name: "update preserves binding", localUID: "source-uid", wantUID: "source-uid",
			remote: &offloadingv1beta1.ShadowPod{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{forge.LiqoOriginPodUIDAnnotation: "source-uid"},
			}},
		},
		{
			name: "reused source name cannot rebind", localUID: "replacement-uid", wantUID: "original-uid",
			remote: &offloadingv1beta1.ShadowPod{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{forge.LiqoOriginPodUIDAnnotation: "original-uid"},
			}},
		},
		{
			name: "legacy ShadowPod stays unbound", localUID: "source-uid",
			remote: &offloadingv1beta1.ShadowPod{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			local := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "source", Namespace: "source-namespace", UID: tt.localUID,
				Annotations: map[string]string{
					forge.LiqoOriginPodUIDAnnotation: "spoofed-uid",
					"example.com/annotation":         "preserved",
				},
			}}
			localBefore, remoteBefore := local.DeepCopy(), tt.remote.DeepCopy()
			// Provenance is reflector-owned, even if annotation filtering requests its removal.
			output := forge.RemoteShadowPod(local, tt.remote, "provider-namespace", &forge.ForgingOpts{
				AnnotationsNotReflected: []string{forge.LiqoOriginPodUIDAnnotation},
			})
			if got := output.Annotations[forge.LiqoOriginPodUIDAnnotation]; got != tt.wantUID {
				t.Errorf("origin Pod UID = %q, want %q", got, tt.wantUID)
			}
			unfiltered := forge.RemoteShadowPod(local, tt.remote, "provider-namespace", &forge.ForgingOpts{})
			if got := unfiltered.Annotations[forge.LiqoOriginPodUIDAnnotation]; got != tt.wantUID {
				t.Errorf("unfiltered origin Pod UID = %q, want %q", got, tt.wantUID)
			}
			if tt.wantUID == "" {
				if _, exists := output.Annotations[forge.LiqoOriginPodUIDAnnotation]; exists {
					t.Error("unbound ShadowPod must omit provenance")
				}
			}
			if got := output.Annotations["example.com/annotation"]; got != "preserved" {
				t.Errorf("ordinary annotation = %q, want preserved", got)
			}
			if !reflect.DeepEqual(local, localBefore) || !reflect.DeepEqual(tt.remote, remoteBefore) {
				t.Error("forging mutated its input objects")
			}
		})
	}
}
