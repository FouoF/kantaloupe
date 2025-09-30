/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

export enum SortDir {
  desc = "desc",
  asc = "asc",
}

export enum SortBy {
  SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED",
  field_name = "field_name",
  created_at = "created_at",
}

export type Pagination = {
  total?: number
  page?: number
  pageSize?: number
  pages?: number
}

export type SortOption = {
  field?: string
  asc?: boolean
}