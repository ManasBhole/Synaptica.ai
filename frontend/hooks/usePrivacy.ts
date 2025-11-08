'use client';

import { useQuery } from "@tanstack/react-query";
import { DLPStatsResponse, fetchDLPStats } from "../lib/api";

const fallbackDLP: DLPStatsResponse = {
  todayFailed: 2,
  todayAccepted: 58,
  tokenVaultSize: 412,
  topReasons: [
    { reason: "PII detection: SSN", count: 1 },
    { reason: "Unknown reason", count: 1 }
  ],
  recentIncidents: [
    {
      id: "demo-dlp-1",
      source: "hospital",
      format: "FHIR",
      status: "failed",
      error: "PII detection: SSN",
      retryCount: 1,
      createdAt: new Date(Date.now() - 20 * 60_000).toISOString(),
      updatedAt: new Date(Date.now() - 5 * 60_000).toISOString()
    }
  ]
};

export const useDLPStats = () =>
  useQuery({
    queryKey: ["dlp-stats"],
    queryFn: fetchDLPStats,
    refetchInterval: 20_000,
    placeholderData: fallbackDLP
  });
