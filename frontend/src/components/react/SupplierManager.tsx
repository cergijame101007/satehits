import { useState, useCallback } from 'react';
import type { Supplier } from '../../types/supplier';
import { getSuppliers } from '../../mocks/supplier';
import Button from '@/components/react/ui/Button';
import Card from '@/components/react/ui/Card';
import Modal from '@/components/react/ui/Modal';
import Textarea from '@/components/react/ui/Textarea';
import { inputClassName } from '@/components/react/ui/inputStyles';
import { cx } from '@/lib/cx';

interface SupplierFormData {
  name: string;
  description: string;
  instagram_url: string;
  is_active: boolean;
}

const emptyForm: SupplierFormData = {
  name: '',
  description: '',
  instagram_url: '',
  is_active: true,
};

export default function SupplierManager() {
  const [suppliers, setSuppliers] = useState<Supplier[]>(() => getSuppliers());
  const [editingId, setEditingId] = useState<number | 'new' | null>(null);
  const [form, setForm] = useState<SupplierFormData>(emptyForm);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [draggedId, setDraggedId] = useState<number | null>(null);

  const openCreate = useCallback(() => {
    setEditingId('new');
    setForm(emptyForm);
    setErrors({});
  }, []);

  const openEdit = useCallback(
    (supplier: Supplier) => {
      setEditingId(supplier.id);
      setForm({
        name: supplier.name,
        description: supplier.description,
        instagram_url: supplier.instagram_url ?? '',
        is_active: supplier.is_active,
      });
      setErrors({});
    },
    [],
  );

  const closeModal = useCallback(() => {
    setEditingId(null);
    setForm(emptyForm);
    setErrors({});
  }, []);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!form.name.trim()) newErrors.name = '取引先名を入力してください';
    if (!form.description.trim()) newErrors.description = '説明文を入力してください';
    if (
      form.instagram_url.trim() &&
      !form.instagram_url.match(/^https?:\/\/(www\.)?instagram\.com\//)
    ) {
      newErrors.instagram_url = '有効なInstagramのURLを入力してください';
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validate()) return;
    setSubmitting(true);

    // TODO: POST or PUT /api/v1/admin/suppliers に置き換え
    await new Promise((r) => setTimeout(r, 400));

    if (editingId === 'new') {
      const newId = Math.max(...suppliers.map((s) => s.id), 0) + 1;
      const maxOrder = Math.max(...suppliers.map((s) => s.display_order), 0);
      const newSupplier: Supplier = {
        id: newId,
        name: form.name.trim(),
        description: form.description.trim(),
        instagram_url: form.instagram_url.trim() || null,
        image_url: null,
        display_order: maxOrder + 1,
        is_active: form.is_active,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      setSuppliers((prev) => [...prev, newSupplier]);
      console.log('[Mock] 取引先を追加:', newSupplier);
    } else if (typeof editingId === 'number') {
      setSuppliers((prev) =>
        prev.map((s) =>
          s.id === editingId
            ? {
                ...s,
                name: form.name.trim(),
                description: form.description.trim(),
                instagram_url: form.instagram_url.trim() || null,
                is_active: form.is_active,
                updated_at: new Date().toISOString(),
              }
            : s,
        ),
      );
      console.log('[Mock] 取引先を更新:', editingId);
    }

    setSubmitting(false);
    closeModal();
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('この取引先を削除しますか？')) return;

    // TODO: DELETE /api/v1/admin/suppliers/:id に置き換え
    await new Promise((r) => setTimeout(r, 300));
    setSuppliers((prev) => prev.filter((s) => s.id !== id));
    console.log('[Mock] 取引先を削除:', id);
  };

  const handleDragStart = (id: number) => {
    setDraggedId(id);
  };

  const handleDragOver = (e: React.DragEvent, targetId: number) => {
    e.preventDefault();
    if (draggedId === null || draggedId === targetId) return;

    setSuppliers((prev) => {
      const sorted = [...prev].sort((a, b) => a.display_order - b.display_order);
      const dragIdx = sorted.findIndex((s) => s.id === draggedId);
      const targetIdx = sorted.findIndex((s) => s.id === targetId);
      if (dragIdx === -1 || targetIdx === -1) return prev;

      const [moved] = sorted.splice(dragIdx, 1);
      sorted.splice(targetIdx, 0, moved);

      return sorted.map((s, i) => ({ ...s, display_order: i + 1 }));
    });
  };

  const handleDragEnd = () => {
    setDraggedId(null);
    // TODO: PUT /api/v1/admin/suppliers/order に置き換え
    console.log(
      '[Mock] 並び替え:',
      suppliers.map((s) => ({ id: s.id, order: s.display_order })),
    );
  };

  const sorted = [...suppliers].sort((a, b) => a.display_order - b.display_order);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-medium">取引先管理</h1>
      </div>

      <button
        type="button"
        onClick={openCreate}
        className="w-full py-3 border-2 border-dashed border-gray-300 rounded-xl text-gray-500 hover:border-primary/50 hover:text-primary transition-colors font-medium"
      >
        + 取引先を追加
      </button>

      <div className="space-y-3">
        {sorted.map((supplier) => (
          <Card
            key={supplier.id}
            compact
            className={cx(
              'transition-all',
              draggedId === supplier.id && 'border-primary shadow-md opacity-70',
            )}
          >
            <div
              draggable
              onDragStart={() => handleDragStart(supplier.id)}
              onDragOver={(e) => handleDragOver(e, supplier.id)}
              onDragEnd={handleDragEnd}
              className="flex items-start gap-3"
            >
              {/* ドラッグハンドル */}
              <div className="mt-1 cursor-grab text-gray-300 hover:text-gray-500 active:cursor-grabbing">
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M7 2a2 2 0 10.001 4.001A2 2 0 007 2zm0 6a2 2 0 10.001 4.001A2 2 0 007 8zm0 6a2 2 0 10.001 4.001A2 2 0 007 14zm6-8a2 2 0 10-.001-4.001A2 2 0 0013 6zm0 2a2 2 0 10.001 4.001A2 2 0 0013 8zm0 6a2 2 0 10.001 4.001A2 2 0 0013 14z" />
                </svg>
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="font-medium text-lg">{supplier.name}</h3>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full ${
                      supplier.is_active
                        ? 'bg-green-100 text-green-700'
                        : 'bg-gray-100 text-gray-500'
                    }`}
                  >
                    {supplier.is_active ? '表示中' : '非表示'}
                  </span>
                </div>
                <p className="text-sm text-gray-600 line-clamp-2">{supplier.description}</p>
                {supplier.instagram_url && (
                  <a
                    href={supplier.instagram_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 mt-1 text-xs text-primary hover:underline"
                  >
                    <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 100 12.324 6.162 6.162 0 000-12.324zM12 16a4 4 0 110-8 4 4 0 010 8zm6.406-11.845a1.44 1.44 0 100 2.881 1.44 1.44 0 000-2.881z" />
                    </svg>
                    Instagram
                  </a>
                )}
              </div>

              {/* アクションボタン */}
              <div className="flex items-center gap-2 shrink-0">
                <Button variant="ghost" size="sm" onClick={() => openEdit(supplier)}>
                  編集
                </Button>
                <Button variant="ghost" size="sm" className="text-red-500 hover:border-red-300 hover:bg-red-50" onClick={() => handleDelete(supplier.id)}>
                  削除
                </Button>
              </div>
            </div>
          </Card>
        ))}

        {sorted.length === 0 && (
          <div className="text-center py-12 text-gray-400">
            <p>取引先が登録されていません</p>
          </div>
        )}
      </div>

      <p className="text-xs text-gray-400 text-center">
        ※ ドラッグで並び替え可能
      </p>

      <Modal
        open={editingId !== null}
        title={editingId === 'new' ? '取引先を追加' : '取引先を編集'}
        onClose={closeModal}
        size="lg"
        closeDisabled={submitting}
        footer={
          <div className="flex items-center gap-3 p-5">
            <Button variant="ghost" size="md" fullWidth className="rounded-xl py-3" onClick={closeModal} disabled={submitting}>
              キャンセル
            </Button>
            <Button variant="primary" size="md" fullWidth className="rounded-xl py-3" onClick={handleSubmit} disabled={submitting}>
              {submitting ? '保存中...' : '保存する'}
            </Button>
          </div>
        }
      >
        <div className="space-y-5">
          <div>
            <label className="block text-sm font-medium mb-1.5">
              取引先名 <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              className={inputClassName(!!errors.name)}
              placeholder="例: 〇〇農園"
            />
            {errors.name && <p className="mt-1 text-sm text-red-500">{errors.name}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5">
              説明文 <span className="text-red-400">*</span>
            </label>
            <Textarea
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              rows={4}
              hasError={!!errors.description}
              placeholder="取引先の紹介文を入力"
            />
            {errors.description && <p className="mt-1 text-sm text-red-500">{errors.description}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5">Instagram URL</label>
            <input
              type="url"
              value={form.instagram_url}
              onChange={(e) => setForm((f) => ({ ...f, instagram_url: e.target.value }))}
              className={inputClassName(!!errors.instagram_url)}
              placeholder="https://instagram.com/..."
            />
            {errors.instagram_url && <p className="mt-1 text-sm text-red-500">{errors.instagram_url}</p>}
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5">画像</label>
            <div className="border-2 border-dashed border-gray-200 rounded-lg p-6 text-center text-gray-400">
              <svg className="w-8 h-8 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                />
              </svg>
              <p className="text-sm">画像アップロード（未実装）</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={form.is_active}
                onChange={(e) => setForm((f) => ({ ...f, is_active: e.target.checked }))}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:ring-2 peer-focus:ring-primary/30 after:content-[''] after:absolute after:top-0.5 after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
            </label>
            <span className="text-sm">サイトに表示する</span>
          </div>
        </div>
      </Modal>
    </div>
  );
}
