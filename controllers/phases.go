package controllers

import ctrl "sigs.k8s.io/controller-runtime"

type PhaseFunction func(*MachinePoolRequest) (ctrl.Result, error)
