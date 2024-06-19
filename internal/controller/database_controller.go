/*
Copyright 2024 Falk Winkler.

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

package controller

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mssqlv1alpha1 "github.com/falkwinkler/mssql-operator/api/v1alpha1"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mssql.f-wi.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mssql.f-wi.com,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mssql.f-wi.com,resources=databases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Database object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (reconciler *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconcile started for Database CRD")

	database := &mssqlv1alpha1.Database{}

	err := reconciler.Get(ctx, req.NamespacedName, database)

	if err != nil {
		return ctrl.Result{}, err
	}

	var databaseCluster mssqlv1alpha1.Cluster

	err = reconciler.Get(ctx, client.ObjectKey{
		Namespace: req.Namespace,
		Name:      database.Spec.ClusterName,
	}, &databaseCluster)

	if err != nil {
		log.Info("Cluster Keyobject not found")
		return ctrl.Result{}, err
	}

	port := 1433 //databaseCluster.Spec.Port
	server := databaseCluster.GetServiceName()
	user := "sa"
	password := "admin@123"

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;InitialCatalog=master", server, user, password, port)

	log.Info("Open Database with ConnectionString " + connString)

	conn, err := sql.Open("mssql", connString)
	if err != nil {
		log.Error(err, "Open connection failed:")
	}
	defer conn.Close()

	statement := fmt.Sprintf("SELECT COUNT(*) FROM SYS.DATABASES WHERE NAME = %s", database.Spec.Name)
	result, err := conn.Exec(statement)
	if err != nil {
		log.Error(err, "Prepare failed:")
	}

	rowCount, err := result.RowsAffected()

	if rowCount == 0 {
		log.Info(fmt.Sprintf("Database %s (%s) was not found!", database.Spec.Name, database.GetName()))

		statement := "CREATE DATABASE " + database.Spec.Name
		_, err := conn.Exec(statement)

		if err != nil {
			log.Error(err, "create database failed:")
		}

		log.Info("Database " + database.Spec.Name + " succeefully created!")
	}

	log.Info("bye\n")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mssqlv1alpha1.Database{}).
		Complete(r)
}
