/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume from "./persistentvolume.pb"

export enum PVCPhase {
  PVC_PHASE_UNSPECIFIED = "PVC_PHASE_UNSPECIFIED",
  PVC_Pending = "PVC_Pending",
  PVC_Bound = "PVC_Bound",
  PVC_Lost = "PVC_Lost",
}

export enum PersistentVolumeClaimSpecVolumeMode {
  VOLUME_MODE_UNSPECIFIED = "VOLUME_MODE_UNSPECIFIED",
  Block = "Block",
  Filesystem = "Filesystem",
}

export type PersistentVolumeClaim = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: PersistentVolumeClaimSpec
  status?: PersistentVolumeClaimStatus
}

export type PersistentVolumeClaimSpec = {
  accessModes?: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.PersistentVolumeAccessMode[]
  selector?: KantaloupeDynamiaAiApiTypesObjectmeta.LabelSelector
  resources?: ResourceRequirements
  volumeName?: string
  storageClassName?: string
  volumeMode?: PersistentVolumeClaimSpecVolumeMode
  dataSource?: TypedLocalObjectReference
  dataSourceRef?: TypedObjectReference
  supportExpansion?: boolean
  supportSnapshot?: boolean
}

export type PersistentVolumeClaimStatus = {
  phase?: PVCPhase
  accessModes?: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.PersistentVolumeAccessMode[]
  capacity?: ResourceList
  conditions?: PersistentVolumeClaimCondition[]
  podName?: string[]
  snapshotCount?: number
}

export type PersistentVolumeClaimCondition = {
  type?: string
  status?: string
  lastProbeTime?: string
  lastTransitionTime?: string
  reason?: string
  message?: string
}

export type TypedLocalObjectReference = {
  apiGroup?: string
  kind?: string
  name?: string
}

export type TypedObjectReference = {
  apiGroup?: string
  kind?: string
  name?: string
  namespace?: string
}

export type ResourceList = {
  storage?: string
}

export type ResourceRequirements = {
  limits?: ResourceList
  requests?: ResourceList
}