package device_selection

import (
	"context"
	"errors"
	"time"

	"github.com/flightctl/flightctl/internal/store"
	"github.com/flightctl/flightctl/internal/store/model"
	"github.com/flightctl/flightctl/internal/tasks"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

const RolloutDeviceSelectionInterval = 2 * time.Minute

type Reconciler interface {
	Reconcile()
}

type reconciler struct {
	store           store.Store
	log             logrus.FieldLogger
	callbackManager tasks.CallbackManager
}

func NewReconciler(store store.Store, callbackManager tasks.CallbackManager, log logrus.FieldLogger) Reconciler {
	return &reconciler{
		store:           store,
		log:             log,
		callbackManager: callbackManager,
	}
}

func (r *reconciler) Reconcile() {
	ctx := context.Background()

	// Get all relevant fleets
	orgId := store.NullOrgId

	fleetList, err := r.store.Fleet().ListRolloutDeviceSelection(ctx, orgId)
	if err != nil {
		r.log.WithError(err)
		return
	}
	var errs []error
	for _, fleet := range fleetList.Items {
		if fleet.Spec.RolloutPolicy == nil {
			continue
		}
		annotations := lo.FromPtr(fleet.Metadata.Annotations)
		if annotations == nil {
			continue
		}
		templateVersionName, exists := annotations[model.FleetAnnotationTemplateVersion]
		if !exists {
			continue
		}
		selector, err := NewRolloutDeviceSelector(fleet.Spec.RolloutPolicy.DeviceSelection, r.store, orgId, &fleet, templateVersionName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if selector.IsRolloutNew() {
			if err := selector.OnNewRollout(ctx); err != nil {
				errs = append(errs, err)
				continue
			}
			if err := selector.Reset(ctx); err != nil {
				errs = append(errs, err)
				continue
			}
		}

		selection, err := selector.CurrentSelection(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for {
			isRolledOut, err := selection.IsRolledOut(ctx)
			if err != nil {
				errs = append(errs, err)
				break
			}
			if !isRolledOut {
				if !selection.IsApproved() {
					mayApprove, err := selection.MayApproveAutomatically()
					if err != nil {
						errs = append(errs, err)
						break
					}
					if mayApprove {
						if err = selection.Approve(ctx); err != nil {
							errs = append(errs, err)
							break
						}
					} else {
						break
					}
				}
				modelFleet, err := model.NewFleetFromApiResource(&fleet)
				if err != nil {
					errs = append(errs, err)
					break
				}
				r.callbackManager.FleetRolloutSelectionUpdated(modelFleet)
			}
			isComplete, err := selection.IsComplete(ctx)
			if err != nil {
				errs = append(errs, err)
				break
			}
			if !isComplete {
				break
			}
			if err = selection.SetSuccessPercentage(ctx); err != nil {
				errs = append(errs, err)
				break
			}
			hasMoreSelections, err := selector.HasMoreSelections(ctx)
			if err != nil {
				errs = append(errs, err)
				break
			}
			if !hasMoreSelections {
				break
			}
			if err = selector.Advance(ctx); err != nil {
				errs = append(errs, err)
				break
			}
			selection, err = selector.CurrentSelection(ctx)
			if err != nil {
				errs = append(errs, err)
				break
			}
		}
	}
	if err = errors.Join(errs...); err != nil {
		r.log.WithError(err).Error("reconciliation errors:")
	}
}
