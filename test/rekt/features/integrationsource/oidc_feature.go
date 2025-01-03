/*
Copyright 2024 The Knative Authors

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

package integrationsource

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/test"
	"knative.dev/eventing/test/rekt/features/featureflags"
	"knative.dev/eventing/test/rekt/features/source"
	"knative.dev/eventing/test/rekt/resources/integrationsource"
	"knative.dev/reconciler-test/pkg/eventshub"
	"knative.dev/reconciler-test/pkg/eventshub/assert"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/resources/service"
)

func SendsEventsWithSinkRefOIDC() *feature.Feature {
	src := feature.MakeRandomK8sName("integrationsource")
	sink := feature.MakeRandomK8sName("sink")
	sinkAudience := "audience"
	f := feature.NewFeature()

	f.Prerequisite("OIDC authentication is enabled", featureflags.AuthenticationOIDCEnabled())
	f.Prerequisite("transport encryption is strict", featureflags.TransportEncryptionStrict())
	f.Prerequisite("should not run when Istio is enabled", featureflags.IstioDisabled())

	f.Setup("install sink", eventshub.Install(sink,
		eventshub.OIDCReceiverAudience(sinkAudience),
		eventshub.StartReceiverTLS))

	f.Requirement("install integrationsource", func(ctx context.Context, t feature.T) {
		d := service.AsDestinationRef(sink)
		d.CACerts = eventshub.GetCaCerts(ctx)
		d.Audience = &sinkAudience

		integrationsource.Install(src, integrationsource.WithSink(d))(ctx, t)
	})

	f.Requirement("integrationsource goes ready", integrationsource.IsReady(src))

	f.Stable("integrationsource as event source").
		Must("delivers events",
			assert.OnStore(sink).MatchEvent(test.HasType("dev.knative.eventing.timer")).AtLeast(1)).
		Must("uses integrationsources identity for OIDC", assert.OnStore(sink).MatchWithContext(
			assert.MatchKind(eventshub.EventReceived).WithContext(),
			assert.MatchOIDCUserFromResource(integrationsource.Gvr(), src)).AtLeast(1)).
		Must("Set sinkURI to HTTPS endpoint", source.ExpectHTTPSSink(integrationsource.Gvr(), src)).
		Must("Set sinkCACerts to non empty CA certs", source.ExpectCACerts(integrationsource.Gvr(), src))
	return f
}
