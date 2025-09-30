/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesPage from "../../types/page.pb"

export enum EventType {
  EVENT_TYPE_UNSPECIFIED = "EVENT_TYPE_UNSPECIFIED",
  Normal = "Normal",
  Warning = "Warning",
}

export enum ListEventsRequestKind {
  KIND_UNSPECIFIED = "KIND_UNSPECIFIED",
  Deployment = "Deployment",
  StatefulSet = "StatefulSet",
  DaemonSet = "DaemonSet",
  Pod = "Pod",
  Service = "Service",
  Ingress = "Ingress",
  Job = "Job",
  CronJob = "CronJob",
  HorizontalPodAutoscaler = "HorizontalPodAutoscaler",
  ReplicaSet = "ReplicaSet",
  CronHPA = "CronHPA",
  PersistentVolumeClaim = "PersistentVolumeClaim",
  GroupVersionResource = "GroupVersionResource",
}

export type Event = {
  involvedObject?: ObjectReference
  reason?: string
  message?: string
  source?: EventSource
  lastTimestamp?: string
  type?: EventType
  firstTimestamp?: string
}

export type ObjectReference = {
  kind?: string
  name?: string
  namespace?: string
  uid?: string
  apiVersion?: string
  resourceVersion?: string
}

export type EventSource = {
  component?: string
  host?: string
}

export type ListClusterEventsRequest = {
  cluster?: string
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  type?: EventType[]
  kind?: string
  name?: string
  namespace?: string
}

export type ListClusterEventsResponse = {
  items?: Event[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}

export type ListClusterEventKindsRequest = {
  cluster?: string
}

export type ListClusterEventKindsResponse = {
  data?: string[]
}

export type ListEventsRequest = {
  cluster?: string
  namespace?: string
  kind?: ListEventsRequestKind
  kindName?: string
  name?: string
  page?: number
  pageSize?: number
  sortOption?: KantaloupeDynamiaAiApiTypesPage.SortOption
  type?: EventType[]
  group?: string
  version?: string
  resource?: string
}

export type ListEventsResponse = {
  items?: Event[]
  pagination?: KantaloupeDynamiaAiApiTypesPage.Pagination
}