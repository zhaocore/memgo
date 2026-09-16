'use client';

import { useState } from 'react';
import { Pencil, Plus, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from '@/components/ui/use-toast';
import { getErrorMessage } from '@/lib/error-message';
import { api } from '@/utils/api';
import { WEBHOOK_ENDPOINTS } from '@/utils/api-endpoints';
import { useAuth } from '@/hooks/use-auth';
import { useApiQuery } from '@/hooks/use-api-query';

// 与服务端词表一致 (server/webhook.AllEventTypes)。
const EVENT_TYPES = [
  'memory_add',
  'memory_update',
  'memory_delete',
  'memory_categorize',
] as const;

type EventType = (typeof EVENT_TYPES)[number];

type Webhook = {
  id: string;
  name: string;
  url: string;
  event_types: EventType[];
  created_at: string;
};

// 编辑态草稿 (id 为空表示新建表单)。
type WebhookDraft = {
  id: string;
  name: string;
  url: string;
  eventTypes: EventType[];
};

const emptyDraft = (): WebhookDraft => ({
  id: '',
  name: '',
  url: '',
  eventTypes: [...EVENT_TYPES],
});

function EventCheckboxes({
  selected,
  onToggle,
  disabled,
}: {
  selected: EventType[];
  onToggle: (event: EventType) => void;
  disabled: boolean;
}) {
  return (
    <div className="flex flex-col gap-2">
      {EVENT_TYPES.map((event) => (
        <div key={event} className="flex items-center gap-2">
          <Checkbox
            id={`event-${event}-${selected.join(',')}`}
            checked={selected.includes(event)}
            onCheckedChange={() => onToggle(event)}
            disabled={disabled}
          />
          <Label
            htmlFor={`event-${event}-${selected.join(',')}`}
            className="text-sm font-normal"
          >
            {event}
          </Label>
        </div>
      ))}
    </div>
  );
}

export default function WebhooksPage() {
  const { isAdmin } = useAuth();
  const { data, isLoading, refetch } = useApiQuery<Webhook[]>(
    async () => {
      const res = await api.get<Webhook[]>(WEBHOOK_ENDPOINTS.BASE);
      return res.data;
    },
    { errorToast: 'Failed to load webhooks' },
  );
  const [draft, setDraft] = useState<WebhookDraft>(emptyDraft());
  const [editingId, setEditingId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const webhooks = data ?? [];

  const handleCreate = async () => {
    if (!draft.url.trim()) {
      toast({ title: 'Endpoint URL is required', variant: 'destructive' });
      return;
    }
    setBusy(true);
    try {
      await api.post(WEBHOOK_ENDPOINTS.BASE, {
        url: draft.url.trim(),
        name: draft.name.trim(),
        event_types: draft.eventTypes,
      });
      toast({ title: 'Webhook created', variant: 'success' });
      setDraft(emptyDraft());
      await refetch();
    } catch (error) {
      toast({
        title: 'Failed to create webhook',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setBusy(false);
    }
  };

  const handleUpdate = async (id: string) => {
    setBusy(true);
    try {
      await api.put(WEBHOOK_ENDPOINTS.BY_ID(id), {
        url: draft.url.trim(),
        name: draft.name.trim(),
        event_types: draft.eventTypes,
      });
      toast({ title: 'Webhook updated', variant: 'success' });
      setEditingId(null);
      await refetch();
    } catch (error) {
      toast({
        title: 'Failed to update webhook',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setBusy(false);
    }
  };

  const handleDelete = async (id: string) => {
    setBusy(true);
    try {
      await api.delete(WEBHOOK_ENDPOINTS.BY_ID(id));
      toast({ title: 'Webhook deleted', variant: 'success' });
      await refetch();
    } catch (error) {
      toast({
        title: 'Failed to delete webhook',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setBusy(false);
    }
  };

  const toggleEvent = (
    event: EventType,
    source: WebhookDraft,
    onChange: (next: WebhookDraft) => void,
  ) => {
    const selected = source.eventTypes.includes(event)
      ? source.eventTypes.filter((item) => item !== event)
      : [...source.eventTypes, event];
    onChange({ ...source, eventTypes: selected });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="text-xl font-semibold font-fustat">Webhooks</h1>
        <p className="text-sm text-onSurface-default-secondary">
          Real-time event notifications for memory operations. Registered
          endpoints receive an HTTP POST for each subscribed event.
        </p>
      </div>

      {isAdmin && (
        <Card className="border-memBorder-primary">
          <CardHeader>
            <CardTitle className="text-sm">Create Webhook</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label className="text-xs">Endpoint URL</Label>
                <Input
                  placeholder="https://your-app.com/webhooks/memgo"
                  value={draft.url}
                  onChange={(e) => setDraft({ ...draft, url: e.target.value })}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Name</Label>
                <Input
                  placeholder="Memory Logger"
                  value={draft.name}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                />
              </div>
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Events</Label>
              <EventCheckboxes
                selected={draft.eventTypes}
                onToggle={(event) => toggleEvent(event, draft, setDraft)}
                disabled={false}
              />
            </div>
            <Button onClick={handleCreate} disabled={busy}>
              <Plus className="size-4" />
              Create Webhook
            </Button>
          </CardContent>
        </Card>
      )}

      <Card className="border-memBorder-primary">
        <CardHeader>
          <CardTitle className="text-sm">
            Registered Webhooks
            {webhooks.length > 0 && (
              <span className="ml-2 font-normal text-onSurface-default-tertiary">
                ({webhooks.length})
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <p className="text-sm text-onSurface-default-tertiary">
              Loading webhooks...
            </p>
          ) : webhooks.length === 0 ? (
            <div className="rounded-md border border-dashed border-memBorder-primary p-4">
              <p className="text-sm text-onSurface-default-secondary">
                No webhooks registered — event notifications are off.
              </p>
            </div>
          ) : (
            webhooks.map((hook) =>
              editingId === hook.id ? (
                <div
                  key={hook.id}
                  className="space-y-3 rounded-md border border-memBorder-primary p-3"
                >
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <div className="space-y-1">
                      <Label className="text-xs">Endpoint URL</Label>
                      <Input
                        value={draft.url}
                        onChange={(e) =>
                          setDraft({ ...draft, url: e.target.value })
                        }
                      />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-xs">Name</Label>
                      <Input
                        value={draft.name}
                        onChange={(e) =>
                          setDraft({ ...draft, name: e.target.value })
                        }
                      />
                    </div>
                  </div>
                  <EventCheckboxes
                    selected={draft.eventTypes}
                    onToggle={(event) => toggleEvent(event, draft, setDraft)}
                    disabled={false}
                  />
                  <div className="flex gap-2">
                    <Button
                      onClick={() => handleUpdate(hook.id)}
                      disabled={busy}
                    >
                      Save
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => setEditingId(null)}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : (
                <div
                  key={hook.id}
                  className="flex flex-col gap-2 rounded-md border border-memBorder-primary p-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="min-w-0 space-y-1">
                    <p className="truncate text-sm font-medium">
                      {hook.name || 'Unnamed webhook'}
                    </p>
                    <p className="truncate text-xs text-onSurface-default-tertiary">
                      {hook.url}
                    </p>
                    <div className="flex flex-wrap gap-1">
                      {hook.event_types.map((event) => (
                        <Badge
                          key={event}
                          variant="secondary"
                          className="text-xs"
                        >
                          {event}
                        </Badge>
                      ))}
                    </div>
                  </div>
                  {isAdmin && (
                    <div className="flex shrink-0 gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Edit webhook ${hook.name || hook.id}`}
                        onClick={() => {
                          setEditingId(hook.id);
                          setDraft({
                            id: hook.id,
                            name: hook.name,
                            url: hook.url,
                            eventTypes: [...hook.event_types],
                          });
                        }}
                      >
                        <Pencil className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Delete webhook ${hook.name || hook.id}`}
                        onClick={() => handleDelete(hook.id)}
                        disabled={busy}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  )}
                </div>
              ),
            )
          )}
          {!isAdmin && (
            <p className="text-xs text-onSurface-default-tertiary">
              Admin role required to manage webhooks.
            </p>
          )}
        </CardContent>
      </Card>

      <div className="space-y-1 text-xs text-onSurface-default-tertiary">
        <p>
          Memory events carry the memory ID, content and event type (ADD /
          UPDATE / DELETE). Categorization events carry the memory ID and the
          assigned category.
        </p>
        <p>
          Deliveries are best-effort: a failing endpoint is logged and skipped,
          it never blocks memory operations.
        </p>
      </div>
    </div>
  );
}
