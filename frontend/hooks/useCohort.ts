'use client';

import { useMutation } from "@tanstack/react-query";
import { CohortQueryPayload, CohortResult, runCohortQuery, verifyCohortDSL } from "../lib/api";

export const useCohortQuery = () =>
  useMutation<CohortResult, Error, CohortQueryPayload>({
    mutationKey: ["cohort-query"],
    mutationFn: runCohortQuery
  });

export const useCohortVerify = () =>
  useMutation<{ status: string }, Error, string>({
    mutationKey: ["cohort-verify"],
    mutationFn: (dsl) => verifyCohortDSL(dsl)
  });
