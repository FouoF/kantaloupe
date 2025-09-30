/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as KantaloupeDynamiaAiApiTypesObjectmeta from "../../types/objectmeta.pb"
export type ConfigMap = {
  metadata?: KantaloupeDynamiaAiApiTypesObjectmeta.ObjectMeta
  immutable?: boolean
  data?: {[key: string]: string}
  binaryData?: {[key: string]: Uint8Array}
}

export type GetConfigMapRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type GetConfigMapJSONRequest = {
  cluster?: string
  namespace?: string
  name?: string
}

export type GetConfigMapJSONResponse = {
  data?: string
}

export type UpdateConfigMapRequest = {
  cluster?: string
  namespace?: string
  name?: string
  data?: string
}

export type UpdateConfigMapResponse = {
  data?: string
}