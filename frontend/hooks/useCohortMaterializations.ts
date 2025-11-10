import { useQuery } from "@tanstack/react-query";
import { listCohortMaterializations, type CohortMaterialization } from "../lib/api";

const fallback: CohortMaterialization[] = [];

export function useCohortMaterializations(limit = 25) {
  return useQuery({
    queryKey: ["cohort-materializations", limit],
    queryFn: () => listCohortMaterializations(limit),
    initialData: fallback,
    refetchInterval: 15000
  });
}
