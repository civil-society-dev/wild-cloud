package v1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// AsyncOperation is a function that performs an async operation.
// It receives the operations manager and operation ID for progress updates.
// The function should return an error if the operation fails.
type AsyncOperation func(opsMgr *operations.Manager, opID string) error

// StartAsyncOperation starts an async operation with consistent error handling,
// panic recovery, and status tracking.
//
// This is the preferred way to start async operations like deployments, backups, etc.
// It handles:
// - Creating the operation record
// - Starting a goroutine with panic recovery
// - Updating operation status on completion/failure
// - Returning the standard accepted response
func (api *API) StartAsyncOperation(
	w http.ResponseWriter,
	instanceName, operationType, target string,
	operation AsyncOperation,
) {
	opsMgr := operations.NewManager(api.dataDir)
	opID, err := opsMgr.Start(instanceName, operationType, target)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start operation")
		return
	}

	go func() {
		// Always recover from panics to prevent goroutine crashes from taking down the server
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ERROR] Panic in async operation %s/%s: %v", operationType, target, r)
				_ = opsMgr.Update(instanceName, opID, "failed", fmt.Sprintf("Internal error: %v", r), 0)
			}
		}()

		_ = opsMgr.UpdateStatus(instanceName, opID, "running")

		if err := operation(opsMgr, opID); err != nil {
			_ = opsMgr.Update(instanceName, opID, "failed", err.Error(), 0)
		} else {
			_ = opsMgr.Update(instanceName, opID, "completed", "Operation completed successfully", 100)
		}
	}()

	respondAccepted(w, opID, fmt.Sprintf("%s initiated", operationType))
}

// StartAsyncOperationWithMessage is like StartAsyncOperation but allows a custom success message.
func (api *API) StartAsyncOperationWithMessage(
	w http.ResponseWriter,
	instanceName, operationType, target, successMessage string,
	operation AsyncOperation,
) {
	opsMgr := operations.NewManager(api.dataDir)
	opID, err := opsMgr.Start(instanceName, operationType, target)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start operation")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ERROR] Panic in async operation %s/%s: %v", operationType, target, r)
				_ = opsMgr.Update(instanceName, opID, "failed", fmt.Sprintf("Internal error: %v", r), 0)
			}
		}()

		_ = opsMgr.UpdateStatus(instanceName, opID, "running")

		if err := operation(opsMgr, opID); err != nil {
			_ = opsMgr.Update(instanceName, opID, "failed", err.Error(), 0)
		} else {
			_ = opsMgr.Update(instanceName, opID, "completed", successMessage, 100)
		}
	}()

	respondAccepted(w, opID, fmt.Sprintf("%s initiated", operationType))
}

// StartAsyncOperationWithBroadcaster is like StartAsyncOperation but also passes the broadcaster
// for real-time output streaming.
func (api *API) StartAsyncOperationWithBroadcaster(
	w http.ResponseWriter,
	instanceName, operationType, target string,
	operation func(opsMgr *operations.Manager, opID string, broadcaster *operations.Broadcaster) error,
) {
	opsMgr := operations.NewManager(api.dataDir)
	opID, err := opsMgr.Start(instanceName, operationType, target)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start operation")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ERROR] Panic in async operation %s/%s: %v", operationType, target, r)
				_ = opsMgr.Update(instanceName, opID, "failed", fmt.Sprintf("Internal error: %v", r), 0)
			}
		}()

		_ = opsMgr.UpdateStatus(instanceName, opID, "running")

		if err := operation(opsMgr, opID, api.broadcaster); err != nil {
			_ = opsMgr.Update(instanceName, opID, "failed", err.Error(), 0)
		} else {
			_ = opsMgr.Update(instanceName, opID, "completed", "Operation completed successfully", 100)
		}
	}()

	respondAccepted(w, opID, fmt.Sprintf("%s initiated", operationType))
}
