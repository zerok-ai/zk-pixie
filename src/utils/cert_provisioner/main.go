package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"pixielabs.ai/pixielabs/src/utils/shared/certs"
	"pixielabs.ai/pixielabs/src/utils/shared/k8s"
)

func init() {
	pflag.String("namespace", "pl", "The namespace used by Pixie")
}

func main() {
	pflag.Parse()

	// Must call after all flags are setup.
	viper.AutomaticEnv()
	viper.SetEnvPrefix("PL")
	viper.BindPFlags(pflag.CommandLine)

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to create in cluster config")
	}
	// Create k8s client.
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create in cluster client set")
	}

	log.Info("Checking if certs already exist...")

	ns := viper.GetString("namespace")
	s := k8s.GetSecret(clientset, ns, "proxy-tls-certs")

	if s != nil {
		log.Info("Certs already exist... Exiting job")
		return
	}

	// Assign JWT signing key.
	jwtSigningKey := make([]byte, 64)
	_, err = rand.Read(jwtSigningKey)
	if err != nil {
		log.WithError(err).Fatal("Could not generate JWT signing key")
	}
	s = k8s.GetSecret(clientset, ns, "pl-cluster-secrets")
	if s == nil {
		log.Fatal("pl-cluster-secrets does not exist")
	}
	s.Data["jwt-signing-key"] = []byte(fmt.Sprintf("%x", jwtSigningKey))

	s, err = clientset.CoreV1().Secrets(ns).Update(context.Background(), s, metav1.UpdateOptions{})
	if err != nil {
		log.WithError(err).Fatal("Could not update cluster secrets")
	}

	certYAMLs, err := certs.DefaultGenerateCertYAMLs(ns)
	if err != nil {
		log.WithError(err).Fatal("Failed to generate cert YAMLs")
	}

	err = k8s.ApplyYAML(clientset, kubeConfig, ns, strings.NewReader(certYAMLs))
	if err != nil {
		log.WithError(err).Fatalf("Failed deploy cert YAMLs")
	}

	log.Info("Done provisioning certs!")
}
