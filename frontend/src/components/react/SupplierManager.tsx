import { useState, useCallback, useEffect, useRef } from 'react';
import type { Supplier } from '../../types/supplier';
import {
  createSupplier,
  deleteSupplier,
  listAdminSuppliers,
  reorderSuppliers,
  SupplierApiError,
  toSupplierErrorMessage,
  updateSupplier,
  uploadSupplierImage,
  validateSupplierImageFile,
} from '@/lib/suppliers';
import Alert from '@/components/react/ui/Alert';
import Button from '@/components/react/ui/Button';
import Card from '@/components/react/ui/Card';
import ConfirmModal from '@/components/react/ui/ConfirmModal';
import Modal from '@/components/react/ui/Modal';
import Textarea from '@/components/react/ui/Textarea';
import { inputClassName } from '@/lib/ui/inputStyles';
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
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [editingId, setEditingId] = useState<number | 'new' | null>(null);
  const [form, setForm] = useState<SupplierFormData>(emptyForm);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [draggedId, setDraggedId] = useState<number | null>(null);
  const [deleteTargetId, setDeleteTargetId] = useState<number | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreviewUrl, setImagePreviewUrl] = useState<string | null>(null);
  const [currentImageUrl, setCurrentImageUrl] = useState<string | null>(null);
  const suppliersRef = useRef<Supplier[]>([]);

  useEffect(() => {
    suppliersRef.current = suppliers;
  }, [suppliers]);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setLoadError('');

      try {
        const data = await listAdminSuppliers();
        if (!cancelled) {
          setSuppliers(data);
        }
      } catch (err) {
        if (!cancelled) {
          setSuppliers([]);
          setLoadError(toSupplierErrorMessage(err));
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const resetImageState = useCallback(() => {
    setImageFile(null);
    setImagePreviewUrl(null);
    setCurrentImageUrl(null);
  }, []);

  const openCreate = useCallback(() => {
    setEditingId('new');
    setForm(emptyForm);
    setErrors({});
    resetImageState();
  }, [resetImageState]);

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
      setImageFile(null);
      setImagePreviewUrl(null);
      setCurrentImageUrl(supplier.image_url);
    },
    [],
  );

  const closeModal = useCallback(() => {
    setEditingId(null);
    setForm(emptyForm);
    setErrors({});
    resetImageState();
  }, [resetImageState]);

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

  const applyApiFieldErrors = (err: SupplierApiError) => {
    if (!err.details?.length) {
      setErrors({ submit: err.message });
      return;
    }
    const fieldErrors: Record<string, string> = {};
    for (const detail of err.details) {
      fieldErrors[detail.field] = detail.message;
    }
    setErrors(fieldErrors);
  };

  const handleImageSelect = (file: File | null) => {
    if (!file) {
      setImageFile(null);
      setImagePreviewUrl(null);
      setErrors((prev) => {
        const next = { ...prev };
        delete next.image;
        return next;
      });
      return;
    }

    const message = validateSupplierImageFile(file);
    if (message) {
      setImageFile(null);
      setImagePreviewUrl(null);
      setErrors((prev) => ({ ...prev, image: message }));
      return;
    }

    setImageFile(file);
    setImagePreviewUrl(URL.createObjectURL(file));
    setErrors((prev) => {
      const next = { ...prev };
      delete next.image;
      return next;
    });
  };

  const handleSubmit = async () => {
    if (!validate()) return;
    setSubmitting(true);
    setErrors((prev) => {
      const next = { ...prev };
      delete next.submit;
      return next;
    });

    const instagramUrl = form.instagram_url.trim() || null;

    const isCreate = editingId === 'new';

    try {
      let saved: Supplier;

      if (isCreate) {
        saved = await createSupplier({
          name: form.name.trim(),
          description: form.description.trim(),
          instagram_url: instagramUrl,
          is_active: form.is_active,
        });
        // 画像アップロード失敗時の再保存で二重作成しないよう、直後に編集モードへ切り替える
        setEditingId(saved.id);
        setSuppliers((prev) => {
          const next = [...prev, saved];
          suppliersRef.current = next;
          return next;
        });
      } else if (typeof editingId === 'number') {
        saved = await updateSupplier(editingId, {
          name: form.name.trim(),
          description: form.description.trim(),
          instagram_url: instagramUrl,
          is_active: form.is_active,
        });
      } else {
        return;
      }

      if (imageFile) {
        const imageUrl = await uploadSupplierImage(saved.id, imageFile);
        saved = { ...saved, image_url: imageUrl };
      }

      setSuppliers((prev) => {
        const next = prev.map((s) => (s.id === saved.id ? saved : s));
        suppliersRef.current = next;
        return next;
      });
      closeModal();
    } catch (err) {
      if (err instanceof SupplierApiError) {
        applyApiFieldErrors(err);
      } else {
        setErrors({ submit: toSupplierErrorMessage(err) });
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteConfirm = async () => {
    if (deleteTargetId === null) return;
    setDeleting(true);

    try {
      await deleteSupplier(deleteTargetId);
      setSuppliers((prev) => prev.filter((s) => s.id !== deleteTargetId));
      setDeleteTargetId(null);
    } catch (err) {
      setLoadError(toSupplierErrorMessage(err));
      setDeleteTargetId(null);
    } finally {
      setDeleting(false);
    }
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

      const next = sorted.map((s, i) => ({ ...s, display_order: i + 1 }));
      suppliersRef.current = next;
      return next;
    });
  };

  const handleDragEnd = async () => {
    setDraggedId(null);
    const orderedIds = [...suppliersRef.current]
      .sort((a, b) => a.display_order - b.display_order)
      .map((s) => s.id);

    try {
      const updated = await reorderSuppliers(orderedIds);
      setSuppliers(updated);
    } catch (err) {
      setLoadError(toSupplierErrorMessage(err));
      try {
        const data = await listAdminSuppliers();
        setSuppliers(data);
      } catch {
        // 一覧復元も失敗した場合は loadError を維持
      }
    }
  };

  const sorted = [...suppliers].sort((a, b) => a.display_order - b.display_order);
  const previewUrl = imagePreviewUrl ?? currentImageUrl;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-medium">取引先管理</h1>
      </div>

      {loadError && <Alert variant="error">{loadError}</Alert>}

      <button
        type="button"
        onClick={openCreate}
        disabled={isLoading}
        className="w-full rounded-xl border-2 border-dashed border-gray-300 py-3 font-medium text-gray-500 transition-colors hover:border-primary/50 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50"
      >
        + 取引先を追加
      </button>

      {isLoading ? (
        <div className="py-12 text-center text-gray-400">読み込み中...</div>
      ) : (
        <div className="space-y-3">
          {sorted.map((supplier) => (
            <Card
              key={supplier.id}
              compact
              className={cx(
                'transition-all',
                draggedId === supplier.id && 'border-primary opacity-70 shadow-md',
              )}
            >
              <div
                draggable
                onDragStart={() => handleDragStart(supplier.id)}
                onDragOver={(e) => handleDragOver(e, supplier.id)}
                onDragEnd={handleDragEnd}
                className="flex items-start gap-3"
              >
                <div className="mt-1 cursor-grab text-gray-300 hover:text-gray-500 active:cursor-grabbing">
                  <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M7 2a2 2 0 10.001 4.001A2 2 0 007 2zm0 6a2 2 0 10.001 4.001A2 2 0 007 8zm0 6a2 2 0 10.001 4.001A2 2 0 007 14zm6-8a2 2 0 10-.001-4.001A2 2 0 0013 6zm0 2a2 2 0 10.001 4.001A2 2 0 0013 8zm0 6a2 2 0 10.001 4.001A2 2 0 0013 14z" />
                  </svg>
                </div>

                {supplier.image_url && (
                  <img
                    src={supplier.image_url}
                    alt=""
                    className="h-14 w-14 shrink-0 rounded-lg object-cover"
                  />
                )}

                <div className="min-w-0 flex-1">
                  <div className="mb-1 flex items-center gap-2">
                    <h3 className="text-lg font-medium">{supplier.name}</h3>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs ${
                        supplier.is_active
                          ? 'bg-green-100 text-green-700'
                          : 'bg-gray-100 text-gray-500'
                      }`}
                    >
                      {supplier.is_active ? '表示中' : '非表示'}
                    </span>
                  </div>
                  <p className="line-clamp-2 text-sm text-gray-600">{supplier.description}</p>
                  {supplier.instagram_url && (
                    <a
                      href={supplier.instagram_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="mt-1 inline-flex items-center gap-1 text-xs text-primary hover:underline"
                    >
                      <svg className="h-3.5 w-3.5" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 100 12.324 6.162 6.162 0 000-12.324zM12 16a4 4 0 110-8 4 4 0 010 8zm6.406-11.845a1.44 1.44 0 100 2.881 1.44 1.44 0 000-2.881z" />
                      </svg>
                      Instagram
                    </a>
                  )}
                </div>

                <div className="flex shrink-0 items-center gap-2">
                  <Button variant="ghost" size="sm" onClick={() => openEdit(supplier)}>
                    編集
                  </Button>
                  <Button variant="danger" size="sm" onClick={() => setDeleteTargetId(supplier.id)}>
                    削除
                  </Button>
                </div>
              </div>
            </Card>
          ))}

          {sorted.length === 0 && (
            <div className="py-12 text-center text-gray-400">
              <p>取引先が登録されていません</p>
            </div>
          )}
        </div>
      )}

      <p className="text-center text-xs text-gray-400">※ ドラッグで並び替え可能</p>

      <Modal
        open={editingId !== null}
        title={editingId === 'new' ? '取引先を追加' : '取引先を編集'}
        onClose={closeModal}
        size="lg"
        closeDisabled={submitting}
        footer={
          <div className="flex items-center gap-3 p-5">
            <Button variant="ghost" size="lg" onClick={closeModal} disabled={submitting}>
              キャンセル
            </Button>
            <Button variant="primary" size="lg" onClick={handleSubmit} disabled={submitting}>
              {submitting ? '保存中...' : '保存する'}
            </Button>
          </div>
        }
      >
        <div className="space-y-5">
          {errors.submit && <Alert variant="error">{errors.submit}</Alert>}

          <div>
            <label className="mb-1.5 block text-sm font-medium">
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
            <label className="mb-1.5 block text-sm font-medium">
              説明文 <span className="text-red-400">*</span>
            </label>
            <Textarea
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              rows={4}
              hasError={!!errors.description}
              placeholder="取引先の紹介文を入力"
            />
            {errors.description && (
              <p className="mt-1 text-sm text-red-500">{errors.description}</p>
            )}
          </div>

          <div>
            <label className="mb-1.5 block text-sm font-medium">Instagram URL</label>
            <input
              type="url"
              value={form.instagram_url}
              onChange={(e) => setForm((f) => ({ ...f, instagram_url: e.target.value }))}
              className={inputClassName(!!errors.instagram_url)}
              placeholder="https://instagram.com/..."
            />
            {errors.instagram_url && (
              <p className="mt-1 text-sm text-red-500">{errors.instagram_url}</p>
            )}
          </div>

          <div>
            <label className="mb-1.5 block text-sm font-medium">画像</label>
            <div className="space-y-3">
              {previewUrl && (
                <img
                  src={previewUrl}
                  alt="プレビュー"
                  className="mx-auto h-40 max-w-full rounded-lg object-cover"
                />
              )}
              <label className="flex cursor-pointer flex-col items-center rounded-lg border-2 border-dashed border-gray-200 p-6 text-center text-gray-400 transition-colors hover:border-primary/40 hover:text-primary">
                <svg
                  className="mx-auto mb-2 h-8 w-8"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                  />
                </svg>
                <span className="text-sm">JPEG / PNG / WebP（5MB以内）</span>
                <input
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  className="sr-only"
                  disabled={submitting}
                  onChange={(e) => handleImageSelect(e.target.files?.[0] ?? null)}
                />
              </label>
            </div>
            {errors.image && <p className="mt-1 text-sm text-red-500">{errors.image}</p>}
          </div>

          <div className="flex items-center gap-3">
            <label className="relative inline-flex cursor-pointer items-center">
              <input
                type="checkbox"
                checked={form.is_active}
                onChange={(e) => setForm((f) => ({ ...f, is_active: e.target.checked }))}
                className="peer sr-only"
              />
              <div className="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:start-[2px] after:top-0.5 after:h-5 after:w-5 after:rounded-full after:bg-white after:transition-all peer-checked:bg-primary peer-checked:after:translate-x-full peer-focus:ring-2 peer-focus:ring-primary/30" />
            </label>
            <span className="text-sm">サイトに表示する</span>
          </div>
        </div>
      </Modal>

      <ConfirmModal
        open={deleteTargetId !== null}
        title="取引先を削除"
        description="この取引先を削除しますか？この操作は取り消せません。"
        confirmLabel="削除する"
        confirmVariant="danger"
        isSubmitting={deleting}
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteTargetId(null)}
      />
    </div>
  );
}
