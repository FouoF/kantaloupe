/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum PersistentVolumeMode {
  PERSISTENT_VOLUME_MODE_UNSPECIFIED = "PERSISTENT_VOLUME_MODE_UNSPECIFIED",
  Block = "Block",
  Filesystem = "Filesystem",
}

export enum PersistentVolumeReclaimPolicy {
  PERSISTENT_VOLUME_RECLAIM_POLICY_UNSPECIFIED = "PERSISTENT_VOLUME_RECLAIM_POLICY_UNSPECIFIED",
  Recycle = "Recycle",
  Delete = "Delete",
  Retain = "Retain",
}

export enum PersistentVolumeAccessMode {
  PERSISTENT_VOLUME_ACCESS_MODE_UNSPECIFIED = "PERSISTENT_VOLUME_ACCESS_MODE_UNSPECIFIED",
  ReadWriteOnce = "ReadWriteOnce",
  ReadOnlyMany = "ReadOnlyMany",
  ReadWriteMany = "ReadWriteMany",
  ReadWriteOncePod = "ReadWriteOncePod",
}

export enum Phase {
  PHASE_UNSPECIFIED = "PHASE_UNSPECIFIED",
  Pending = "Pending",
  Available = "Available",
  Bound = "Bound",
  Released = "Released",
  Failed = "Failed",
}

export type PersistentVolume = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  spec?: PersistentVolumeSpec
  status?: PersistentVolumeStatus
}

export type PersistentVolumeSpec = {
  capacity?: string
  accessModes?: PersistentVolumeAccessMode[]
  persistentVolumeReclaimPolicy?: PersistentVolumeReclaimPolicy
  mountOptions?: string[]
  volumeMode?: PersistentVolumeMode
  persistentVolumeSource?: PersistentVolumeSource
  storageClassName?: string
}

export type PersistentVolumeSource = {
  hostPath?: HostPathVolumeSource
  nfs?: NFSVolumeSource
  local?: LocalVolumeSource
}

export type NFSVolumeSource = {
  server?: string
  path?: string
  readOnly?: boolean
}

export type HostPathVolumeSource = {
  path?: string
  type?: string
}

export type LocalVolumeSource = {
  path?: string
  fsType?: string
}

export type PersistentVolumeStatus = {
  phase?: Phase
  message?: string
  reason?: string
}

export type ListPersistentVolumesRequest = {
  cluster?: string
  name?: string
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  labelSelector?: string
  fieldSelector?: string
  fuzzyName?: string
}

export type ListPersistentVolumesResponse = {
  items?: PersistentVolume[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type GetPersistentVolumeRequest = {
  cluster?: string
  name?: string
}

export type GetPersistentVolumeJSONRequest = {
  cluster?: string
  name?: string
}

export type GetPersistentVolumeJSONResponse = {
  data?: string
}

export type GetPersistentVolumeResponse = {
  data?: PersistentVolume
}

export type CreatePersistentVolumeRequest = {
  cluster?: string
  data?: PersistentVolume
}

export type CreatePersistentVolumeResponse = {
  data?: PersistentVolume
}

export type UpdatePersistentVolumeRequest = {
  cluster?: string
  name?: string
  data?: PersistentVolume
}

export type UpdatePersistentVolumeResponse = {
  data?: PersistentVolume
}

export type DeletePersistentVolumeRequest = {
  cluster?: string
  name?: string
}