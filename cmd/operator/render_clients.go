package main

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"

	radiusv1alpha1 "github.com/example/freeradius-operator/api/v1alpha1"
	"github.com/example/freeradius-operator/internal/renderer"
)

func renderClientsToFile(ctx context.Context, k8sClient client.Reader, namespace, clusterName, outputPath string) error {
	list := &radiusv1alpha1.RadiusClientList{}
	if err := k8sClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("listing RadiusClients: %w", err)
	}

	var clients []renderer.ClientSpec
	for _, c := range list.Items {
		if c.Spec.ClusterRef != clusterName {
			continue
		}
		clients = append(clients, renderer.ClientSpec{
			Name:      c.Name,
			IP:        c.Spec.IP,
			SecretRef: renderer.SecretRef{Name: c.Spec.SecretRef.Name, Key: c.Spec.SecretRef.Key},
			NASType:   c.Spec.NASType,
		})
	}

	content, err := renderer.RenderClients(clients)
	if err != nil {
		return fmt.Errorf("rendering clients.conf: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	fmt.Fprintf(os.Stderr, "rendered %d clients to %s\n", len(clients), outputPath)
	return nil
}
