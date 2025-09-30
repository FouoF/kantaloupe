/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume from "../../core/v1alpha1/persistentvolume.pb"
import * as KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolumeclaim from "../../core/v1alpha1/persistentvolumeclaim.pb"
import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum StorageType {
  StorageTypeUnspecified = "StorageTypeUnspecified",
  LocalPV = "LocalPV",
  NFS = "NFS",
  PVC = "PVC",
}

export type CreateStorageRequest = {
  cluster?: string
  storageType?: StorageType
  isManage?: boolean
  storageName?: string
  namespace?: string
  accessMode?: string
  storageSize?: string
  nfsServer?: string
  dataPath?: string
  localPath?: string
  nodeName?: string
  storageClassName?: string
}

export type Storage = {
  persistentVolumeClaim?: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolumeclaim.PersistentVolumeClaim
  storageType?: StorageType
  persistentVolume?: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolume.PersistentVolume
}

export type DeleteStorageRequest = {
  cluster?: string
  name?: string
  namespace?: string
}

export type ListStoragesRequest = {
  cluster?: string
  namespace?: string
  page?: number
  pageSize?: number
  name?: string
  phase?: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolumeclaim.PVCPhase
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  labelSelector?: string
  fieldSelector?: string
  fuzzyName?: string
  isManage?: boolean
  storageType?: StorageType
}

export type ListStoragesResponse = {
  items?: KantaloupeDynamiaAiApiCoreV1alpha1Persistentvolumeclaim.PersistentVolumeClaim[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}