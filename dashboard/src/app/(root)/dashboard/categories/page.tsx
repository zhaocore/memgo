'use client';

import { useEffect, useRef, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { toast } from '@/components/ui/use-toast';
import { getErrorMessage } from '@/lib/error-message';
import { api } from '@/utils/api';
import { MEMORY_ENDPOINTS } from '@/utils/api-endpoints';
import { useAuth } from '@/hooks/use-auth';
import { useApiQuery } from '@/hooks/use-api-query';

// 编辑态条目 (id 仅作 React key, name 唯一性由服务端语义保证——重复名不报错但无意义)。
type CategoryEntry = {
  id: string;
  name: string;
  description: string;
};

// wire 形态 [{"分类名": "描述"}] → 编辑条目。忽略非法形状 (非对象/多键/非字符串描述),
// 服务端保存时会再做完整校验。
function parseCategories(raw: unknown): CategoryEntry[] {
  if (!Array.isArray(raw)) {
    return [];
  }
  const out: CategoryEntry[] = [];
  for (const item of raw) {
    if (typeof item !== 'object' || item === null) {
      continue;
    }
    const record = item as Record<string, unknown>;
    const keys = Object.keys(record);
    if (keys.length !== 1) {
      continue;
    }
    const name = keys[0];
    const description = record[name];
    if (typeof description !== 'string' || !name.trim()) {
      continue;
    }
    out.push({ id: `${name}-${out.length}`, name, description });
  }
  return out;
}

// 编辑条目 → wire 形态 (name 去首尾空白)。
function toWire(entries: CategoryEntry[]) {
  return entries.map((entry) => ({
    [entry.name.trim()]: entry.description,
  }));
}

export default function CategoriesPage() {
  const { isAdmin } = useAuth();
  const [entries, setEntries] = useState<CategoryEntry[]>([]);
  const [savedSnapshot, setSavedSnapshot] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const newIdCounter = useRef(0);

  const { data, isLoading, refetch } = useApiQuery<
    Record<string, unknown> | undefined
  >(
    async () => {
      const res = await api.get(MEMORY_ENDPOINTS.CONFIGURE);
      return res.data as Record<string, unknown> | undefined;
    },
    { errorToast: 'Failed to load server configuration' },
  );

  useEffect(() => {
    // 服务端数据变化 (首载/保存后 refetch) → 回填编辑态。
    const parsed = parseCategories(data?.custom_categories);
    setEntries(parsed);
    setSavedSnapshot(JSON.stringify(toWire(parsed)));
  }, [data]);

  const dirty = JSON.stringify(toWire(entries)) !== savedSnapshot;
  const hasEmptyName = entries.some((entry) => !entry.name.trim());

  const handleAdd = () => {
    newIdCounter.current += 1;
    setEntries((prev) => [
      ...prev,
      { id: `new-${newIdCounter.current}`, name: '', description: '' },
    ]);
  };

  const handleRemove = (id: string) => {
    setEntries((prev) => prev.filter((entry) => entry.id !== id));
  };

  const handleSave = async () => {
    if (hasEmptyName) {
      toast({
        title: 'Every category needs a name',
        variant: 'destructive',
      });
      return;
    }
    setIsSaving(true);
    try {
      // 空列表 = 显式关闭打标 (服务端深合并对数组整体替换)。
      await api.post(MEMORY_ENDPOINTS.CONFIGURE, {
        custom_categories: toWire(entries),
      });
      toast({ title: 'Categories saved', variant: 'success' });
      await refetch();
    } catch (error) {
      toast({
        title: 'Failed to save categories',
        description: getErrorMessage(error),
        variant: 'destructive',
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="text-xl font-semibold font-fustat">Custom Categories</h1>
        <p className="text-sm text-onSurface-default-secondary">
          Memories added with infer are tagged with the closest matching
          category from this catalog. Changing the list only affects new
          memories — existing memories keep their tag.
        </p>
      </div>

      <Card className="border-memBorder-primary">
        <CardHeader>
          <CardTitle className="text-sm">
            Active Categories
            {entries.length > 0 && (
              <span className="ml-2 font-normal text-onSurface-default-tertiary">
                ({entries.length})
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <p className="text-sm text-onSurface-default-tertiary">
              Loading categories...
            </p>
          ) : entries.length === 0 ? (
            <div className="rounded-md border border-dashed border-memBorder-primary p-4">
              <p className="text-sm text-onSurface-default-secondary">
                No categories configured — memory tagging is off. New memories
                are stored without a category.
              </p>
            </div>
          ) : (
            entries.map((entry) => (
              <div
                key={entry.id}
                className="grid grid-cols-1 gap-3 sm:grid-cols-[minmax(0,200px)_1fr_auto] sm:items-end"
              >
                <div className="space-y-1">
                  <Label className="text-xs">Name</Label>
                  <Input
                    placeholder="category_name"
                    value={entry.name}
                    onChange={(e) => {
                      const name = e.target.value;
                      setEntries((prev) =>
                        prev.map((item) =>
                          item.id === entry.id ? { ...item, name } : item,
                        ),
                      );
                    }}
                    disabled={!isAdmin}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Description</Label>
                  <Input
                    placeholder="What this category covers (helps the LLM match)"
                    value={entry.description}
                    onChange={(e) => {
                      const description = e.target.value;
                      setEntries((prev) =>
                        prev.map((item) =>
                          item.id === entry.id
                            ? { ...item, description }
                            : item,
                        ),
                      );
                    }}
                    disabled={!isAdmin}
                  />
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={`Remove category ${entry.name || 'unnamed'}`}
                  onClick={() => handleRemove(entry.id)}
                  disabled={!isAdmin}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))
          )}

          {isAdmin && (
            <Button variant="outline" onClick={handleAdd}>
              <Plus className="size-4" />
              Add category
            </Button>
          )}
        </CardContent>
      </Card>

      <div className="space-y-1 text-xs text-onSurface-default-tertiary">
        <p>
          An empty list disables tagging entirely. Tags are assigned by the LLM
          at ingestion time.
        </p>
        <p>
          Per-call override: pass{' '}
          <code className="rounded bg-surface-default-secondary px-1">
            custom_categories
          </code>{' '}
          on{' '}
          <code className="rounded bg-surface-default-secondary px-1">
            POST /memories
          </code>{' '}
          to fully replace this catalog for that call (lists are not merged).
        </p>
        {!isAdmin && <p>Admin role required to edit the category catalog.</p>}
      </div>

      {isAdmin && (
        <Button
          onClick={handleSave}
          disabled={isSaving || !dirty || hasEmptyName}
        >
          {isSaving ? 'Saving...' : 'Save Categories'}
        </Button>
      )}
    </div>
  );
}
