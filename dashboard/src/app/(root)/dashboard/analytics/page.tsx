'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuth } from '@/hooks/use-auth';
import { useApiQuery } from '@/hooks/use-api-query';
import { api } from '@/utils/api';
import { ANALYTICS_ENDPOINTS } from '@/utils/api-endpoints';

const DAYS = 14;

type DailyBucket = {
  date: string;
  count: number;
};

type AnalyticsSummary = {
  total_operations: number;
  avg_latency_ms: number;
  success_rate: number;
  operations_over_time: DailyBucket[];
};

function StatTile({ label, value }: { label: string; value: string }) {
  return (
    <Card className="border-memBorder-primary">
      <CardContent className="p-4">
        <p className="text-xs text-onSurface-default-tertiary">{label}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

export default function AnalyticsPage() {
  const { isAdmin } = useAuth();
  const { data, isLoading } = useApiQuery<AnalyticsSummary>(
    async () => {
      const res = await api.get<AnalyticsSummary>(
        `${ANALYTICS_ENDPOINTS.BASE}?days=${DAYS}`,
      );
      return res.data;
    },
    { errorToast: 'Failed to load analytics' },
  );

  if (!isAdmin) {
    return (
      <div className="space-y-6">
        <div className="space-y-1">
          <h1 className="font-fustat text-xl font-semibold">Analytics</h1>
          <p className="text-sm text-onSurface-default-secondary">
            Track memory operations, latency, and usage patterns over time.
          </p>
        </div>
        <Card className="border-memBorder-primary">
          <CardContent className="p-4">
            <p className="text-sm text-onSurface-default-secondary">
              Admin role required to view analytics.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  const summary = data;
  const hasData = (summary?.total_operations ?? 0) > 0;
  const maxCount = Math.max(
    1,
    ...(summary?.operations_over_time ?? []).map((b) => b.count),
  );

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="font-fustat text-xl font-semibold">Analytics</h1>
        <p className="text-sm text-onSurface-default-secondary">
          Memory operations, latency, and success rate over the last {DAYS}{' '}
          days.
        </p>
      </div>

      {isLoading ? (
        <p className="text-sm text-onSurface-default-tertiary">
          Loading analytics...
        </p>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <StatTile
              label="Total Operations"
              value={String(summary?.total_operations ?? 0)}
            />
            <StatTile
              label="Avg Latency"
              value={hasData ? `${summary?.avg_latency_ms ?? 0}ms` : '—'}
            />
            <StatTile
              label="Success Rate"
              value={hasData ? `${summary?.success_rate ?? 0}%` : '—'}
            />
          </div>

          <Card className="border-memBorder-primary">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm">Operations over time</CardTitle>
            </CardHeader>
            <CardContent className="p-4">
              <div className="flex h-[200px] items-end gap-1">
                {(summary?.operations_over_time ?? []).map((bucket) => (
                  <div
                    key={bucket.date}
                    title={`${bucket.date}: ${bucket.count}`}
                    className="h-full flex-1"
                  >
                    <div
                      className="rounded-t bg-surface-default-brand"
                      style={{
                        height: `${(bucket.count / maxCount) * 100}%`,
                        minHeight: bucket.count > 0 ? '2px' : undefined,
                      }}
                    />
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
