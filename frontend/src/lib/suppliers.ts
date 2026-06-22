import { authedFetch } from '@/lib/api';
import type { Supplier } from '@/types/supplier';

interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}

interface SupplierResponse {
  id: number;
  name: string;
  description: string;
  instagram_url?: string | null;
  image_url?: string | null;
  display_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface AdminSupplierListResponse {
  suppliers: SupplierResponse[];
}

interface SupplierImageResponse {
  image_url: string;
}

export class SupplierApiError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ErrorResponse['error']) {
    super(error.message);
    this.name = 'SupplierApiError';
    this.code = error.code;
    this.details = error.details;
  }
}

export const MAX_SUPPLIER_IMAGE_BYTES = 5 * 1024 * 1024;
export const ALLOWED_SUPPLIER_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const;

export function toSupplierErrorMessage(err: unknown): string {
  if (err instanceof SupplierApiError) return err.message;
  if (err instanceof Error) return err.message;
  return '取引先の操作に失敗しました';
}

export async function parseSupplierError(res: Response): Promise<SupplierApiError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new SupplierApiError(body.error);
    }
  } catch {
    // fall through
  }
  return new SupplierApiError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

/** API レスポンスをフロント共通型に正規化する */
export function normalizeSupplier(item: SupplierResponse): Supplier {
  return {
    id: item.id,
    name: item.name,
    description: item.description,
    instagram_url: item.instagram_url ?? null,
    image_url: item.image_url ?? null,
    display_order: item.display_order,
    is_active: item.is_active,
    created_at: item.created_at,
    updated_at: item.updated_at,
  };
}

export function validateSupplierImageFile(file: File): string | null {
  if (!ALLOWED_SUPPLIER_IMAGE_TYPES.includes(file.type as (typeof ALLOWED_SUPPLIER_IMAGE_TYPES)[number])) {
    return 'JPEG / PNG / WebP 形式の画像を選択してください';
  }
  if (file.size > MAX_SUPPLIER_IMAGE_BYTES) {
    return '画像サイズは5MB以内にしてください';
  }
  return null;
}

export interface CreateSupplierRequest {
  name: string;
  description: string;
  instagram_url?: string | null;
  image_url?: string | null;
  display_order?: number;
  is_active?: boolean;
}

export interface UpdateSupplierRequest {
  name?: string;
  description?: string;
  instagram_url?: string | null;
  image_url?: string | null;
  display_order?: number;
  is_active?: boolean;
}

/** GET /api/v1/admin/suppliers */
export async function listAdminSuppliers(): Promise<Supplier[]> {
  const res = await authedFetch('/api/v1/admin/suppliers');

  if (!res.ok) {
    throw await parseSupplierError(res);
  }

  const data = (await res.json()) as AdminSupplierListResponse;
  return data.suppliers.map(normalizeSupplier);
}

/** GET /api/v1/admin/suppliers/{id} */
export async function getSupplier(id: number): Promise<Supplier> {
  const res = await authedFetch(`/api/v1/admin/suppliers/${id}`);

  if (!res.ok) {
    throw await parseSupplierError(res);
  }

  const data = (await res.json()) as SupplierResponse;
  return normalizeSupplier(data);
}

/** POST /api/v1/admin/suppliers */
export async function createSupplier(req: CreateSupplierRequest): Promise<Supplier> {
  const res = await authedFetch('/api/v1/admin/suppliers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  if (!res.ok) {
    throw await parseSupplierError(res);
  }

  const data = (await res.json()) as SupplierResponse;
  return normalizeSupplier(data);
}

/** PUT /api/v1/admin/suppliers/{id} */
export async function updateSupplier(id: number, req: UpdateSupplierRequest): Promise<Supplier> {
  const res = await authedFetch(`/api/v1/admin/suppliers/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  if (!res.ok) {
    throw await parseSupplierError(res);
  }

  const data = (await res.json()) as SupplierResponse;
  return normalizeSupplier(data);
}

/** DELETE /api/v1/admin/suppliers/{id} */
export async function deleteSupplier(id: number): Promise<void> {
  const res = await authedFetch(`/api/v1/admin/suppliers/${id}`, {
    method: 'DELETE',
  });

  if (!res.ok) {
    throw await parseSupplierError(res);
  }
}

/** PUT /api/v1/admin/suppliers/order */
export async function reorderSuppliers(ids: number[]): Promise<Supplier[]> {
  const res = await authedFetch('/api/v1/admin/suppliers/order', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ order: ids }),
  });

  if (!res.ok) {
    throw await parseSupplierError(res);
  }

  const data = (await res.json()) as AdminSupplierListResponse;
  return data.suppliers.map(normalizeSupplier);
}

/** POST /api/v1/admin/suppliers/{id}/image */
export async function uploadSupplierImage(id: number, file: File): Promise<string> {
  const formData = new FormData();
  formData.append('image', file);

  const res = await authedFetch(`/api/v1/admin/suppliers/${id}/image`, {
    method: 'POST',
    body: formData,
  });

  if (!res.ok) {
    throw await parseSupplierError(res);
  }

  const data = (await res.json()) as SupplierImageResponse;
  return data.image_url;
}
