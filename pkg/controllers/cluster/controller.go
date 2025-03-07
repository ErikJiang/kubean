// Copyright 2023 Authors of kubean-io
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	clusterv1alpha1 "github.com/kubean-io/kubean-api/apis/cluster/v1alpha1"
	clusteroperationv1alpha1 "github.com/kubean-io/kubean-api/apis/clusteroperation/v1alpha1"

	kubeanv1alpha1clientset "github.com/kubean-io/kubean-api/client/clientset/versioned"
  // clusterv1alpha1cs "github.com/kubean-io/kubean-api/client/clientset/versioned/typed/cluster/v1alpha1"
	// clusteroperationv1alpha1cs "github.com/kubean-io/kubean-api/client/clientset/versioned/typed/clusteroperation/v1alpha1"

	"github.com/kubean-io/kubean/pkg/util"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	klog "k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RequeueAfter        = time.Second * 15
	KubeanConfigMapName = "kubean-config"
	EliminateScoreAnno  = "kubean.io/eliminate-score"
)

type Controller struct {
	Client          client.Client
	ClientSet       kubernetes.Interface
	KubeanClientSet kubeanv1alpha1clientset.Interface
	// ClusterClientSet clusterv1alpha1cs.ClusterV1alpha1Interface
	// ClusterOperationClientSet clusteroperationv1alpha1cs.ClusterOperationV1alpha1Interface
}

func (c *Controller) Start(ctx context.Context) error {
	klog.Warningf("Cluster Controller Start")
	<-ctx.Done()
	return nil
}

func CompareClusterCondition(conditionA, conditionB clusterv1alpha1.ClusterCondition) bool {
	unixMilli := func(t *metav1.Time) int64 {
		if t == nil {
			return -1
		}
		return t.UnixMilli()
	}
	if conditionA.ClusterOps != conditionB.ClusterOps {
		return false
	}
	if conditionA.Status != conditionB.Status {
		return false
	}
	if unixMilli(conditionA.StartTime) != unixMilli(conditionB.StartTime) {
		return false
	}
	if unixMilli(conditionA.EndTime) != unixMilli(conditionB.EndTime) {
		return false
	}
	return true
}

func CompareClusterConditions(condAList, condBList []clusterv1alpha1.ClusterCondition) bool {
	if len(condAList) != len(condBList) {
		return false
	}
	for i := range condAList {
		if !CompareClusterCondition(condAList[i], condBList[i]) {
			return false
		}
	}
	return true
}

func (c *Controller) UpdateStatus(cluster *clusterv1alpha1.Cluster) error {
	listOpt := metav1.ListOptions{LabelSelector: fmt.Sprintf("clusterName=%s", cluster.Name)}
	clusterOpsList, err := c.KubeanClientSet.ClusterOperationV1alpha1().ClusterOperations().List(context.Background(), listOpt)
	// clusterOpsList, err := c.ClusterOperationClientSet.ClusterOperations().List(context.Background(), listOpt)
	if err != nil {
		return err
	}
	// clusterOps list sort by creation timestamp
	c.SortClusterOperationsByCreation(clusterOpsList.Items)
	newConditions := make([]clusterv1alpha1.ClusterCondition, 0)
	for _, item := range clusterOpsList.Items {
		newConditions = append(newConditions, clusterv1alpha1.ClusterCondition{
			ClusterOps: item.Name,
			Status:     clusterv1alpha1.ClusterConditionType(item.Status.Status),
			StartTime:  item.Status.StartTime,
			EndTime:    item.Status.EndTime,
		})
	}
	if !CompareClusterConditions(cluster.Status.Conditions, newConditions) {
		// need update for newCondition
		cluster.Status.Conditions = newConditions
		klog.Warningf("update cluster %s status.condition", cluster.Name)
		return c.Client.Status().Update(context.Background(), cluster)
	}
	return nil
}

func (c *Controller) GetEliminateScoreValue(operation clusteroperationv1alpha1.ClusterOperation) int {
	value, err := strconv.Atoi(operation.Annotations[EliminateScoreAnno])
	if err != nil {
		return 0
	}
	return value
}

// SortClusterOperationsByCreation sort operations order by EliminateScore ascend , createTime desc.
func (c *Controller) SortClusterOperationsByCreation(operations []clusteroperationv1alpha1.ClusterOperation) {
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].CreationTimestamp.After(operations[j].CreationTimestamp.Time)
	})
	sort.Slice(operations, func(i, j int) bool {
		return c.GetEliminateScoreValue(operations[i]) < c.GetEliminateScoreValue(operations[j])
	})
}

// CleanExcessClusterOps clean up excess ClusterOperation.
func (c *Controller) CleanExcessClusterOps(cluster *clusterv1alpha1.Cluster, OpsBackupNum int) (bool, error) {
	listOpt := metav1.ListOptions{LabelSelector: fmt.Sprintf("clusterName=%s", cluster.Name)}
	clusterOpsList, err := c.KubeanClientSet.ClusterOperationV1alpha1().ClusterOperations().List(context.Background(), listOpt)
	// clusterOpsList, err := c.ClusterOperationClientSet.ClusterOperations().List(context.Background(), listOpt)
	if err != nil {
		return false, err
	}
	if len(clusterOpsList.Items) <= OpsBackupNum {
		return false, nil
	}

	c.SortClusterOperationsByCreation(clusterOpsList.Items)

	excessClusterOpsList := clusterOpsList.Items[OpsBackupNum:]
	for _, item := range excessClusterOpsList {
		if item.Status.Status == clusteroperationv1alpha1.RunningStatus { // keep running job
			continue
		}
		klog.Warningf("Delete ClusterOperation: name: %s, createTime: %s, status: %s", item.Name, item.CreationTimestamp.String(), item.Status.Status)
		c.KubeanClientSet.ClusterOperationV1alpha1().ClusterOperations().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
		// c.ClusterOperationClientSet.ClusterOperations().Delete(context.Background(), item.Name, metav1.DeleteOptions{})
	}
	return true, nil
}

func (c *Controller) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	cluster := &clusterv1alpha1.Cluster{}
	if err := c.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		klog.ErrorS(err, "failed to get cluster", "cluster", req.String())
		return controllerruntime.Result{RequeueAfter: RequeueAfter}, nil
	}
	OpsBackupNum := util.FetchKubeanConfigProperty(c.ClientSet).GetClusterOperationsBackEndLimit()
	needRequeue, err := c.CleanExcessClusterOps(cluster, OpsBackupNum)
	if err != nil {
		klog.ErrorS(err, "failed to clean excess cluster ops", "cluster", cluster.Name)
		return controllerruntime.Result{RequeueAfter: RequeueAfter}, nil
	}
	if needRequeue {
		return controllerruntime.Result{RequeueAfter: RequeueAfter}, nil
	}

	if err := c.UpdateStatus(cluster); err != nil {
		klog.ErrorS(err, "failed to update cluster status", "cluster", cluster.Name)
		return controllerruntime.Result{RequeueAfter: RequeueAfter}, nil
	}
	if err := c.UpdateOwnReferenceToCluster(cluster); err != nil {
		klog.ErrorS(err, "failed to update the ownReference configData or secretData", "cluster", cluster.Name)
		return controllerruntime.Result{RequeueAfter: RequeueAfter}, nil
	}
	return controllerruntime.Result{RequeueAfter: RequeueAfter}, nil // loop
}

func (c *Controller) UpdateOwnReferenceToCluster(cluster *clusterv1alpha1.Cluster) error {
	return util.UpdateOwnReference(c.ClientSet,
		cluster.Spec.ConfigDataList(),
		cluster.Spec.SecretDataList(),
		metav1.OwnerReference{
			APIVersion: clusterv1alpha1.SchemeGroupVersion.String(),
			Kind:       clusterv1alpha1.SchemeGroupVersion.WithKind("Cluster").Kind,
			Name:       cluster.Name,
			UID:        cluster.GetUID(),
		},
	)
}

func (c *Controller) SetupWithManager(mgr controllerruntime.Manager) error {
	return utilerrors.NewAggregate([]error{
		controllerruntime.NewControllerManagedBy(mgr).For(&clusterv1alpha1.Cluster{}).Complete(c),
		mgr.Add(c),
	})
}
