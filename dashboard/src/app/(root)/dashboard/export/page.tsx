'use client';

import { useState } from 'react';
import { Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from '@/components/ui/use-toast';
import { getErrorMessage } from '@/lib/error-message';
import { api } from '@/utils/api';
import { EXPORT_ENDPOINTS } from '@/utils/api-endpoints';

// 与服务端 format 词表一致 (server/api/export.go)。
const FORMATS = [
  { value: 'json', label: 'JSON', ext: 'json' },
  { value: 'csv', label: 'CSV', ext: 'csv' },
  { value: 'schema', label: 'Pydantic Schema', ext: 'schema.json' },
] as const;

type FormatValue = (typeof FORMATS)[number]['value'];

const SAMPLE_OUTPUTS: Record<FormatValue, string> = {
  json: `{
  "memories": [
    {
      "id": "mem_abc123",
      "content": "User prefers dark mode",
      "user_id": "user_1",
      "created_at": "2026-04-10T12:00:00Z"
    }
  ],
  "total": 1247
}`,
  csv: `id,content,user_id,agent_id,run_id,category,created_at
mem_abc123,User prefers dark mode,user_1,,,,
mem_def456,Likes hiking,user_1,,,food,`,
  schema: `{
  "title": "MemoryItem",
  "type": "object",
  "properties": {
    "id": {"type": "string", "format": "uuid"},
    "content": {"type": "string"},
    "user_id": {"anyOf": [{"type": "string"}, {"type": "null"}]},
    "created_at": {"type": "string", "format": "date-time"}
  },
  "required": ["id", "content", "created_at"]
}`,
};

// 客户端拼文件名 (跨域响应头 Content-Disposition 不保证可读)。
function buildFilename(format: FormatValue): string {
  const ext = FORMATS.find((f) => f.value === format)?.ext ?? 'json';
  return `memgo-export-${new Date().toISOString().slice(0, 10).replace(/-/g, '')}.${ext}`;
}

export default function ExportPage() {
  const [format, setFormat] = useState<FormatValue>('json');
  const [userId, setUserId] = useState('');
  const [agentId, setAgentId] = useState('');
  const [runId, setRunId] = useState('');
  const [isExporting, setIsExporting] = useState(false);

  const handleExport = async () => {
    setIsExporting(true);
    try {
      const params = new URLSearchParams({ format });
      if (userId.trim()) {
        params.set('user_id', userId.trim());
      }
      if (agentId.trim()) {
        params.set('agent_id', agentId.trim());
      }
      if (runId.trim()) {
        params.set('run_id', runId.trim());
      }
      const res = await api.get(
        `${EXPORT_ENDPOINTS.BASE}?${params.toString()}`,
        {
          responseType: 'blob',
        },
      );
      const url = URL.createObjectURL(res.data as Blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = buildFilename(format);
      link.click();
      URL.revokeObjectURL(url);
      toast({ title: 'Export downloaded', variant: 'success' });
    } catch (error) {
      toast({
        title: 'Failed to export memories',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="font-fustat text-xl font-semibold">Export</h1>
        <p className="text-sm text-onSurface-default-secondary">
          Export your memories in JSON, CSV, or Pydantic schema format.
        </p>
      </div>

      <Card className="border-memBorder-primary">
        <CardContent className="space-y-4 p-6">
          <div className="space-y-2">
            <Label>Format</Label>
            <div className="flex gap-2">
              {FORMATS.map((item) => (
                <Button
                  key={item.value}
                  variant={format === item.value ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setFormat(item.value)}
                >
                  {item.label}
                </Button>
              ))}
            </div>
          </div>

          <div className="space-y-2">
            <Label>Scope (optional)</Label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <Input
                placeholder="user_id"
                value={userId}
                onChange={(e) => setUserId(e.target.value)}
              />
              <Input
                placeholder="agent_id"
                value={agentId}
                onChange={(e) => setAgentId(e.target.value)}
              />
              <Input
                placeholder="run_id"
                value={runId}
                onChange={(e) => setRunId(e.target.value)}
              />
            </div>
            <p className="text-xs text-onSurface-default-tertiary">
              Leave all scopes empty to export every memory (admin only).
            </p>
          </div>

          <div className="space-y-2">
            <Label>Sample output</Label>
            <Card className="border-memBorder-primary bg-surface-default-secondary">
              <CardContent className="p-3">
                <pre className="overflow-x-auto whitespace-pre-wrap font-mono text-xs text-onSurface-default-secondary">
                  {SAMPLE_OUTPUTS[format]}
                </pre>
              </CardContent>
            </Card>
          </div>

          <Button onClick={handleExport} disabled={isExporting}>
            <Download className="size-4" />
            {isExporting ? 'Exporting...' : 'Export Memories'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
